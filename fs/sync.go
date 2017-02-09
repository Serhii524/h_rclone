// Implementation of sync/copy/move

package fs

import (
	"fmt"
	"sync"
	"time"

	"github.com/pkg/errors"
)

var oldSyncMethod = BoolP("old-sync-method", "", false, "Temporary flag to select old sync method")

type syncCopyMove struct {
	// parameters
	fdst       Fs
	fsrc       Fs
	deleteMode DeleteMode // how we are doing deletions
	DoMove     bool
	dir        string
	// internal state
	noTraverse     bool                // if set don't trafevers the dst
	deletersWg     sync.WaitGroup      // for delete before go routine
	deleteFilesCh  chan Object         // channel to receive deletes if delete before
	trackRenames   bool                // set if we should do server side renames
	dstFilesMu     sync.Mutex          // protect dstFiles
	dstFiles       map[string]Object   // dst files, always filled
	srcFiles       map[string]Object   // src files, only used if deleteBefore
	srcFilesChan   chan Object         // passes src objects
	srcFilesResult chan error          // error result of src listing
	dstFilesResult chan error          // error result of dst listing
	abort          chan struct{}       // signal to abort the copiers
	checkerWg      sync.WaitGroup      // wait for checkers
	toBeChecked    ObjectPairChan      // checkers channel
	transfersWg    sync.WaitGroup      // wait for transfers
	toBeUploaded   ObjectPairChan      // copiers channel
	errorMu        sync.Mutex          // Mutex covering the errors variables
	err            error               // normal error from copy process
	noRetryErr     error               // error with NoRetry set
	fatalErr       error               // fatal error
	commonHash     HashType            // common hash type between src and dst
	renameMapMu    sync.Mutex          // mutex to protect the below
	renameMap      map[string][]Object // dst files by hash - only used by trackRenames
	renamerWg      sync.WaitGroup      // wait for renamers
	toBeRenamed    ObjectPairChan      // renamers channel
	trackRenamesWg sync.WaitGroup      // wg for background track renames
	trackRenamesCh chan Object         // objects are pumped in here
	renameCheck    []Object            // accumulate files to check for rename here
	backupDir      Fs                  // place to store overwrites/deletes
	suffix         string              // suffix to add to files placed in backupDir
}

func newSyncCopyMove(fdst, fsrc Fs, deleteMode DeleteMode, DoMove bool) (*syncCopyMove, error) {
	s := &syncCopyMove{
		fdst:           fdst,
		fsrc:           fsrc,
		deleteMode:     deleteMode,
		DoMove:         DoMove,
		dir:            "",
		srcFilesChan:   make(chan Object, Config.Checkers+Config.Transfers),
		srcFilesResult: make(chan error, 1),
		dstFilesResult: make(chan error, 1),
		noTraverse:     Config.NoTraverse,
		abort:          make(chan struct{}),
		toBeChecked:    make(ObjectPairChan, Config.Transfers),
		toBeUploaded:   make(ObjectPairChan, Config.Transfers),
		deleteFilesCh:  make(chan Object, Config.Checkers),
		trackRenames:   Config.TrackRenames,
		commonHash:     fsrc.Hashes().Overlap(fdst.Hashes()).GetOne(),
		toBeRenamed:    make(ObjectPairChan, Config.Transfers),
		trackRenamesCh: make(chan Object, Config.Checkers),
	}
	if s.noTraverse && s.deleteMode != DeleteModeOff {
		Errorf(nil, "Ignoring --no-traverse with sync")
		s.noTraverse = false
	}
	if s.trackRenames {
		// Don't track renames for remotes without server-side move support.
		if !CanServerSideMove(fdst) {
			Errorf(fdst, "Ignoring --track-renames as the destination does not support server-side move or copy")
			s.trackRenames = false
		}
		if s.commonHash == HashNone {
			Errorf(fdst, "Ignoring --track-renames as the source and destination do not have a common hash")
			s.trackRenames = false
		}
	}
	if s.trackRenames {
		// track renames needs delete after
		if s.deleteMode != DeleteModeOff {
			s.deleteMode = DeleteModeAfter
		}
		if s.noTraverse {
			Errorf(nil, "Ignoring --no-traverse with --track-renames")
			s.noTraverse = false
		}
	}
	// Make Fs for --backup-dir if required
	if Config.BackupDir != "" {
		var err error
		s.backupDir, err = NewFs(Config.BackupDir)
		if err != nil {
			return nil, FatalError(errors.Errorf("Failed to make fs for --backup-dir %q: %v", Config.BackupDir, err))
		}
		if !CanServerSideMove(s.backupDir) {
			return nil, FatalError(errors.New("can't use --backup-dir on a remote which doesn't support server side move or copy"))
		}
		if !SameConfig(fdst, s.backupDir) {
			return nil, FatalError(errors.New("parameter to --backup-dir has to be on the same remote as destination"))
		}
		if Overlapping(fdst, s.backupDir) {
			return nil, FatalError(errors.New("destination and parameter to --backup-dir mustn't overlap"))
		}
		if Overlapping(fsrc, s.backupDir) {
			return nil, FatalError(errors.New("source and parameter to --backup-dir mustn't overlap"))
		}
		s.suffix = Config.Suffix
	}
	return s, nil
}

// Check to see if have set the abort flag
func (s *syncCopyMove) aborting() bool {
	select {
	case <-s.abort:
		return true
	default:
	}
	return false
}

// This reads the map and pumps it into the channel passed in, closing
// the channel at the end
func (s *syncCopyMove) pumpMapToChan(files map[string]Object, out chan<- Object) {
outer:
	for _, o := range files {
		if s.aborting() {
			break outer
		}
		select {
		case out <- o:
		case <-s.abort:
			break outer
		}
	}
	close(out)
	s.srcFilesResult <- nil
}

// This reads the source files from s.srcFiles into srcFilesChan then
// closes it
//
// It returns the final result of the read into s.srcFilesResult
func (s *syncCopyMove) readSrcUsingMap() {
	s.pumpMapToChan(s.srcFiles, s.srcFilesChan)
	s.srcFilesResult <- nil
}

// This reads the source files into srcFilesChan then closes it
//
// It returns the final result of the read into s.srcFilesResult
func (s *syncCopyMove) readSrcUsingChan() {
	err := readFilesFn(s.fsrc, false, s.dir, func(o Object) error {
		if s.aborting() {
			return ErrorListAborted
		}
		select {
		case s.srcFilesChan <- o:
		case <-s.abort:
			return ErrorListAborted
		}
		return nil
	})
	close(s.srcFilesChan)
	if err != nil {
		err = errors.Wrapf(err, "error listing source: %s", s.fsrc)
	}
	s.srcFilesResult <- err
}

// This reads the destination files in into dstFiles
//
// It returns the final result of the read into s.dstFilesResult
func (s *syncCopyMove) readDstFiles() {
	var err error
	s.dstFiles, err = readFilesMap(s.fdst, Config.Filter.DeleteExcluded, s.dir)
	s.dstFilesResult <- err
}

// NeedTransfer checks to see if src needs to be copied to dst using
// the current config.
//
// Returns a flag which indicates whether the file needs to be
// transferred or not.
func NeedTransfer(dst, src Object) bool {
	if dst == nil {
		Debugf(src, "Couldn't find file - need to transfer")
		return true
	}
	// If we should ignore existing files, don't transfer
	if Config.IgnoreExisting {
		Debugf(src, "Destination exists, skipping")
		return false
	}
	// If we should upload unconditionally
	if Config.IgnoreTimes {
		Debugf(src, "Transferring unconditionally as --ignore-times is in use")
		return true
	}
	// If UpdateOlder is in effect, skip if dst is newer than src
	if Config.UpdateOlder {
		srcModTime := src.ModTime()
		dstModTime := dst.ModTime()
		dt := dstModTime.Sub(srcModTime)
		// If have a mutually agreed precision then use that
		modifyWindow := Config.ModifyWindow
		if modifyWindow == ModTimeNotSupported {
			// Otherwise use 1 second as a safe default as
			// the resolution of the time a file was
			// uploaded.
			modifyWindow = time.Second
		}
		switch {
		case dt >= modifyWindow:
			Debugf(src, "Destination is newer than source, skipping")
			return false
		case dt <= -modifyWindow:
			Debugf(src, "Destination is older than source, transferring")
		default:
			if src.Size() == dst.Size() {
				Debugf(src, "Destination mod time is within %v of source and sizes identical, skipping", modifyWindow)
				return false
			}
			Debugf(src, "Destination mod time is within %v of source but sizes differ, transferring", modifyWindow)
		}
	} else {
		// Check to see if changed or not
		if Equal(src, dst) {
			Debugf(src, "Unchanged skipping")
			return false
		}
	}
	return true
}

// This checks the types of errors returned while copying files
func (s *syncCopyMove) processError(err error) {
	if err == nil {
		return
	}
	s.errorMu.Lock()
	defer s.errorMu.Unlock()
	switch {
	case IsFatalError(err):
		if !s.aborting() {
			close(s.abort)
		}
		s.fatalErr = err
	case IsNoRetryError(err):
		s.noRetryErr = err
	default:
		s.err = err
	}
}

// Returns the current error (if any) in the order of prececedence
//   fatalErr
//   normal error
//   noRetryErr
func (s *syncCopyMove) currentError() error {
	s.errorMu.Lock()
	defer s.errorMu.Unlock()
	if s.fatalErr != nil {
		return s.fatalErr
	}
	if s.err != nil {
		return s.err
	}
	return s.noRetryErr
}

// pairChecker reads Objects~s on in send to out if they need transferring.
//
// FIXME potentially doing lots of hashes at once
func (s *syncCopyMove) pairChecker(in ObjectPairChan, out ObjectPairChan, wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		if s.aborting() {
			return
		}
		select {
		case pair, ok := <-in:
			if !ok {
				return
			}
			src := pair.src
			Stats.Checking(src.Remote())
			// Check to see if can store this
			if src.Storable() {
				if NeedTransfer(pair.dst, pair.src) {
					// If destination already exists, then we must move it into --backup-dir if required
					if pair.dst != nil && s.backupDir != nil {
						remoteWithSuffix := pair.dst.Remote() + s.suffix
						overwritten, _ := s.backupDir.NewObject(remoteWithSuffix)
						err := Move(s.backupDir, overwritten, remoteWithSuffix, pair.dst)
						if err != nil {
							s.processError(err)
						} else {
							// If successful zero out the dst as it is no longer there and copy the file
							pair.dst = nil
							out <- pair
						}
					} else {
						out <- pair
					}
				} else {
					// If moving need to delete the files we don't need to copy
					if s.DoMove {
						// Delete src if no error on copy
						s.processError(DeleteFile(src))
					}
				}
			}
			Stats.DoneChecking(src.Remote())
		case <-s.abort:
			return
		}
	}
}

// pairRenamer reads Objects~s on in and attempts to rename them,
// otherwise it sends them out if they need transferring.
func (s *syncCopyMove) pairRenamer(in ObjectPairChan, out ObjectPairChan, wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		if s.aborting() {
			return
		}
		select {
		case pair, ok := <-in:
			if !ok {
				return
			}
			src := pair.src
			if !s.tryRename(src) {
				// pass on if not renamed
				out <- pair
			}
		case <-s.abort:
			return
		}
	}
}

// pairCopyOrMove reads Objects on in and moves or copies them.
func (s *syncCopyMove) pairCopyOrMove(in ObjectPairChan, fdst Fs, wg *sync.WaitGroup) {
	defer wg.Done()
	var err error
	for {
		if s.aborting() {
			return
		}
		select {
		case pair, ok := <-in:
			if !ok {
				return
			}
			src := pair.src
			Stats.Transferring(src.Remote())
			if s.DoMove {
				err = Move(fdst, pair.dst, src.Remote(), src)
			} else {
				err = Copy(fdst, pair.dst, src.Remote(), src)
			}
			s.processError(err)
			Stats.DoneTransferring(src.Remote(), err == nil)
		case <-s.abort:
			return
		}
	}
}

// This starts the background checkers.
func (s *syncCopyMove) startCheckers() {
	s.checkerWg.Add(Config.Checkers)
	for i := 0; i < Config.Checkers; i++ {
		go s.pairChecker(s.toBeChecked, s.toBeUploaded, &s.checkerWg)
	}
}

// This stops the background checkers
func (s *syncCopyMove) stopCheckers() {
	close(s.toBeChecked)
	Logf(s.fdst, "Waiting for checks to finish")
	s.checkerWg.Wait()
}

// This starts the background transfers
func (s *syncCopyMove) startTransfers() {
	s.transfersWg.Add(Config.Transfers)
	for i := 0; i < Config.Transfers; i++ {
		go s.pairCopyOrMove(s.toBeUploaded, s.fdst, &s.transfersWg)
	}
}

// This stops the background transfers
func (s *syncCopyMove) stopTransfers() {
	close(s.toBeUploaded)
	Logf(s.fdst, "Waiting for transfers to finish")
	s.transfersWg.Wait()
}

// This starts the background renamers.
func (s *syncCopyMove) startRenamers() {
	if !s.trackRenames {
		return
	}
	s.renamerWg.Add(Config.Checkers)
	for i := 0; i < Config.Checkers; i++ {
		go s.pairRenamer(s.toBeRenamed, s.toBeUploaded, &s.renamerWg)
	}
}

// This stops the background renamers
func (s *syncCopyMove) stopRenamers() {
	if !s.trackRenames {
		return
	}
	close(s.toBeRenamed)
	Logf(s.fdst, "Waiting for renames to finish")
	s.renamerWg.Wait()
}

// This starts the collection of possible renames
func (s *syncCopyMove) startTrackRenames() {
	if !s.trackRenames {
		return
	}
	s.trackRenamesWg.Add(1)
	go func() {
		defer s.trackRenamesWg.Done()
		for o := range s.trackRenamesCh {
			s.renameCheck = append(s.renameCheck, o)
		}
	}()
}

// This stops the background rename collection
func (s *syncCopyMove) stopTrackRenames() {
	if !s.trackRenames {
		return
	}
	close(s.trackRenamesCh)
	s.trackRenamesWg.Wait()
}

// This starts the background deletion of files for --delete-during
func (s *syncCopyMove) startDeleters() {
	if s.deleteMode != DeleteModeDuring && s.deleteMode != DeleteModeOnly {
		return
	}
	s.deletersWg.Add(1)
	go func() {
		defer s.deletersWg.Done()
		err := deleteFilesWithBackupDir(s.deleteFilesCh, s.backupDir)
		s.processError(err)
	}()
}

// This stops the background deleters
func (s *syncCopyMove) stopDeleters() {
	if s.deleteMode != DeleteModeDuring && s.deleteMode != DeleteModeOnly {
		return
	}
	close(s.deleteFilesCh)
	s.deletersWg.Wait()
}

// This deletes the files in the dstFiles map.  If checkSrcMap is set
// then it checks to see if they exist first in srcFiles the source
// file map, otherwise it unconditionally deletes them.  If
// checkSrcMap is clear then it assumes that the any source files that
// have been found have been removed from dstFiles already.
func (s *syncCopyMove) deleteFiles(checkSrcMap bool) error {
	if Stats.Errored() {
		Errorf(s.fdst, "%v", ErrorNotDeleting)
		return ErrorNotDeleting
	}

	// Delete the spare files
	toDelete := make(ObjectsChan, Config.Transfers)
	go func() {
		for remote, o := range s.dstFiles {
			if checkSrcMap {
				_, exists := s.srcFiles[remote]
				if exists {
					continue
				}
			}
			if s.aborting() {
				break
			}
			toDelete <- o
		}
		close(toDelete)
	}()
	return deleteFilesWithBackupDir(toDelete, s.backupDir)
}

// renameHash makes a string with the size and the hash for rename detection
//
// it may return an empty string in which case no hash could be made
func (s *syncCopyMove) renameHash(obj Object) (hash string) {
	var err error
	hash, err = obj.Hash(s.commonHash)
	if err != nil {
		Debugf(obj, "Hash failed: %v", err)
		return ""
	}
	if hash == "" {
		return ""
	}
	return fmt.Sprintf("%d,%s", obj.Size(), hash)
}

// pushRenameMap adds the object with hash to the rename map
func (s *syncCopyMove) pushRenameMap(hash string, obj Object) {
	s.renameMapMu.Lock()
	s.renameMap[hash] = append(s.renameMap[hash], obj)
	s.renameMapMu.Unlock()
}

// popRenameMap finds the object with hash and pop the first match from
// renameMap or returns nil if not found.
func (s *syncCopyMove) popRenameMap(hash string) (dst Object) {
	s.renameMapMu.Lock()
	dsts, ok := s.renameMap[hash]
	if ok && len(dsts) > 0 {
		dst, dsts = dsts[0], dsts[1:]
		if len(dsts) > 0 {
			s.renameMap[hash] = dsts
		} else {
			delete(s.renameMap, hash)
		}
	}
	s.renameMapMu.Unlock()
	return dst
}

// makeRenameMap builds a map of the destination files by hash that
// match sizes in the slice of objects in s.renameCheck
func (s *syncCopyMove) makeRenameMap() {
	Debugf(s.fdst, "Making map for --track-renames")

	// first make a map of possible sizes we need to check
	possibleSizes := map[int64]struct{}{}
	for _, obj := range s.renameCheck {
		possibleSizes[obj.Size()] = struct{}{}
	}

	// pump all the dstFiles into in
	in := make(chan Object, Config.Checkers)
	go s.pumpMapToChan(s.dstFiles, in)

	// now make a map of size,hash for all dstFiles
	s.renameMap = make(map[string][]Object)
	var wg sync.WaitGroup
	wg.Add(Config.Transfers)
	for i := 0; i < Config.Transfers; i++ {
		go func() {
			defer wg.Done()
			for obj := range in {
				// only create hash for dst Object if its size could match
				if _, found := possibleSizes[obj.Size()]; found {
					Stats.Checking(obj.Remote())
					hash := s.renameHash(obj)
					if hash != "" {
						s.pushRenameMap(hash, obj)
					}
					Stats.DoneChecking(obj.Remote())
				}
			}
		}()
	}
	wg.Wait()
	Debugf(s.fdst, "Finished making map for --track-renames")
}

// tryRename renames a src object when doing track renames if
// possible, it returns true if the object was renamed.
func (s *syncCopyMove) tryRename(src Object) bool {
	Stats.Checking(src.Remote())
	defer Stats.DoneChecking(src.Remote())

	hash := s.renameHash(src)
	if hash == "" {
		return false
	}

	dst := s.popRenameMap(hash)
	if dst == nil {
		return false
	}

	err := MoveFile(s.fdst, s.fdst, src.Remote(), dst.Remote())
	if err != nil {
		Debugf(src, "Failed to rename to %q: %v", dst.Remote(), err)
		return false
	}

	// remove file from dstFiles if present
	s.dstFilesMu.Lock()
	delete(s.dstFiles, dst.Remote())
	s.dstFilesMu.Unlock()

	Debugf(src, "Renamed from %q", dst.Remote())
	return true
}

// Syncs fsrc into fdst
//
// If Delete is true then it deletes any files in fdst that aren't in fsrc
//
// If DoMove is true then files will be moved instead of copied
//
// dir is the start directory, "" for root
func (s *syncCopyMove) runRecursive() error {
	if Same(s.fdst, s.fsrc) {
		Errorf(s.fdst, "Nothing to do as source and destination are the same")
		return nil
	}

	err := Mkdir(s.fdst, "")
	if err != nil {
		return err
	}

	// Start reading dstFiles if required
	if !s.noTraverse {
		go s.readDstFiles()
	}

	// If --delete-before then we need to read the whole source map first
	readSourceMap := s.deleteMode == DeleteModeBefore

	if readSourceMap {
		// Read source files into the map
		s.srcFiles, err = readFilesMap(s.fsrc, false, s.dir)
		if err != nil {
			return err
		}

	}

	// Wait for dstfiles to finish reading if we were reading them
	// and report any errors
	if !s.noTraverse {
		err = <-s.dstFilesResult
		if err != nil {
			return err
		}
	}

	// Delete files first if required
	if s.deleteMode == DeleteModeBefore {
		err = s.deleteFiles(true)
		if err != nil {
			return err
		}
	}

	// Now we can fill the src channel.
	if readSourceMap {
		// Pump the map into s.srcFilesChan
		go s.readSrcUsingMap()
	} else {
		go s.readSrcUsingChan()
	}

	// Start background checking and transferring pipeline
	s.startCheckers()
	s.startRenamers()
	s.startTransfers()

	// Do the transfers
	s.startTrackRenames()
	for src := range s.srcFilesChan {
		remote := src.Remote()
		var dst Object
		if s.noTraverse {
			var err error
			dst, err = s.fdst.NewObject(remote)
			if err != nil {
				dst = nil
				if err != ErrorObjectNotFound {
					Debugf(src, "Error making NewObject: %v", err)
				}
			}
		} else {
			s.dstFilesMu.Lock()
			var ok bool
			dst, ok = s.dstFiles[remote]
			if ok {
				// Remove file from s.dstFiles because it exists in srcFiles
				delete(s.dstFiles, remote)
			}
			s.dstFilesMu.Unlock()
		}
		if dst != nil {
			s.toBeChecked <- ObjectPair{src, dst}
		} else if s.trackRenames {
			// Save object to check for a rename later
			s.trackRenamesCh <- src
		} else {
			// No need to check since doesn't exist
			s.toBeUploaded <- ObjectPair{src, nil}
		}
	}

	s.stopTrackRenames()
	if s.trackRenames {
		// Build the map of the remaining dstFiles by hash
		s.makeRenameMap()
		// Attempt renames for all the files which don't have a matching dst
		for _, src := range s.renameCheck {
			s.toBeRenamed <- ObjectPair{src, nil}
		}
	}

	// Stop background checking and transferring pipeline
	s.stopCheckers()
	s.stopRenamers()
	s.stopTransfers()

	// Retrieve the delayed error from the source listing goroutine
	err = <-s.srcFilesResult

	// Delete files during or after
	if s.deleteMode == DeleteModeDuring || s.deleteMode == DeleteModeAfter {
		if err != nil {
			Errorf(s.fdst, "%v", ErrorNotDeleting)
		} else {
			err = s.deleteFiles(false)
		}
	}
	s.processError(err)
	return s.currentError()
}

// listDirJob describe a directory listing that needs to be done
type listDirJob struct {
	remote   string
	srcDepth int
	dstDepth int
	noSrc    bool
	noDst    bool
}

// Syncs fsrc into fdst
//
// If Delete is true then it deletes any files in fdst that aren't in fsrc
//
// If DoMove is true then files will be moved instead of copied
//
// dir is the start directory, "" for root
func (s *syncCopyMove) runDirAtATime() error {
	srcDepth := Config.MaxDepth
	if srcDepth < 0 {
		srcDepth = MaxLevel
	}
	dstDepth := srcDepth
	if Config.Filter.DeleteExcluded {
		dstDepth = MaxLevel
	}

	if Same(s.fdst, s.fsrc) {
		Errorf(s.fdst, "Nothing to do as source and destination are the same")
		return nil
	}

	// Make the destination directory
	err := Mkdir(s.fdst, s.dir)
	if err != nil {
		return err
	}

	// Start background checking and transferring pipeline
	s.startCheckers()
	s.startRenamers()
	s.startTransfers()
	s.startDeleters()
	s.dstFiles = make(map[string]Object)

	// Start some directory listing go routines
	var wg sync.WaitGroup         // sync closing of go routines
	var traversing sync.WaitGroup // running directory traversals
	in := make(chan listDirJob, Config.Checkers)
	s.startTrackRenames()
	for i := 0; i < Config.Checkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				if s.aborting() {
					return
				}
				select {
				case job, ok := <-in:
					if !ok {
						return
					}
					jobs := s._runDirAtATime(job)
					if len(jobs) > 0 {
						traversing.Add(len(jobs))
						go func() {
							// Now we have traversed this directory, send these
							// jobs off for traversal in the background
							for _, newJob := range jobs {
								in <- newJob
							}
						}()
					}
					traversing.Done()
				case <-s.abort:
					return
				}
			}
		}()
	}

	// Start the process
	traversing.Add(1)
	in <- listDirJob{
		remote:   s.dir,
		srcDepth: srcDepth - 1,
		dstDepth: dstDepth - 1,
	}
	traversing.Wait()
	close(in)
	wg.Wait()

	s.stopTrackRenames()
	if s.trackRenames {
		// Build the map of the remaining dstFiles by hash
		s.makeRenameMap()
		// Attempt renames for all the files which don't have a matching dst
		for _, src := range s.renameCheck {
			s.toBeRenamed <- ObjectPair{src, nil}
		}
	}

	// Stop background checking and transferring pipeline
	s.stopCheckers()
	s.stopRenamers()
	s.stopTransfers()
	s.stopDeleters()

	// Delete files after
	if s.deleteMode == DeleteModeAfter {
		if s.currentError() != nil {
			Errorf(s.fdst, "%v", ErrorNotDeleting)
		} else {
			s.processError(s.deleteFiles(false))
		}
	}
	return s.currentError()
}

// Have an object which is in the destination only
func (s *syncCopyMove) dstOnly(dst BasicInfo, job listDirJob, jobs *[]listDirJob) {
	if s.deleteMode == DeleteModeOff {
		return
	}
	switch x := dst.(type) {
	case Object:
		switch s.deleteMode {
		case DeleteModeAfter:
			// record object as needs deleting
			s.dstFilesMu.Lock()
			s.dstFiles[x.Remote()] = x
			s.dstFilesMu.Unlock()
		case DeleteModeDuring, DeleteModeOnly:
			s.deleteFilesCh <- x
		default:
			panic(fmt.Sprintf("unexpected delete mode %d", s.deleteMode))
		}
	case *Dir:
		// Do the same thing to the entire contents of the directory
		if job.dstDepth > 0 {
			*jobs = append(*jobs, listDirJob{
				remote:   dst.Remote(),
				dstDepth: job.dstDepth - 1,
				noSrc:    true,
			})
		}
	default:
		panic("Bad object in DirEntries")

	}
}

// Have an object which is in the source only
func (s *syncCopyMove) srcOnly(src BasicInfo, job listDirJob, jobs *[]listDirJob) {
	if s.deleteMode == DeleteModeOnly {
		return
	}
	switch x := src.(type) {
	case Object:
		if s.trackRenames {
			// Save object to check for a rename later
			s.trackRenamesCh <- x
		} else {
			// No need to check since doesn't exist
			s.toBeUploaded <- ObjectPair{x, nil}
		}
	case *Dir:
		// Do the same thing to the entire contents of the directory
		if job.srcDepth > 0 {
			*jobs = append(*jobs, listDirJob{
				remote:   src.Remote(),
				srcDepth: job.srcDepth - 1,
				noDst:    true,
			})
		}
	default:
		panic("Bad object in DirEntries")
	}
}

// Given a src and a dst, transfer the src to dst
func (s *syncCopyMove) transfer(dst, src BasicInfo, job listDirJob, jobs *[]listDirJob) {
	switch srcX := src.(type) {
	case Object:
		if s.deleteMode == DeleteModeOnly {
			return
		}
		dstX, ok := dst.(Object)
		if ok {
			s.toBeChecked <- ObjectPair{srcX, dstX}
		} else {
			// FIXME src is file, dst is directory
			err := errors.New("can't overwrite directory with file")
			Errorf(srcX, "%v", err)
			s.processError(err)
		}
	case *Dir:
		// Do the same thing to the entire contents of the directory
		dstX, ok := dst.(*Dir)
		if ok {
			if job.srcDepth > 0 && job.dstDepth > 0 {
				*jobs = append(*jobs, listDirJob{
					remote:   src.Remote(),
					srcDepth: job.srcDepth - 1,
					dstDepth: job.dstDepth - 1,
				})
			}
		} else {
			// FIXME src is dir, dst is file
			err := errors.New("can't overwrite file with directory")
			Errorf(dstX, "%v", err)
			s.processError(err)
		}
	default:
		panic("Bad object in DirEntries")
	}
}

// returns errors using processError
func (s *syncCopyMove) _runDirAtATime(job listDirJob) (jobs []listDirJob) {
	var (
		srcList, dstList       DirEntries
		srcListErr, dstListErr error
		wg                     sync.WaitGroup
	)

	// List the src and dst directories
	if !job.noSrc {
		wg.Add(1)
		go func() {
			defer wg.Done()
			srcList, srcListErr = ListDirSorted(s.fsrc, false, job.remote)
		}()
	}
	if !job.noDst {
		wg.Add(1)
		go func() {
			defer wg.Done()
			dstList, dstListErr = ListDirSorted(s.fdst, Config.Filter.DeleteExcluded, job.remote)
		}()
	}

	// Wait for listings to complete and report errors
	wg.Wait()
	if srcListErr != nil {
		s.processError(errors.Wrapf(srcListErr, "error reading source directory %q", job.remote))
		return nil
	}
	if dstListErr == ErrorDirNotFound {
		// Copy the stuff anyway
	} else if dstListErr != nil {
		s.processError(errors.Wrapf(srcListErr, "error reading destination directory %q", job.remote))
		return nil
	}

	// Process the two listings, matching up the items in the two sorted slices
	for iSrc, iDst := 0, 0; ; iSrc, iDst = iSrc+1, iDst+1 {
		if s.aborting() {
			return nil
		}
		var src, dst BasicInfo
		if iSrc < len(srcList) {
			src = srcList[iSrc]
		}
		if iDst < len(dstList) {
			dst = dstList[iDst]
		}
		if src == nil && dst == nil {
			break
		}
		if src != nil && dst != nil {
			if src.Remote() < dst.Remote() {
				dst = nil
				iDst-- // retry the dst
			} else if src.Remote() > dst.Remote() {
				src = nil
				iSrc-- // retry the src
			}
		}
		// Debugf(nil, "src = %v, dst = %v", src, dst)
		switch {
		case src == nil:
			s.dstOnly(dst, job, &jobs)
		case dst == nil:
			s.srcOnly(src, job, &jobs)
		default:
			s.transfer(dst, src, job, &jobs)
		}
	}
	return jobs
}

// Syncs fsrc into fdst
//
// If Delete is true then it deletes any files in fdst that aren't in fsrc
//
// If DoMove is true then files will be moved instead of copied
//
// dir is the start directory, "" for root
func runSyncCopyMove(fdst, fsrc Fs, deleteMode DeleteMode, DoMove bool) error {
	if deleteMode != DeleteModeOff && DoMove {
		return errors.New("can't delete and move at the same time")
	}
	// Run an extra pass to delete only
	if !*oldSyncMethod && Config.DeleteMode == DeleteModeBefore {
		if Config.TrackRenames {
			return errors.New("can't use --delete-before with --track-renames")
		}
		// only delete stuff during in this pass
		do, err := newSyncCopyMove(fdst, fsrc, DeleteModeOnly, false)
		if err != nil {
			return err
		}
		err = do.runDirAtATime()
		if err != nil {
			return err
		}
		// Next pass does a copy only
		deleteMode = DeleteModeOff
	}
	do, err := newSyncCopyMove(fdst, fsrc, deleteMode, DoMove)
	if err != nil {
		return err
	}
	if *oldSyncMethod {
		return do.runRecursive()
	}
	return do.runDirAtATime()
}

// Sync fsrc into fdst
func Sync(fdst, fsrc Fs) error {
	return runSyncCopyMove(fdst, fsrc, Config.DeleteMode, false)
}

// CopyDir copies fsrc into fdst
func CopyDir(fdst, fsrc Fs) error {
	return runSyncCopyMove(fdst, fsrc, DeleteModeOff, false)
}

// moveDir moves fsrc into fdst
func moveDir(fdst, fsrc Fs) error {
	return runSyncCopyMove(fdst, fsrc, DeleteModeOff, true)
}

// MoveDir moves fsrc into fdst
func MoveDir(fdst, fsrc Fs) error {
	if Same(fdst, fsrc) {
		Errorf(fdst, "Nothing to do as source and destination are the same")
		return nil
	}

	// First attempt to use DirMover if exists, same Fs and no filters are active
	if fdstDirMove := fdst.Features().DirMove; fdstDirMove != nil && SameConfig(fsrc, fdst) && Config.Filter.InActive() {
		if Config.DryRun {
			Logf(fdst, "Not doing server side directory move as --dry-run")
			return nil
		}
		Debugf(fdst, "Using server side directory move")
		err := fdstDirMove(fsrc)
		switch err {
		case ErrorCantDirMove, ErrorDirExists:
			Debugf(fdst, "Server side directory move failed - fallback to file moves: %v", err)
		case nil:
			Debugf(fdst, "Server side directory move succeeded")
			return nil
		default:
			Stats.Error()
			Errorf(fdst, "Server side directory move failed: %v", err)
			return err
		}
	}

	// The two remotes mustn't overlap if we didn't do server side move
	if Overlapping(fdst, fsrc) {
		err := ErrorCantMoveOverlapping
		Errorf(fdst, "%v", err)
		return err
	}

	// Otherwise move the files one by one
	return moveDir(fdst, fsrc)
}

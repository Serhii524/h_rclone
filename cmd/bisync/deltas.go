// Package bisync implements bisync
// Copyright (c) 2017-2020 Chris Nelson
package bisync

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/rclone/rclone/cmd/bisync/bilib"
	"github.com/rclone/rclone/fs"
	"github.com/rclone/rclone/fs/filter"
	"github.com/rclone/rclone/fs/operations"
	"golang.org/x/text/unicode/norm"
)

// delta
type delta uint8

const (
	deltaZero delta = 0
	deltaNew  delta = 1 << iota
	deltaNewer
	deltaOlder
	deltaSize
	deltaHash
	deltaDeleted
)

const (
	deltaModified delta = deltaNewer | deltaOlder | deltaSize | deltaHash | deltaDeleted
	deltaOther    delta = deltaNew | deltaNewer | deltaOlder
)

func (d delta) is(cond delta) bool {
	return d&cond != 0
}

// deltaSet
type deltaSet struct {
	deltas     map[string]delta
	opt        *Options
	fs         fs.Fs  // base filesystem
	msg        string // filesystem name for logging
	oldCount   int    // original number of files (for "excess deletes" check)
	deleted    int    // number of deleted files (for "excess deletes" check)
	foundSame  bool   // true if found at least one unchanged file
	checkFiles bilib.Names
}

func (ds *deltaSet) empty() bool {
	return len(ds.deltas) == 0
}

func (ds *deltaSet) sort() (sorted []string) {
	if ds.empty() {
		return
	}
	sorted = make([]string, 0, len(ds.deltas))
	for file := range ds.deltas {
		sorted = append(sorted, file)
	}
	sort.Strings(sorted)
	return
}

func (ds *deltaSet) printStats() {
	if ds.empty() {
		return
	}
	nAll := len(ds.deltas)
	nNew := 0
	nNewer := 0
	nOlder := 0
	nDeleted := 0
	for _, d := range ds.deltas {
		if d.is(deltaNew) {
			nNew++
		}
		if d.is(deltaNewer) {
			nNewer++
		}
		if d.is(deltaOlder) {
			nOlder++
		}
		if d.is(deltaDeleted) {
			nDeleted++
		}
	}
	fs.Infof(nil, "%s: %4d changes: %4d new, %4d newer, %4d older, %4d deleted",
		ds.msg, nAll, nNew, nNewer, nOlder, nDeleted)
}

// findDeltas
func (b *bisyncRun) findDeltas(fctx context.Context, f fs.Fs, oldListing string, now *fileList, msg string) (ds *deltaSet, err error) {
	var old *fileList
	newListing := oldListing + "-new"

	old, err = b.loadListing(oldListing)
	if err != nil {
		fs.Errorf(nil, "Failed loading prior %s listing: %s", msg, oldListing)
		b.abort = true
		return
	}
	if err = b.checkListing(old, oldListing, "prior "+msg); err != nil {
		return
	}

	if err == nil {
		err = b.checkListing(now, newListing, "current "+msg)
	}
	if err != nil {
		return
	}

	ds = &deltaSet{
		deltas:     map[string]delta{},
		fs:         f,
		msg:        msg,
		oldCount:   len(old.list),
		opt:        b.opt,
		checkFiles: bilib.Names{},
	}

	for _, file := range old.list {
		d := deltaZero
		if !now.has(file) {
			b.indent(msg, file, "File was deleted")
			ds.deleted++
			d |= deltaDeleted
		} else {
			// skip dirs here, as we only care if they are new/deleted, not newer/older
			if !now.isDir(file) {
				if old.getTime(file) != now.getTime(file) {
					if old.beforeOther(now, file) {
						fs.Debugf(file, "(old: %v current: %v", old.getTime(file), now.getTime(file))
						b.indent(msg, file, "File is newer")
						d |= deltaNewer
					} else { // Current version is older than prior sync.
						fs.Debugf(file, "(old: %v current: %v", old.getTime(file), now.getTime(file))
						b.indent(msg, file, "File is OLDER")
						d |= deltaOlder
					}
				}
				// TODO Compare sizes and hashes
			}
		}

		if d.is(deltaModified) {
			ds.deltas[file] = d
		} else {
			// Once we've found at least one unchanged file,
			// we know that not everything has changed,
			// as with a DST time change
			ds.foundSame = true
		}
	}

	for _, file := range now.list {
		if !old.has(file) {
			b.indent(msg, file, "File is new")
			ds.deltas[file] = deltaNew
		}
	}

	if b.opt.CheckAccess {
		// checkFiles is a small structure compared with the `now`, so we
		// return it alone and let the full delta map be garbage collected.
		for _, file := range now.list {
			if filepath.Base(file) == b.opt.CheckFilename {
				ds.checkFiles.Add(file)
			}
		}
	}

	return
}

// applyDeltas
func (b *bisyncRun) applyDeltas(ctx context.Context, ds1, ds2 *deltaSet) (changes1, changes2 bool, results2to1, results1to2 []Results, queues queues, err error) {
	path1 := bilib.FsPath(b.fs1)
	path2 := bilib.FsPath(b.fs2)

	copy1to2 := bilib.Names{}
	copy2to1 := bilib.Names{}
	delete1 := bilib.Names{}
	delete2 := bilib.Names{}
	handled := bilib.Names{}
	renamed1 := bilib.Names{}
	renamed2 := bilib.Names{}
	renameSkipped := bilib.Names{}
	deletedonboth := bilib.Names{}
	skippedDirs1 := newFileList()
	skippedDirs2 := newFileList()

	ctxMove := b.opt.setDryRun(ctx)

	// update AliasMap for deleted files, as march does not know about them
	b.updateAliases(ctx, ds1, ds2)

	// efficient isDir check
	// we load the listing just once and store only the dirs
	dirs1, dirs1Err := b.listDirsOnly(1)
	if dirs1Err != nil {
		b.critical = true
		b.retryable = true
		fs.Debugf(nil, "Error generating dirsonly list for path1: %v", dirs1Err)
		return
	}

	dirs2, dirs2Err := b.listDirsOnly(2)
	if dirs2Err != nil {
		b.critical = true
		b.retryable = true
		fs.Debugf(nil, "Error generating dirsonly list for path2: %v", dirs2Err)
		return
	}

	// build a list of only the "deltaOther"s so we don't have to check more files than necessary
	// this is essentially the same as running rclone check with a --files-from filter, then exempting the --match results from being renamed
	// we therefore avoid having to list the same directory more than once.

	// we are intentionally overriding DryRun here because we need to perform the check, even during a dry run, or the results would be inaccurate.
	// check is a read-only operation by its nature, so it's already "dry" in that sense.
	ctxNew, ciCheck := fs.AddConfig(ctx)
	ciCheck.DryRun = false

	ctxCheck, filterCheck := filter.AddConfig(ctxNew)

	for _, file := range ds1.sort() {
		alias := b.aliases.Alias(file)
		d1 := ds1.deltas[file]
		if d1.is(deltaOther) {
			d2, in2 := ds2.deltas[file]
			if !in2 && file != alias {
				d2 = ds2.deltas[alias]
			}
			if d2.is(deltaOther) {
				checkit := func(filename string) {
					if err := filterCheck.AddFile(filename); err != nil {
						fs.Debugf(nil, "Non-critical error adding file to list of potential conflicts to check: %s", err)
					} else {
						fs.Debugf(nil, "Added file to list of potential conflicts to check: %s", filename)
					}
				}
				checkit(file)
				if file != alias {
					checkit(alias)
				}
			}
		}
	}

	//if there are potential conflicts to check, check them all here (outside the loop) in one fell swoop
	matches, err := b.checkconflicts(ctxCheck, filterCheck, b.fs1, b.fs2)

	for _, file := range ds1.sort() {
		alias := b.aliases.Alias(file)
		p1 := path1 + file
		p2 := path2 + alias
		d1 := ds1.deltas[file]

		if d1.is(deltaOther) {
			d2, in2 := ds2.deltas[file]
			// try looking under alternate name
			if !in2 && file != alias {
				d2, in2 = ds2.deltas[alias]
			}
			if !in2 {
				b.indent("Path1", p2, "Queue copy to Path2")
				copy1to2.Add(file)
			} else if d2.is(deltaDeleted) {
				b.indent("Path1", p2, "Queue copy to Path2")
				copy1to2.Add(file)
				handled.Add(file)
			} else if d2.is(deltaOther) {
				b.indent("!WARNING", file, "New or changed in both paths")

				//if files are identical, leave them alone instead of renaming
				if (dirs1.has(file) || dirs1.has(alias)) && (dirs2.has(file) || dirs2.has(alias)) {
					fs.Debugf(nil, "This is a directory, not a file. Skipping equality check and will not rename: %s", file)
					ls1.getPut(file, skippedDirs1)
					ls2.getPut(file, skippedDirs2)
				} else {
					equal := matches.Has(file)
					if !equal {
						equal = matches.Has(alias)
					}
					if equal {
						if ciCheck.FixCase && file != alias {
							// the content is equal but filename still needs to be FixCase'd, so copy1to2
							// the Path1 version is deemed "correct" in this scenario
							fs.Infof(alias, "Files are equal but will copy anyway to fix case to %s", file)
							copy1to2.Add(file)
						} else {
							fs.Infof(nil, "Files are equal! Skipping: %s", file)
							renameSkipped.Add(file)
							renameSkipped.Add(alias)
						}
					} else {
						fs.Debugf(nil, "Files are NOT equal: %s", file)
						b.indent("!Path1", p1+"..path1", "Renaming Path1 copy")
						ctxMove = b.setBackupDir(ctxMove, 1) // in case already a file with new name
						if err = operations.MoveFile(ctxMove, b.fs1, b.fs1, file+"..path1", file); err != nil {
							err = fmt.Errorf("path1 rename failed for %s: %w", p1, err)
							b.critical = true
							return
						}
						if b.opt.DryRun {
							renameSkipped.Add(file)
						} else {
							renamed1.Add(file)
						}
						b.indent("!Path1", p2+"..path1", "Queue copy to Path2")
						copy1to2.Add(file + "..path1")

						b.indent("!Path2", p2+"..path2", "Renaming Path2 copy")
						ctxMove = b.setBackupDir(ctxMove, 2) // in case already a file with new name
						if err = operations.MoveFile(ctxMove, b.fs2, b.fs2, alias+"..path2", alias); err != nil {
							err = fmt.Errorf("path2 rename failed for %s: %w", alias, err)
							return
						}
						if b.opt.DryRun {
							renameSkipped.Add(alias)
						} else {
							renamed2.Add(alias)
						}
						b.indent("!Path2", p1+"..path2", "Queue copy to Path1")
						copy2to1.Add(alias + "..path2")
					}
				}
				handled.Add(file)
			}
		} else {
			// Path1 deleted
			d2, in2 := ds2.deltas[file]
			// try looking under alternate name
			fs.Debugf(file, "alias: %s, in2: %v", alias, in2)
			if !in2 && file != alias {
				fs.Debugf(file, "looking for alias: %s", alias)
				d2, in2 = ds2.deltas[alias]
				if in2 {
					fs.Debugf(file, "detected alias: %s", alias)
				}
			}
			if !in2 {
				b.indent("Path2", p2, "Queue delete")
				delete2.Add(file)
				copy1to2.Add(file)
			} else if d2.is(deltaOther) {
				b.indent("Path2", p1, "Queue copy to Path1")
				copy2to1.Add(file)
				handled.Add(file)
			} else if d2.is(deltaDeleted) {
				handled.Add(file)
				deletedonboth.Add(file)
				deletedonboth.Add(alias)
			}
		}
	}

	for _, file := range ds2.sort() {
		alias := b.aliases.Alias(file)
		p1 := path1 + alias
		d2 := ds2.deltas[file]

		if handled.Has(file) || handled.Has(alias) {
			continue
		}
		if d2.is(deltaOther) {
			b.indent("Path2", p1, "Queue copy to Path1")
			copy2to1.Add(file)
		} else {
			// Deleted
			b.indent("Path1", p1, "Queue delete")
			delete1.Add(file)
			copy2to1.Add(file)
		}
	}

	// Do the batch operation
	if copy2to1.NotEmpty() {
		changes1 = true
		b.indent("Path2", "Path1", "Do queued copies to")
		ctx = b.setBackupDir(ctx, 1)
		results2to1, err = b.fastCopy(ctx, b.fs2, b.fs1, copy2to1, "copy2to1")

		// retries, if any
		results2to1, err = b.retryFastCopy(ctx, b.fs2, b.fs1, copy2to1, "copy2to1", results2to1, err)

		if err != nil {
			return
		}

		//copy empty dirs from path2 to path1 (if --create-empty-src-dirs)
		b.syncEmptyDirs(ctx, b.fs1, copy2to1, dirs2, &results2to1, "make")
	}

	if copy1to2.NotEmpty() {
		changes2 = true
		b.indent("Path1", "Path2", "Do queued copies to")
		ctx = b.setBackupDir(ctx, 2)
		results1to2, err = b.fastCopy(ctx, b.fs1, b.fs2, copy1to2, "copy1to2")

		// retries, if any
		results1to2, err = b.retryFastCopy(ctx, b.fs1, b.fs2, copy1to2, "copy1to2", results1to2, err)

		if err != nil {
			return
		}

		//copy empty dirs from path1 to path2 (if --create-empty-src-dirs)
		b.syncEmptyDirs(ctx, b.fs2, copy1to2, dirs1, &results1to2, "make")
	}

	if delete1.NotEmpty() {
		if err = b.saveQueue(delete1, "delete1"); err != nil {
			return
		}
		//propagate deletions of empty dirs from path2 to path1 (if --create-empty-src-dirs)
		b.syncEmptyDirs(ctx, b.fs1, delete1, dirs1, &results2to1, "remove")
	}

	if delete2.NotEmpty() {
		if err = b.saveQueue(delete2, "delete2"); err != nil {
			return
		}
		//propagate deletions of empty dirs from path1 to path2 (if --create-empty-src-dirs)
		b.syncEmptyDirs(ctx, b.fs2, delete2, dirs2, &results1to2, "remove")
	}

	queues.copy1to2 = copy1to2
	queues.copy2to1 = copy2to1
	queues.renamed1 = renamed1
	queues.renamed2 = renamed2
	queues.renameSkipped = renameSkipped
	queues.deletedonboth = deletedonboth
	queues.skippedDirs1 = skippedDirs1
	queues.skippedDirs2 = skippedDirs2

	return
}

// excessDeletes checks whether number of deletes is within allowed range
func (ds *deltaSet) excessDeletes() bool {
	maxDelete := ds.opt.MaxDelete
	maxRatio := float64(maxDelete) / 100.0
	curRatio := 0.0
	if ds.deleted > 0 && ds.oldCount > 0 {
		curRatio = float64(ds.deleted) / float64(ds.oldCount)
	}

	if curRatio <= maxRatio {
		return false
	}

	fs.Errorf("Safety abort",
		"too many deletes (>%d%%, %d of %d) on %s %s. Run with --force if desired.",
		maxDelete, ds.deleted, ds.oldCount, ds.msg, quotePath(bilib.FsPath(ds.fs)))
	return true
}

// normally we build the AliasMap from march results,
// however, march does not know about deleted files, so need to manually check them for aliases
func (b *bisyncRun) updateAliases(ctx context.Context, ds1, ds2 *deltaSet) {
	ci := fs.GetConfig(ctx)
	// skip if not needed
	if ci.NoUnicodeNormalization && !ci.IgnoreCaseSync && !b.fs1.Features().CaseInsensitive && !b.fs2.Features().CaseInsensitive {
		return
	}
	if ds1.deleted < 1 && ds2.deleted < 1 {
		return
	}

	fs.Debugf(nil, "Updating AliasMap")

	transform := func(s string) string {
		if !ci.NoUnicodeNormalization {
			s = norm.NFC.String(s)
		}
		// note: march only checks the dest, but we check both here
		if ci.IgnoreCaseSync || b.fs1.Features().CaseInsensitive || b.fs2.Features().CaseInsensitive {
			s = strings.ToLower(s)
		}
		return s
	}

	delMap1 := map[string]string{}  // [transformedname]originalname
	delMap2 := map[string]string{}  // [transformedname]originalname
	fullMap1 := map[string]string{} // [transformedname]originalname
	fullMap2 := map[string]string{} // [transformedname]originalname

	for _, name := range ls1.list {
		fullMap1[transform(name)] = name
	}
	for _, name := range ls2.list {
		fullMap2[transform(name)] = name
	}

	addDeletes := func(ds *deltaSet, delMap, fullMap map[string]string) {
		for _, file := range ds.sort() {
			d := ds.deltas[file]
			if d.is(deltaDeleted) {
				delMap[transform(file)] = file
				fullMap[transform(file)] = file
			}
		}
	}
	addDeletes(ds1, delMap1, fullMap1)
	addDeletes(ds2, delMap2, fullMap2)

	addAliases := func(delMap, fullMap map[string]string) {
		for transformedname, name := range delMap {
			matchedName, found := fullMap[transformedname]
			if found && name != matchedName {
				fs.Debugf(name, "adding alias %s", matchedName)
				b.aliases.Add(name, matchedName)
			}
		}
	}
	addAliases(delMap1, fullMap2)
	addAliases(delMap2, fullMap1)
}

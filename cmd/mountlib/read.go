package mountlib

import (
	"io"
	"sync"

	"github.com/ncw/rclone/fs"
)

// ReadFileHandle is an open for read file handle on a File
type ReadFileHandle struct {
	mu         sync.Mutex
	closed     bool // set if handle has been closed
	r          *fs.Account
	o          fs.Object
	readCalled bool // set if read has been called
	offset     int64
	noSeek     bool
	file       *File
}

func newReadFileHandle(f *File, o fs.Object, noSeek bool) (*ReadFileHandle, error) {
	r, err := o.Open()
	if err != nil {
		return nil, err
	}
	fh := &ReadFileHandle{
		o:      o,
		r:      fs.NewAccount(r, o).WithBuffer(), // account the transfer
		noSeek: noSeek,
		file:   f,
	}
	fs.Stats.Transferring(fh.o.Remote())
	return fh, nil
}

// Node returns the Node assocuated with this - satisfies Noder interface
func (fh *ReadFileHandle) Node() Node {
	return fh.file
}

// seek to a new offset
//
// if reopen is true, then we won't attempt to use an io.Seeker interface
//
// Must be called with fh.mu held
func (fh *ReadFileHandle) seek(offset int64, reopen bool) (err error) {
	if fh.noSeek {
		return ESPIPE
	}
	fh.r.StopBuffering() // stop the background reading first
	oldReader := fh.r.GetReader()
	r := oldReader
	// Can we seek it directly?
	if do, ok := oldReader.(io.Seeker); !reopen && ok {
		fs.Debugf(fh.o, "ReadFileHandle.seek from %d to %d (io.Seeker)", fh.offset, offset)
		_, err = do.Seek(offset, 0)
		if err != nil {
			fs.Debugf(fh.o, "ReadFileHandle.Read io.Seeker failed: %v", err)
			return err
		}
	} else {
		fs.Debugf(fh.o, "ReadFileHandle.seek from %d to %d", fh.offset, offset)
		// close old one
		err = oldReader.Close()
		if err != nil {
			fs.Debugf(fh.o, "ReadFileHandle.Read seek close old failed: %v", err)
		}
		// re-open with a seek
		r, err = fh.o.Open(&fs.SeekOption{Offset: offset})
		if err != nil {
			fs.Debugf(fh.o, "ReadFileHandle.Read seek failed: %v", err)
			return err
		}
	}
	fh.r.UpdateReader(r)
	fh.offset = offset
	return nil
}

// Read from the file handle
func (fh *ReadFileHandle) Read(reqSize, reqOffset int64) (respData []byte, err error) {
	fh.mu.Lock()
	defer fh.mu.Unlock()
	fs.Debugf(fh.o, "ReadFileHandle.Read size %d offset %d", reqSize, reqOffset)
	if fh.closed {
		fs.Errorf(fh.o, "ReadFileHandle.Read error: %v", EBADF)
		return nil, EBADF
	}
	doSeek := reqOffset != fh.offset
	var n int
	var newOffset int64
	retries := 0
	buf := make([]byte, reqSize)
	doReopen := false
	for {
		if doSeek {
			// Are we attempting to seek beyond the end of the
			// file - if so just return EOF leaving the underlying
			// file in an unchanged state.
			if reqOffset >= fh.o.Size() {
				fs.Debugf(fh.o, "ReadFileHandle.Read attempt to read beyond end of file: %d > %d", reqOffset, fh.o.Size())
				respData = nil
				return nil, nil
			}
			// Otherwise do the seek
			err = fh.seek(reqOffset, doReopen)
		} else {
			err = nil
		}
		if err == nil {
			if reqSize > 0 {
				fh.readCalled = true
			}
			// One exception to the above is if we fail to fully populate a
			// page cache page; a read into page cache is always page aligned.
			// Make sure we never serve a partial read, to avoid that.
			n, err = io.ReadFull(fh.r, buf)
			newOffset = fh.offset + int64(n)
			// if err == nil && rand.Intn(10) == 0 {
			// 	err = errors.New("random error")
			// }
			if err == nil {
				break
			} else if (err == io.ErrUnexpectedEOF || err == io.EOF) && newOffset == fh.o.Size() {
				// Have read to end of file - reset error
				err = nil
				break
			}
		}
		if retries >= fs.Config.LowLevelRetries {
			break
		}
		retries++
		fs.Errorf(fh.o, "ReadFileHandle.Read error: low level retry %d/%d: %v", retries, fs.Config.LowLevelRetries, err)
		doSeek = true
		doReopen = true
	}
	if err != nil {
		fs.Errorf(fh.o, "ReadFileHandle.Read error: %v", err)
	} else {
		respData = buf[:n]
		fh.offset = newOffset
		fs.Debugf(fh.o, "ReadFileHandle.Read OK")
	}
	return respData, err
}

// close the file handle returning EBADF if it has been
// closed already.
//
// Must be called with fh.mu held
func (fh *ReadFileHandle) close() error {
	if fh.closed {
		return EBADF
	}
	fh.closed = true
	fs.Stats.DoneTransferring(fh.o.Remote(), true)
	return fh.r.Close()
}

// Flush is called each time the file or directory is closed.
// Because there can be multiple file descriptors referring to a
// single opened file, Flush can be called multiple times.
func (fh *ReadFileHandle) Flush() error {
	fh.mu.Lock()
	defer fh.mu.Unlock()
	fs.Debugf(fh.o, "ReadFileHandle.Flush")

	// Ignore the Flush as there is nothing we can sensibly do and
	// it seems quite common for Flush to be called from
	// different threads each of which have read some data.
	if false {
		// If Read hasn't been called then ignore the Flush - Release
		// will pick it up
		if !fh.readCalled {
			fs.Debugf(fh.o, "ReadFileHandle.Flush ignoring flush on unread handle")
			return nil

		}
		err := fh.close()
		if err != nil {
			fs.Errorf(fh.o, "ReadFileHandle.Flush error: %v", err)
			return err
		}
	}
	fs.Debugf(fh.o, "ReadFileHandle.Flush OK")
	return nil
}

// Release is called when we are finished with the file handle
//
// It isn't called directly from userspace so the error is ignored by
// the kernel
func (fh *ReadFileHandle) Release() error {
	fh.mu.Lock()
	defer fh.mu.Unlock()
	if fh.closed {
		fs.Debugf(fh.o, "ReadFileHandle.Release nothing to do")
		return nil
	}
	fs.Debugf(fh.o, "ReadFileHandle.Release closing")
	err := fh.close()
	if err != nil {
		fs.Errorf(fh.o, "ReadFileHandle.Release error: %v", err)
	} else {
		fs.Debugf(fh.o, "ReadFileHandle.Release OK")
	}
	return err
}

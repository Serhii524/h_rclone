package fs

import (
	"io"

	"github.com/pkg/errors"
)

// A RepeatableReader implements the io.ReadSeeker it allow to seek cached data
// back and forth within the reader but will only read data from the internal Reader as necessary
// and will play nicely with the Account and io.LimitedReader to reflect current speed
type RepeatableReader struct {
	in io.Reader // Input reader
	i  int64     // current reading index
	b  []byte    // internal cache buffer
}

var _ io.ReadSeeker = (*RepeatableReader)(nil)

// Seek implements the io.Seeker interface.
// If seek position is passed the cache buffer length the function will return
// the maximum offset that can be used and "fs.RepeatableReader.Seek: offset is unavailable" Error
func (r *RepeatableReader) Seek(offset int64, whence int) (int64, error) {
	var abs int64
	cacheLen := int64(len(r.b))
	switch whence {
	case 0: //io.SeekStart
		abs = offset
	case 1: //io.SeekCurrent
		abs = r.i + offset
	case 2: //io.SeekEnd
		abs = cacheLen + offset
	default:
		return 0, errors.New("fs.RepeatableReader.Seek: invalid whence")
	}
	if abs < 0 {
		return 0, errors.New("fs.RepeatableReader.Seek: negative position")
	}
	if abs > cacheLen {
		return offset - (abs - cacheLen), errors.New("fs.RepeatableReader.Seek: offset is unavailable")
	}
	r.i = abs
	return abs, nil
}

// Read data from original Reader into bytes
// Data is either served from the underlying Reader or from cache if was already read
func (r *RepeatableReader) Read(b []byte) (n int, err error) {
	cacheLen := int64(len(r.b))
	if r.i == cacheLen {
		n, err = r.in.Read(b)
		if n > 0 {
			r.b = append(r.b, b[:n]...)
		}
	} else {
		n = copy(b, r.b[r.i:])
	}
	r.i += int64(n)
	return n, err
}

// NewRepeatableReader create new repeatable reader from Reader r
func NewRepeatableReader(r io.Reader) *RepeatableReader {
	return &RepeatableReader{in: r}
}

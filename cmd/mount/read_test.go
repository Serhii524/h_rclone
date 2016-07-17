// +build linux darwin freebsd

package mount

import (
	"io"
	"os"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Read by byte including don't read any bytes
func TestReadByByte(t *testing.T) {
	var data = []byte("hellohello")
	run.createFile(t, "testfile", string(data))
	run.checkDir(t, "testfile 10")

	for i := 0; i < len(data); i++ {
		fd, err := os.Open(run.path("testfile"))
		assert.NoError(t, err)
		for j := 0; j < i; j++ {
			buf := make([]byte, 1)
			n, err := io.ReadFull(fd, buf)
			assert.NoError(t, err)
			assert.Equal(t, 1, n)
			assert.Equal(t, buf[0], data[j])
		}
		err = fd.Close()
		assert.NoError(t, err)
	}

	run.rm(t, "testfile")
}

// Test double close
func TestReadFileDoubleClose(t *testing.T) {
	run.createFile(t, "testdoubleclose", "hello")

	in, err := os.Open(run.path("testdoubleclose"))
	assert.NoError(t, err)
	fd := in.Fd()

	fd1, err := syscall.Dup(int(fd))
	assert.NoError(t, err)

	fd2, err := syscall.Dup(int(fd))
	assert.NoError(t, err)

	// close one of the dups - should produce no error
	err = syscall.Close(fd1)
	assert.NoError(t, err)

	// read from the file
	buf := make([]byte, 1)
	_, err = in.Read(buf)
	assert.NoError(t, err)

	// close it
	err = in.Close()
	assert.NoError(t, err)

	// read from the other dup - should produce no error as this
	// file is now buffered
	n, err := syscall.Read(fd2, buf)
	assert.NoError(t, err)
	assert.Equal(t, 1, n)

	// close the dup - should produce an error
	err = syscall.Close(fd2)
	assert.Error(t, err, "input/output error")

	run.rm(t, "testdoubleclose")
}

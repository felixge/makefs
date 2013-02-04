package makefs

import (
	"fmt"
	"io"
	"os"
	"syscall"
	"time"
)

type MemoryFile struct {
	name    string
	isDir   bool
	modTime int64
	size    int64
	data    string
	offset  int64
	closed  bool
}

func (f *MemoryFile) Read(buf []byte) (int, error) {
	if f.closed {
		// @TODO: POSIX compliance.
		return 0, fmt.Errorf("can not read from %s, already closed", f.name)
	}

	if f.offset >= int64(len(f.data)) {
		return 0, io.EOF
	}

	n := copy(buf, f.data[f.offset:])
	f.offset += int64(n)
	return n, nil
}

func (f *MemoryFile) Close() error {
	if f.closed {
		// @TODO: POSIX compliance.
		return fmt.Errorf("can close %s, already closed", f.name)
	}

	f.closed = true
	return nil
}

func (f MemoryFile) Stat() (MemoryFileInfo, error) {
	result := NewMemoryFileInfo(f)

	return result, nil
}

// @TODO
func (f *MemoryFile) ReadDir(count int) ([]os.FileInfo, error) {
	result := []os.FileInfo{}
	return result, nil
}

func (f *MemoryFile) Seek(offset int64, whence int) (int64, error) {
	if f.isDir {
		return f.offset, &os.PathError{"seek", f.name, syscall.EISDIR}
	}

	var start int64

	switch whence {
	case os.SEEK_SET:
		start = 0
	case os.SEEK_CUR:
		start = f.offset
	case os.SEEK_END:
		start = f.size
	default:
		return f.offset, &os.PathError{"seek", f.name, syscall.EINVAL}
	}

	result := start + offset

	if result < 0 {
		return f.offset, &os.PathError{"seek", f.name, syscall.EINVAL}
	}

	f.offset = result

	return result, nil
}

type MemoryFileInfo struct {
	f MemoryFile
}

func NewMemoryFileInfo(f MemoryFile) MemoryFileInfo {
	return MemoryFileInfo{
		f: f,
	}
}

func (f *MemoryFileInfo) Size() int64 {
	return f.f.size
}

// @TODO
func (f *MemoryFileInfo) Mode() os.FileMode {
	if f.IsDir() {
		return os.ModeDir
	}

	return os.ModeDir
}

func (f *MemoryFileInfo) ModTime() time.Time {
	return time.Unix(f.f.modTime, 0)
}

func (f *MemoryFileInfo) IsDir() bool {
	return f.f.isDir
}

func (f *MemoryFileInfo) Sys() interface{} {
	return nil
}

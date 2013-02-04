package makefs

import (
	"io"
	"os"
	"syscall"
	"time"
)

type MemoryFile struct {
	Name    string
	IsDir   bool
	Data    string
	ModTime int64
	offset  int64
	closed  bool
}

func (f *MemoryFile) Read(buf []byte) (int, error) {
	if f.closed {
		return -1, &os.PathError{"read", f.Name, syscall.EBADF}
	}

	if f.offset >= int64(len(f.Data)) {
		return 0, io.EOF
	}

	n := copy(buf, f.Data[f.offset:])
	f.offset += int64(n)
	return n, nil
}

func (f *MemoryFile) Close() error {
	if f.closed {
		return &os.PathError{"close", f.Name, syscall.EBADF}
	}

	f.closed = true
	return nil
}

func (f *MemoryFile) Stat() (os.FileInfo, error) {
	return newMemoryFileInfo(f), nil
}

// @TODO
func (f *MemoryFile) ReadDir(count int) ([]os.FileInfo, error) {
	result := []os.FileInfo{}
	return result, nil
}

func (f *MemoryFile) Seek(offset int64, whence int) (int64, error) {
	if f.closed {
		return -1, &os.PathError{"lseek", f.Name, syscall.EBADF}
	}

	if f.IsDir {
		return f.offset, &os.PathError{"lseek", f.Name, syscall.EISDIR}
	}

	var start int64

	switch whence {
	case os.SEEK_SET:
		start = 0
	case os.SEEK_CUR:
		start = f.offset
	case os.SEEK_END:
		start = int64(len(f.Data))
	default:
		return f.offset, &os.PathError{"lseek", f.Name, syscall.EINVAL}
	}

	result := start + offset

	if result < 0 {
		return f.offset, &os.PathError{"lseek", f.Name, syscall.EINVAL}
	}

	f.offset = result

	return result, nil
}

func newMemoryFileInfo(file *MemoryFile) *memoryFileInfo {
	return &memoryFileInfo{file: file}
}

type memoryFileInfo struct {
	file *MemoryFile
}

func (f *memoryFileInfo) Name() string {
	return f.file.Name
}

func (f *memoryFileInfo) Size() int64 {
	return int64(len(f.file.Data))
}

func (f *memoryFileInfo) Mode() os.FileMode {
	// 4 = read
	mode := os.FileMode(0444)
	if f.IsDir() {
		// 1 = execute
		mode = mode | os.ModeDir | 0111
	}
	return mode
}

func (f *memoryFileInfo) ModTime() time.Time {
	return time.Unix(f.file.ModTime, 0)
}

func (f *memoryFileInfo) IsDir() bool {
	return f.file.IsDir
}

// @TODO Should we return something here?
func (f *memoryFileInfo) Sys() interface{} {
	return nil
}

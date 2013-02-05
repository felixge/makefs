package makefs

import (
	"io"
	"net/http"
	"os"
	gopath "path"
	"strings"
	"syscall"
	"time"
)

type MemoryFs struct {
	root MemoryFile
}

func NewMemoryFs(root MemoryFile) *MemoryFs {
	return &MemoryFs{root: root}
}

func (f *MemoryFs) Open(path string) (http.File, error) {
	current := f.root
	if path == "/" {
		return &current, nil
	}

	trailingSlash := strings.HasSuffix(path, "/")
	path = gopath.Clean(path)
	parts := strings.Split(path[1:], "/")

	for i, part := range parts {
		for _, file := range current.Children {
			if file.Name == part {
				current = file
				if lastPart := i+1 >= len(parts); lastPart {
					if trailingSlash && !file.IsDir {
						break
					}
					return &current, nil
				}
				break
			}
		}
	}
	return nil, &os.PathError{"open", path, os.ErrNotExist}
}

type MemoryFile struct {
	Name          string
	IsDir         bool
	Data          string
	ModTime       int64
	Children      []MemoryFile
	readdirOffset int
	readOffset    int64
	closed        bool
}

func (f *MemoryFile) Read(buf []byte) (int, error) {
	if f.closed {
		return -1, &os.PathError{"read", f.Name, syscall.EBADF}
	}

	if f.readOffset >= int64(len(f.Data)) {
		return 0, io.EOF
	}

	n := copy(buf, f.Data[f.readOffset:])
	f.readOffset += int64(n)
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
	if f.closed {
		return nil, &os.PathError{"fstat", f.Name, syscall.EBADF}
	}

	return newMemoryFileInfo(f), nil
}

func (f *MemoryFile) Readdir(count int) ([]os.FileInfo, error) {
	if f.readdirOffset >= len(f.Children) {
		return nil, io.EOF
	}

	files := f.Children[f.readdirOffset:]
	if count > 0 {
		files = files[0:count]
	}

	stats := make([]os.FileInfo, len(files))
	for i, _ := range files {
		stats[i] = newMemoryFileInfo(&files[i])
	}

	f.readdirOffset += len(stats)
	return stats, nil
}

func (f *MemoryFile) Seek(offset int64, whence int) (int64, error) {
	if f.closed {
		return -1, &os.PathError{"lseek", f.Name, syscall.EBADF}
	}

	if f.IsDir {
		return f.readOffset, &os.PathError{"lseek", f.Name, syscall.EISDIR}
	}

	var start int64

	switch whence {
	case os.SEEK_SET:
		start = 0
	case os.SEEK_CUR:
		start = f.readOffset
	case os.SEEK_END:
		start = int64(len(f.Data))
	default:
		return f.readOffset, &os.PathError{"lseek", f.Name, syscall.EINVAL}
	}

	result := start + offset

	if result < 0 {
		return f.readOffset, &os.PathError{"lseek", f.Name, syscall.EINVAL}
	}

	f.readOffset = result

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
	mode := os.FileMode(0400)
	if f.IsDir() {
		// 1 = execute
		mode = mode | os.ModeDir | 0100
	}
	return mode
}

func (f *memoryFileInfo) ModTime() time.Time {
	return time.Unix(f.file.ModTime, 0)
}

func (f *memoryFileInfo) IsDir() bool {
	return f.file.IsDir
}

func (f *memoryFileInfo) Sys() interface{} {
	// @TODO Should we return something here?
	return nil
}

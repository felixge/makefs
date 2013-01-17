package makefs

import (
	"net/http"
	"os"
	gopath "path"
	"strings"
)

func NewSubFs(base http.FileSystem, path string) http.FileSystem {
	return &SubFs{base: base, path: path}
}

type SubFs struct {
	base http.FileSystem
	path string
}

func (fs *SubFs) Open(path string) (http.File, error) {
	subPath := gopath.Join(fs.path, path)
	if !strings.HasPrefix(subPath, fs.path) {
		return nil, os.ErrPermission
	}
	return fs.base.Open(subPath)
}

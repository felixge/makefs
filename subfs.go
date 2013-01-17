package makefs

import (
	"net/http"
	gopath "path"
)

func NewSubFs(base http.FileSystem, path string) http.FileSystem {
	return &SubFs{base: base, path: path}
}

type SubFs struct {
	base http.FileSystem
	path string
}

func (fs *SubFs) Open(path string) (http.File, error) {
	subPath := gopath.Join(fs.path, gopath.Clean("/"+path))
	return fs.base.Open(subPath)
}

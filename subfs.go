package makefs

import (
	"net/http"
	"os"
	gopath "path"
	"strings"
)

func NewSubFs(base http.FileSystem, newRoot string) *SubFs {
	return &SubFs{base: base, root: newRoot}
}

type SubFs struct {
	base http.FileSystem
	root string
}

func (fs *SubFs) Open(path string) (http.File, error) {
	subPath := gopath.Join(fs.root, path)
	if !strings.HasPrefix(subPath, fs.root) {
		return nil, os.ErrPermission
	}
	return fs.base.Open(subPath)
}

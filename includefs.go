package makefs

import (
	"net/http"
	"os"
	"strings"
)

func NewIncludeFs(base http.FileSystem, includes []string) http.FileSystem {
	return &IncludeFs{base: base, includes: includes}
}

type IncludeFs struct {
	base     http.FileSystem
	includes []string
}

func (fs *IncludeFs) Open(path string) (http.File, error) {
	for _, include := range fs.includes {
		if strings.HasPrefix(path, include) {
			return fs.base.Open(path)
		}
	}
	return nil, &os.PathError{"open", path, os.ErrNotExist}
}

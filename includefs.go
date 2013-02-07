package makefs

import (
	"net/http"
	"os"
	gopath "path"
	"strings"
)

func NewIncludeFs(base http.FileSystem, includes ...string) http.FileSystem {
	return &IncludeFs{base: base, includes: includes}
}

type IncludeFs struct {
	base     http.FileSystem
	includes []string
}

func (fs *IncludeFs) Open(path string) (http.File, error) {
	for _, include := range fs.includes {
		if strings.HasPrefix(path, include) || strings.HasPrefix(include, path) {
			file, err := fs.base.Open(path)
			if file != nil {
				file = &includeFsFile{File: file, path: path, includes: fs.includes}
			}
			return file, err
		}
	}
	return nil, &os.PathError{"open", path, os.ErrNotExist}
}

type includeFsFile struct {
	http.File
	path     string
	includes []string
}

func (f *includeFsFile) Readdir(count int) ([]os.FileInfo, error) {
	stats, err := f.File.Readdir(count)
	if err != nil {
		return stats, err
	}

	results := make([]os.FileInfo, 0, len(stats))
	for _, stat := range stats {
		path := gopath.Join(f.path, stat.Name())
		for _, include := range f.includes {
			if strings.HasPrefix(path, include) || strings.HasPrefix(include, path) {
				results = append(results, stat)
				break
			}
		}
	}
	return results, nil
}

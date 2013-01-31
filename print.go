package makefs

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	gopath "path"
	"time"
)

func Fprint(w io.Writer, fs http.FileSystem, pkg string, varName string) error {
	return fprintPath(w, fs, "/")
}

func fprintPath(w io.Writer, fs http.FileSystem, path string) error {
	file, err := fs.Open(path)
	if err != nil {
		return err
	}

	stat, err := file.Stat()
	if err != nil {
		return err
	}

	if err := fprintFile(w, file, stat, path); err != nil {
		return err
	}

	if !stat.IsDir() {
		return nil
	}

	stats, err := file.Readdir(-1)
	if err != nil {
		return err
	}

	for _, stat := range stats {
		filePath := gopath.Join(path, stat.Name())
		if err := fprintPath(w, fs, filePath); err != nil {
			return err
		}
	}

	return nil
}

const entry = `{
	isDir: %#v,
	modTime: %#v,
	data: %#v,
},
`

func fprintFile(w io.Writer, file http.File, stat os.FileInfo, path string) error {
	var data string

	isDir := stat.IsDir()
	if !isDir {
		d, err := ioutil.ReadAll(file)
		if err != nil {
			return err
		}
		data = string(d)
	}

	if _, err := fmt.Fprintf(
		w,
		entry,
		isDir,
		stat.ModTime().Format(time.RFC3339),
		data,
	); err != nil {
		return err
	}
	return nil
}

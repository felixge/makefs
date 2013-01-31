package makefs

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	gopath "path"
)

var headerTmpl = `// machine generated; do not edit by hand

package %s

func init() {
	bundledFs = newBundleFs(map[string]MemoryFile{
		"/": &MemoryFile{
			name		: %#v,
			isDir		: %#v,
			modTime	: %#v,
			data		: %#v,
		}
	})
}
`

const fileTemplate = `{
	name		: %#v,
	isDir		: %#v,
	modTime	: %#v,
	data		: %#v,
},
`

func Fprint(w io.Writer, fs http.FileSystem, pkgName string, varName string) error {
	printer := &printer{
		w:       w,
		pkgName: pkgName,
		varName: varName,
		fs:      fs,
	}
	return printer.Print("/")
}

type printer struct {
	w       io.Writer
	pkgName string
	varName string
	fs      http.FileSystem
	indent  int
}

func (p *printer) Write(buf []byte) (int, error) {
	return p.w.Write(buf)
}

func (p *printer) Print(rootPath string) error {
	if _, err := fmt.Fprintf(p.w, headerTemplate, p.pkgName); err != nil {
		return err
	}
	return p.printPath(rootPath)
}

func (p *printer) printPath(path string) error {
	file, err := p.fs.Open(path)
	if err != nil {
		return err
	}

	stat, err := file.Stat()
	if err != nil {
		return err
	}

	if err := p.printFile(file, stat); err != nil {
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
		if err := p.printPath(gopath.Join(path, stat.Name())); err != nil {
			return err
		}
	}

	return nil
}

func (p *printer) printFile(file http.File, stat os.FileInfo) error {
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
		p,
		fileTemplate,
		stat.Name(),
		isDir,
		stat.ModTime().Unix(),
		data,
	); err != nil {
		return err
	}
	return nil
}

func (p *printer) fileEnd() error {
	return nil
}

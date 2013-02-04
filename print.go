package makefs

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	gopath "path"
	"strings"
)

var headerTmpl = `// machine generated; do not edit by hand

package %s

func init() {
	bundledFs = newBundleFs(map[string]*{{prefix}}MemoryFile{
`

var footerTmpl = `
	},
)
}
`

const fileTemplate = `%#v: &%sMemoryFile{
	name		: %#v,
	isDir		: %#v,
	modTime	: %#v,
	size	  : %#v,
	data		: %#v,
},
`

func Fprint(w io.Writer, fs http.FileSystem, pkgname string, varname string) error {
	printer := &printer{
		w:       w,
		pkgname: pkgname,
		prefix:  varname,
		varname: varname,
		fs:      fs,
	}
	return printer.Print("/")
}

type printer struct {
	w       io.Writer
	pkgname string
	prefix  string
	varname string
	fs      http.FileSystem
}

func (p *printer) Print(rootPath string) error {
	headerTmpl = strings.Replace(headerTmpl, "{{prefix}}", p.prefix, -1)

	_, err := fmt.Fprintf(p.w, headerTmpl, p.pkgname)
	if err != nil {
		return err
	}

	if err := p.printPath(rootPath); err != nil {
		return err
	}

	_, err = fmt.Fprintf(p.w, footerTmpl)
	if err != nil {
		return err
	}

	return nil
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

	if err := p.printFile(path, file, stat); err != nil {
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
		subPath := gopath.Join(path, stat.Name())
		if err := p.printPath(subPath); err != nil {
			return err
		}
	}

	return nil
}

func (p *printer) printFile(path string, file http.File, stat os.FileInfo) error {
	var data string

	isDir := stat.IsDir()

	if !isDir {
		d, err := ioutil.ReadAll(file)
		if err != nil {
			return err
		}
		data = string(d)
	}

	_, err := fmt.Fprintf(
		p.w,
		fileTemplate,
		path,
		p.prefix,
		stat.Name(),
		isDir,
		stat.ModTime().Unix(),
		stat.Size(),
		data,
	)
	return err
}

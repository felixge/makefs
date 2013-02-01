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

import (
	"os"
	"time"
)

type {{prefix}}MemoryFile struct{
	Name    string
	IsDir   bool
	ModTime int64
	Size    int64
	Data    string
	offset  int64
}

// @TODO
func (f *{{prefix}}MemoryFile) Read([]byte) (int, error) {
	return 0, nil
}

func (f *{{prefix}}MemoryFile) Close() error {
	return nil
}

func (f {{prefix}}MemoryFile) Stat() (os.FileInfo, error) {
	result := New{{prefix}}MemoryFileInfo(f)

	return result, nil
}

// @TODO
func (f *{{prefix}}MemoryFile) ReadDir(count int) ([]os.FileInfo, error) {
	result := []os.FileInfo{}
	return result, nil
}

func (f *{{prefix}}MemoryFile) Seek(offset int64, whence int) (int64, error) {
	if f.IsDir {
		theErr := fmt.Errorf("File %s is a directory - can't seek", f.Name)
		err := &PathError{"seek", f.name, theErr}
		return 0, err
	}

	start := 0

	switch whence {
		case SEEK_SET:
			start := 0
		case SEEK_CUR:
			start := f.offset
		case SEEK_END:
			start := f.Size - 1
	}

	result := start + offset
	if result < 0 || result > f.Size - 1 {
		theErr := fmt.Errorf("Seek out of bounds")
		err := &PathError{"seek", f.name, theErr}
		return 0, err
	}

	f.offset = result

	return result, nil
}

type {{prefix}}MemoryFileInfo struct{
	f {{prefix}}MemoryFile
}

func New{{prefix}}MemoryFileInfo(f {{prefix}}MemoryFile) *{{prefix}}MemoryFileInfo {
	return &{{prefix}}MemoryFileInfo{
		f: f,
	}
}

func (f *{{prefix}}MemoryFileInfo) Size() int64 {
	return f.f.Size
}

// @TODO
func (f *{{prefix}}MemoryFileInfo) Mode() FileMode {
	if f.IsDir() {
		return ModeDir
	}

	return nil
}

func (f *{{prefix}}MemoryFileInfo) ModTime() time.Time {
	return time.Unix(f.f.ModTime, 0)
}

func (f *{{prefix}}MemoryFileInfo) IsDir() bool {
	return f.f.IsDir
}

func (f *{{prefix}}MemoryFileInfo) Sys() interface{} {
	return nil
}


func init() {
	bundledFs = newBundleFs(map[string]*{{prefix}}MemoryFile{
`

var footerTmpl = `
	},
)
}
`

const fileTemplate = `%#v: &%sMemoryFile{
	Name		: %#v,
	IsDir		: %#v,
	ModTime	: %#v,
	Size	  : %#v,
	Data		: %#v,
},
`

func Fprint(w io.Writer, fs http.FileSystem, pkgName string, varName string) error {
	printer := &printer{
		w:       w,
		pkgName: pkgName,
		prefix:  varName,
		varName: varName,
		fs:      fs,
	}
	return printer.Print("/")
}

type printer struct {
	w       io.Writer
	pkgName string
	prefix  string
	varName string
	fs      http.FileSystem
}

func (p *printer) Print(rootPath string) error {
	headerTmpl = strings.Replace(headerTmpl, "{{prefix}}", p.prefix, -1)

	_, err := fmt.Fprintf(p.w, headerTmpl, p.pkgName)
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

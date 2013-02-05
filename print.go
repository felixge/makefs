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

func Fprint(w io.Writer, fs http.FileSystem, pkgName string, varName string) error {
	printer := &printer{
		w:       w,
		pkgName: pkgName,
		varName: varName,
		fs:      fs,
	}
	return printer.print("/")
}

type printer struct {
	w       io.Writer
	pkgName string
	prefix  string
	varName string
	fs      http.FileSystem
	indent  int
}

func (p *printer) write(str string, args ...interface{}) {
	if _, err := fmt.Fprintf(p.w, str, args...); err != nil {
		panic(err)
	}
}

func (p *printer) line(str string, args ...interface{}) {
	p.startLine(str+"\n", args...)
}

func (p *printer) startLine(str string, args ...interface{}) {
	str = strings.Repeat("\t", p.indent) + str
	p.write(str, args...)
}

func (p *printer) endLine(str string, args ...interface{}) {
	p.write(str+"\n", args...)
}

func (p *printer) print(rootPath string) error {
	p.line("package %s", p.pkgName)
	p.line("")
	p.line("/* machine generated; do not edit */")
	p.line("")
	p.line(`import "github.com/felixge/makefs"`)
	p.line("")
	p.line("func init() {")
	p.indent++
	p.startLine("%s = makefs.NewMemoryFs(makefs.MemoryFile", p.varName)
	if err := p.printPath("/"); err != nil {
		return err
	}
	p.endLine(")")
	p.indent--
	p.line("}")
	return nil
}

func (p *printer) printPath(path string) error {
	file, err := p.fs.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return err
	}

	if err := p.startFile(file, stat); err != nil {
		return err
	}

	if !stat.IsDir() {
		return p.endFile(file, stat)
	}

	stats, err := file.Readdir(-1)
	if err != nil {
		return err
	}

	if len(stats) > 0 {
		p.endLine("")
		p.indent++
		p.startLine("")
	}

	for i, stat := range stats {
		subPath := gopath.Join(path, stat.Name())
		if err := p.printPath(subPath); err != nil {
			return err
		}

		if last := i+1 >= len(stats); last {
			p.indent--
		}
		p.startLine("")
	}

	if path != "/" {
		return p.endFile(file, stat)
	}

	p.endLine("},")
	p.indent--
	p.startLine("}")
	return nil
}

func (p *printer) startFile(file http.File, stat os.FileInfo) error {
	p.endLine("{")
	p.indent++
	p.line("Name:\t\t%#v,", stat.Name())
	p.line("IsDir:\t\t%#v,", stat.IsDir())
	p.line("ModTime:\t%#v,", stat.ModTime().Unix())
	if !stat.IsDir() {
		data, err := ioutil.ReadAll(file)
		if err != nil {
			return err
		}
		p.line("Data:\t\t%#v,", string(data))
	} else {
		p.startLine("Children: []makefs.MemoryFile{")
	}
	return nil
}

func (p *printer) endFile(file http.File, stat os.FileInfo) error {
	if stat.IsDir() {
		p.endLine("},")
	}
	p.indent--
	p.line("},")
	return nil
}

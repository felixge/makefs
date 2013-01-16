package makefs

/*
Vocabulary:

* pattern: a string containing exactly one % sign.
* glob: a string containing one or more * character. Single stars expand to any
	character except for the path separator. Double stars expand to any character,
	including the separator.


*/

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"syscall"
)

type ruleFs struct {
	parent http.FileSystem
	rule   *rule
}

//var errUndefined := fmt.Errorf("")

func (fs *ruleFs) Open(path string) (http.File, error) {
	task := &task{target: newBroadcast()}

	if len(fs.rule.targets) > 1 {
		return nil, fmt.Errorf("not done yet: multiple targets")
	}

	if len(fs.rule.sources) > 1 {
		return nil, fmt.Errorf("not done yet: multiple sources")
	}

outer:
	for _, target := range fs.rule.targets {
		if isPattern(target) {
			stem := findStem(path, target)

			// target pattern did not match, rule does not apply
			if stem == "" {
				break
			}

			for _, source := range fs.rule.sources {
				if isPattern(source) {
					sourcePath := insertStem(source, stem)
					sourceFile, err := fs.parent.Open(sourcePath)
					if isNotFound(err) {
						break outer
					} else if err != nil {
						return nil, err
					}

					task.source = &Source{file: sourceFile}
				}
			}
		}
	}

	go func() {
		fs.rule.recipe(task)
		task.Source().file.Close()
	}()

	return &proxyFile{reader: task.target.Client()}, nil
}

func (fs *ruleFs) readdir(file *proxyFile, count int) ([]os.FileInfo, error) {
	return file.parent.Readdir(count)
}

func (fs *ruleFs) stat(file *proxyFile) (os.FileInfo, error) {
	return file.parent.Stat()
}

func isPattern(str string) bool {
	return strings.Contains(str, "%")
}

// findStem returns the value the % wildcard in pattern fills in the given str,
// or "" if the pattern does not match.
func findStem(str string, pattern string) string {
	stemOffset := strings.Index(pattern, "%")
	if stemOffset < 0 {
		return ""
	}

	prefix := pattern[0:stemOffset]
	suffix := pattern[stemOffset+1:]

	if str[0:len(prefix)] != prefix {
		return ""
	}

	if str[len(str)-len(suffix):] != suffix {
		return ""
	}

	return str[len(prefix) : len(str)-len(suffix)]
}

func insertStem(pattern string, stem string) string {
	return strings.Replace(pattern, "%", stem, -1)
}

func isGlob(str string) bool {
	return strings.Contains(str, "*")
}

func isNotFound(err error) bool {
	pathErr, ok := err.(*os.PathError)
	if !ok {
		return false
	}

	return pathErr.Err == syscall.ENOENT
}

type proxyFile struct {
	ruleFs *ruleFs
	path   string
	reader io.Reader

	parent http.File

	//Close() error
	//Stat() (os.FileInfo, error)
	//Readdir(count int) ([]os.FileInfo, error)
	//Read([]byte) (int, error)
	//Seek(offset int64, whence int) (int64, error)
}

func (file *proxyFile) Close() error {
	return fmt.Errorf("close not implemented yet")
}

func (file *proxyFile) Read(buf []byte) (int, error) {
	return file.reader.Read(buf)
}

func (file *proxyFile) Seek(offset int64, whence int) (int64, error) {
	return 0, fmt.Errorf("eseek not implemented yet")
}

func (file *proxyFile) Readdir(count int) ([]os.FileInfo, error) {
	return file.ruleFs.readdir(file, count)
}

func (file *proxyFile) Stat() (os.FileInfo, error) {
	return file.ruleFs.stat(file)
}

type task struct {
	target *broadcast
	source *Source
}

type Target struct {
}

type Source struct {
	file http.File
}

func (source *Source) Read(buf []byte) (int, error) {
	return source.file.Read(buf)
}

func (t *task) Target() io.WriteCloser {
	return t.target
}

func (t *task) Source() *Source {
	return t.source
}

type rule struct {
	targets []string
	sources []string
	recipe  func(*task)
}

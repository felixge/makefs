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
)

type ruleFs struct {
	parent http.FileSystem
	rule   *rule
}

//var errUndefined := fmt.Errorf("")

func (fs *ruleFs) Open(path string) (http.File, error) {
	task := &task{}

	if len(fs.rule.targets) > 1 {
		return nil, fmt.Errorf("not done yet: multiple targets")
	}

	if len(fs.rule.sources) > 1 {
		return nil, fmt.Errorf("not done yet: multiple sources")
	}

	for _, target := range fs.rule.targets {
		if !isPattern(target) {
			return nil, fmt.Errorf("not done yet: non-pattern targets")
		}

		stem := findStem(path, target)

		// target pattern did not match, rule does not apply
		if stem == "" {
			break
		}

		task.target = newBroadcast()

		for _, source := range fs.rule.sources {
			if !isPattern(source) {
				return nil, fmt.Errorf("not done yet: non-pattern sources")
			}

			sourcePath := insertStem(source, stem)
			sourceFile, err := fs.parent.Open(sourcePath)
			if os.IsNotExist(err) {
				return nil, fmt.Errorf("not done yet: pattern source not found")
			} else if err != nil {
				return nil, err
			}

			task.source = &Source{file: sourceFile}
		}
	}

	if task.target == nil {
		file, err := fs.parent.Open(path)
		if file == nil {
			return file, err
		}
		return &proxyFile{File: file, ruleFs: fs}, err
	}

	go func() {
		if err := fs.rule.recipe(task); err != nil {
			// what?
		}
		task.Source().file.Close()
	}()

	return newTargetFile(task.target.Client()), nil
}

func (fs *ruleFs) readdir(file *proxyFile, count int) ([]os.FileInfo, error) {
	return file.File.Readdir(count)
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

	if len(str) < len(suffix) || str[len(str)-len(suffix):] != suffix {
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

type proxyFile struct {
	http.File
	ruleFs *ruleFs
}

func (file *proxyFile) Readdir(count int) ([]os.FileInfo, error) {
	return file.ruleFs.readdir(file, count)
}

func newTargetFile(reader *client) *targetFile {
	return &targetFile{
		reader: reader,
	}
}

type targetFile struct {
	ruleFs *ruleFs
	reader *client
}

func (file *targetFile) Close() error {
	return fmt.Errorf("not done yet: Close()")
}

func (file *targetFile) Read(buf []byte) (int, error) {
	return file.reader.Read(buf)
}

func (file *targetFile) Seek(offset int64, whence int) (int64, error) {
	return 0, fmt.Errorf("not done yet: Seek()")
}

func (file *targetFile) Readdir(count int) ([]os.FileInfo, error) {
	// @TODO is there something more idomatic we can return here that makes sense
	// cross-plattform?
	return nil, fmt.Errorf("readdir: target file is not a dir")
}

func (file *targetFile) Stat() (os.FileInfo, error) {
	return nil, fmt.Errorf("not done yet: Stat()")
}

type task struct {
	target *broadcast
	source *Source
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
	recipe  func(*task) error
}

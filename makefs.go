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
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func NewFs(base http.FileSystem) *Fs {
	return &Fs{head: base}
}

type Fs struct {
	head http.FileSystem
}

func (fs *Fs) Open(path string) (http.File, error) {
	return fs.head.Open(path)
}

func (fs *Fs) Make(target string, source string, recipe Recipe) {
	rule := &rule{
		targets: []string{target},
		sources: []string{source},
		recipe: recipe,
	}

	fs.head = &ruleFs{
		parent: fs.head,
		rule: rule,
	}
}

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

			task.source = sourceFile
		}
	}

	if task.target == nil {
		file, err := fs.parent.Open(path)
		if file == nil {
			return file, err
		}
		return &proxyFile{File: file, ruleFs: fs, path: path}, err
	}

	// @TODO, do not execute recipe until first Read() ?
	go func() {
		if err := fs.rule.recipe(task); err != nil {
			// what?
		}
	}()

	return newTargetFile(task.target, path), nil
}

func (fs *ruleFs) readdir(file *proxyFile, count int) ([]os.FileInfo, error) {
	if len(fs.rule.targets) > 1 {
		return nil, fmt.Errorf("not done yet: multiple targets")
	}

	if len(fs.rule.sources) > 1 {
		return nil, fmt.Errorf("not done yet: multiple sources")
	}

	stats, err := file.File.Readdir(count)
	if err != nil {
		return nil, err
	}

	results := []os.FileInfo{}
	for _, stat := range stats {
		for _, source := range fs.rule.sources {
			if !isPattern(source) {
				return nil, fmt.Errorf("not done yet: non-pattern sources")
			}

			stem := findStem(filepath.Join(file.path, stat.Name()), source)

			// source pattern did not match, break inner loop
			if stem == "" {
				results = append(results, stat)
				break
			}

			for _, target := range fs.rule.targets {
				if !isPattern(target) {
					return nil, fmt.Errorf("not done yet: non-pattern targets")
				}

				targetPath := insertStem(target, stem)
				targetFile, err := fs.Open(targetPath)
				if err != nil {
					return nil, err
				}
				defer targetFile.Close()

				targetStat, err := targetFile.Stat()
				if err != nil {
					return nil, err
				}

				results = append(results, targetStat)
			}
		}
	}

	return results, nil
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
	path   string
	ruleFs *ruleFs
}

func (file *proxyFile) Readdir(count int) ([]os.FileInfo, error) {
	return file.ruleFs.readdir(file, count)
}

func newTargetFile(broadcast *broadcast, path string) *targetFile {
	return &targetFile{
		reader:    broadcast.Client(),
		broadcast: broadcast,
		path:      path,
	}
}

type targetFile struct {
	ruleFs    *ruleFs
	reader    *client
	broadcast *broadcast
	path      string
}

func (file *targetFile) Close() error {
	// @TODO make future read calls fail
	return nil
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
	stat := &targetStat{targetFile: file}
	return stat, nil
}

type targetStat struct {
	targetFile *targetFile
}

func (s *targetStat) IsDir() bool {
	// @TODO support targets that are directories
	return false
}

func (s *targetStat) ModTime() time.Time {
	// @TODO finish
	return time.Now()
}

func (s *targetStat) Mode() os.FileMode {
	// @TODO Finish
	return 0
}

func (s *targetStat) Name() string {
	return filepath.Base(s.targetFile.path)
}

// Size determines the size of the target file by creating a new broadcast
// client, and counting the bytes until EOF. It returns -1 if the broadcast
// client returns an error other than EOF from read.
//
// This means that calling this methods requires executing the recipe.
func (s *targetStat) Size() int64 {
	client := s.targetFile.broadcast.Client()
	n, err := io.Copy(ioutil.Discard, client)
	if err != nil {
		return -1
	}
	return n
}

func (s *targetStat) Sys() interface{} {
	return nil
}

type task struct {
	target *broadcast
	source http.File
}

func (t *task) Target() io.WriteCloser {
	return t.target
}

func (t *task) Source() io.ReadCloser {
	return t.source
}

type Recipe func(*task) error

type rule struct {
	targets []string
	sources []string
	recipe  Recipe
}

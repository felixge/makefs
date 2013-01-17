package makefs

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type ruleFs struct {
	parent http.FileSystem
	rule   *rule
}

type errInvalidRule string

func (err errInvalidRule) Error() string {
	return "invalid rule: " + string(err)
}

func (fs *ruleFs) Open(path string) (http.File, error) {
	if err := fs.checkRule(); err != nil {
		return nil, err
	}

	if fs.isSource(path) {
		return nil, os.ErrNotExist
	}

	// path is not a taget, so we'll return whatever the parent fs has to offer.
	// If the parent fs returns a file we'll proxy all Readdir calls on the file
	// back to our ruleFs (as we need to apply our rule to them).
	if !fs.isTarget(path) {
		file, err := fs.parent.Open(path)
		if file == nil {
			return nil, err
		}
		return &readdirProxy{File: file, ruleFs: fs, path: path}, err
	}

	task, err := fs.task(path)

	// something went wrong (invalid rule, source could not be opened, etc.).
	if err != nil {
		return nil, err
	}

	return newTargetFile(task, path), nil
}

func (fs *ruleFs) task(path string) (*Task, error) {
	target := fs.rule.targets[0]
	stem := findStem(path, target)

	// target pattern did not match, no task can be synthesized
	if stem == "" {
		return nil, nil
	}

	task := &Task{target: newBroadcast()}

	source := fs.rule.sources[0]
	sourcePath := insertStem(source, stem)
	sourceFile, err := fs.parent.Open(sourcePath)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("not done yet: pattern source not found")
	} else if err != nil {
		return nil, err
	}

	task.source = sourceFile

	task.runFunc = func() {
		err := fs.rule.recipe(task)
		task.target.CloseWithError(err)
		task.source.Close()
	}

	return task, nil
}

func (fs *ruleFs) isSource(path string) bool {
	for _, source := range fs.rule.sources {
		// non-pattern sources not done yet
		if !isPattern(source) {
			return false
		}

		stem := findStem(path, source)
		if stem != "" {
			return true
		}
	}
	return false
}

func (fs *ruleFs) isTarget(path string) bool {
	for _, target := range fs.rule.targets {
		// non-pattern targets not done yet
		if !isPattern(target) {
			return false
		}

		stem := findStem(path, target)
		if stem != "" {
			return true
		}
	}
	return false
}

func (fs *ruleFs) readdir(file *readdirProxy, count int) ([]os.FileInfo, error) {
	if err := fs.checkRule(); err != nil {
		return nil, err
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

// checkRule determines if the given rule can be executed by ruleFs. It will
// return an error if the rule is invalid, or support for it has not been
// implemented yet.
func (fs *ruleFs) checkRule() error  {
	// check if the rule itself is valid
	if err := fs.rule.Check(); err != nil {
		return err
	}

	// then make sure ruleFs supports it already

	if len(fs.rule.targets) > 1 {
		return errInvalidRule("multiple targets not supported yet")
	}

	if len(fs.rule.sources) > 1 {
		return errInvalidRule("multiple sources not supported yet")
	}

	if !isPattern(fs.rule.targets[0]) {
		return errInvalidRule("non-pattern targets not supported yet")
	}

	if !isPattern(fs.rule.sources[0]) {
		return errInvalidRule("non-pattern sources not supported yet")
	}

	return nil
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

	if len(str) < len(prefix) || str[0:len(prefix)] != prefix {
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

type readdirProxy struct {
	http.File
	path   string
	ruleFs *ruleFs
}

func (file *readdirProxy) Readdir(count int) ([]os.FileInfo, error) {
	return file.ruleFs.readdir(file, count)
}

func newTargetFile(task *Task, path string) *targetFile {
	return &targetFile{
		task: task,
		path: path,
	}
}

type targetFile struct {
	task   *Task
	path   string
	reader io.Reader
}

func (file *targetFile) Close() error {
	// @TODO make future read calls fail
	return nil
}

func (file *targetFile) Read(buf []byte) (int, error) {
	if file.reader == nil {
		file.reader = file.client()
	}
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

func (file *targetFile) client() io.Reader {
	// make sure our recipe is executed
	file.task.start()
	return file.task.target.Client()
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
	client := s.targetFile.client()
	n, err := io.Copy(ioutil.Discard, client)
	if err != nil {
		return -1
	}
	return n
}

func (s *targetStat) Sys() interface{} {
	return nil
}

type Task struct {
	runFunc func()
	runOnce sync.Once
	target  *broadcast
	source  http.File
}

func (t *Task) Target() io.Writer {
	return t.target
}

func (t *Task) Source() io.Reader {
	return t.source
}

// start executes the recipe unless it has already started executing, in which
// case the call is ignored.
func (t *Task) start() {
	go t.runOnce.Do(t.runFunc)
}

type Recipe func(*Task) error

type rule struct {
	targets []string
	sources []string
	recipe  Recipe
}

func (r *rule) Check() error {
	if len(r.targets) < 1 {
		return errInvalidRule("does not contain any targets")
	}
	return nil
}

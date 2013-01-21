package makefs

import (
	"io"
	"net/http"
	"os"
	gopath "path"
	"regexp"
	"strings"
	"sync"
)

func newRuleFs(parent http.FileSystem, rule *rule) (*ruleFs, error) {
	if err := rule.Check(); err != nil {
		return nil, err
	}

	ruleFs := &ruleFs{
		parent: parent,
		rule:   rule,
		cache:  make(map[string]*Task),
	}

	return ruleFs, nil
}

type ruleFs struct {
	parent    http.FileSystem
	rule      *rule
	cacheLock sync.Mutex
	cache     map[string]*Task
}

type errInvalidRule string

func (err errInvalidRule) Error() string {
	return "invalid rule: " + string(err)
}

func (fs *ruleFs) Open(path string) (http.File, error) {
	// Try to synthesize a task for the give path
	task, err := fs.task(path)
	if err != nil {
		return nil, err
	}

	// No task means we just forward this request to the parent fs. However, we
	// return a readdirProxy to hijack any Readdir() calls on the returned file.
	if task == nil {
		file, err := fs.parent.Open(path)
		if file == nil {
			return nil, err
		}
		return &readdirProxy{File: file, ruleFs: fs, path: path}, err
	}

	return task.target.httpFile(), nil
}

func (fs *ruleFs) task(path string) (*Task, error) {
	// Aquire lock to synchronize task creation / cache access
	fs.cacheLock.Lock()
	defer fs.cacheLock.Unlock()

	// Find all sources
	sources, err := fs.rule.findSources(path, fs.parent)
	if err != nil {
		return nil, err
		// Note: This is different from len(sources) == 0, which is a valid task
		// that does not depend on any sources (.PHONY in make).
	} else if sources == nil {
		return nil, nil
	}

	// Synthesize task
	task := newTask(path, sources)

	// Check if we already synthesized this task before and can reuse it.
	if cachedTask, ok := fs.cache[path]; ok {
		if cachedTask.current(task) {
			task = cachedTask
		}
	}

	// Update cache
	fs.cache[path] = task

	return task, nil
}

func (fs *ruleFs) readdir(file *readdirProxy, count int) ([]os.FileInfo, error) {
	// Get stats from parent file system
	stats, err := file.File.Readdir(count)
	if err != nil {
		return nil, err
	}

	fileDir := gopath.Clean(file.path)

	var knownTargets map[string]bool
	for _, stat := range stats {
		// Resolve full path
		path := gopath.Join(fileDir, stat.Name())

		// Returns the target this path is a source of
		target := fs.rule.findTarget(path)
		if target == nil {
			continue
		}

		// If we already found this target, skip it from now on
		if knownTargets[target.path] {
			continue
		}
		knownTargets[target.path] = true

		// We only care about targets inside the directory being read
		targetDir := gopath.Dir(target.path)
		if targetDir != fileDir {
			continue
		}

		// Open the target http file
		targetFile, err := fs.Open(target.path)
		if err != nil {
			return nil, err
		}
		defer targetFile.Close()

		// Get the stat info (this does not trigger the recipe unless Size()
		// is invoked).
		targetStat, err := targetFile.Stat()
		if err != nil {
			return nil, err
		}

		// @TODO check if this target is overwriting a source, if so replace
		// it instead of append.
		stats = append(stats, targetStat)
	}

	return stats, nil
}

func isPattern(str string) bool {
	return strings.Contains(str, "%")
}

func isAbs(str string) bool {
	return gopath.IsAbs(str)
}

// findStem returns the value the % wildcard in pattern fills in the given str,
// or "" if the pattern does not match.
func findStem(path string, pattern string) string {
	pattern = regexp.QuoteMeta(pattern)
	pattern = strings.Replace(pattern, "%", "(.+?)", 1) + "$"

	matcher, err := regexp.Compile(pattern)
	if err != nil {
		panic("unreachable")
	}

	match := matcher.FindStringSubmatch(path)
	if len(match) != 2 {
		return ""
	}

	return match[1]
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

//func newTargetFile(task *Task, path string) *targetFile {
//return &targetFile{
//task: task,
//path: path,
//}
//}

//type targetFile struct {
//task   *Task
//path   string
//reader io.Reader
//}

//func (file *targetFile) Close() error {
//// @TODO make future read calls fail
//return nil
//}

//func (file *targetFile) Read(buf []byte) (int, error) {
//if file.reader == nil {
//file.reader = file.client()
//}
//return file.reader.Read(buf)
//}

//func (file *targetFile) Seek(offset int64, whence int) (int64, error) {
//return 0, fmt.Errorf("not done yet: Seek()")
//}

//func (file *targetFile) Readdir(count int) ([]os.FileInfo, error) {
//// @TODO is there something more idomatic we can return here that makes sense
//// cross-plattform?
//return nil, fmt.Errorf("readdir: target file is not a dir")
//}

//func (file *targetFile) Stat() (os.FileInfo, error) {
//stat := &targetStat{targetFile: file}
//return stat, nil
//}

//func (file *targetFile) client() io.Reader {
//// make sure our recipe is executed
//file.task.start()
//return file.task.target.Client()
//}

//type targetStat struct {
//targetFile *targetFile
//}

//func (s *targetStat) IsDir() bool {
//// @TODO support targets that are directories
//return false
//}

//func (s *targetStat) ModTime() time.Time {
//// @TODO finish
//return time.Now()
//}

//func (s *targetStat) Mode() os.FileMode {
//// @TODO Finish
//return 0
//}

//func (s *targetStat) Name() string {
//return gopath.Base(s.targetFile.path)
//}

//// Size determines the size of the target file by creating a new broadcast
//// client, and counting the bytes until EOF. It returns -1 if the broadcast
//// client returns an error other than EOF from read.
////
//// This means that calling this methods requires executing the recipe.
//func (s *targetStat) Size() int64 {
//client := s.targetFile.client()
//n, err := io.Copy(ioutil.Discard, client)
//if err != nil {
//return -1
//}
//return n
//}

//func (s *targetStat) Sys() interface{} {
//return nil
//}

func newTask(targetPath string, sources []*Source) *Task {
	return nil
}

type Task struct {
	target *Target
}

func (t *Task) id() string {
	return ""
}

func (t *Task) current(other *Task) bool {
	return false
}

func (t *Task) Target() io.Writer {
	return nil
}

func (t *Task) Source() io.Reader {
	return nil
}

type Recipe func(*Task) error

type rule struct {
	target  string
	sources []string
	recipe  Recipe
}

func (r *rule) Check() error {
	if r.target == "" {
		return errInvalidRule("does not contain any targets")
	}

	return nil
}

func (r *rule) findSources(targetPath string, fs http.FileSystem) ([]*Source, error) {
	stem := findStem(targetPath, r.target)
	if stem == "" && targetPath != r.target {
		return nil, nil
	}

	results := make([]*Source, 0)
	for _, source := range r.sources {
		source = insertStem(source, stem)
		
		sources, err := expand(source, fs)
		if err != nil {
			return nil, err
		}

		results = append(results, sources...)
	}

	return results, nil
}

func (r *rule) findTarget(sourcePath string) *Target {
	return nil
}

func newTarget(path string) *Target {
	return &Target{path: path}
}

type Target struct {
	path string
}

func (t *Target) httpFile() http.File {
	return nil
}

type Source struct{}

func expand(pattern string, fs http.FileSystem) ([]*Source, error) {
	return nil, nil
}

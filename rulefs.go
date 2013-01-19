package makefs

import (
	"io"
	"net/http"
	"os"
	gopath "path"
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
	parent http.FileSystem
	rule   *rule
	cacheLock      sync.Mutex
	cache  map[string]*Task
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

	// Task can have multiple targets, return the right one
	for _, target := range task.targets {
		if target.path == path {
			return target.httpFile(), nil
		}
	}
	panic("unreachable")
}

func (fs *ruleFs) task(path string) (*Task, error) {
	// Aquire lock to synchronize task creation / cache access
	fs.cacheLock.Lock()
	defer fs.cacheLock.Unlock()

	// Find all targets
	targets := fs.rule.targetsForTargetPath(path)
	if targets == nil {
		return nil, nil
	}

	// Find all sources
	sources, err := fs.rule.sourcesForTargets(targets, fs.parent)
	if err != nil {
		return nil, err
	} else if sources == nil {
		// Note: This is different from len(sources) == 0, which is a valid task
		// that does not depend on any sources (.PHONY in make).
		return nil, nil
	}

	// Synthesize task
	task := newTask(targets, sources)

	// Check if we already synthesized this task before and can reuse it.
	id := task.id()
	if cachedTask, ok := fs.cache[id]; ok {
		if cachedTask.current(task) {
			task = cachedTask
		}
	}

	// Update cache
	fs.cache[id] = task

	return task, nil
}

func (fs *ruleFs) readdir(file *readdirProxy, count int) ([]os.FileInfo, error) {
	parentStats, err := file.File.Readdir(count)
	if err != nil {
		return nil, err
	}

	var results []os.FileInfo
	var knownTargets map[string]bool

	for _, parentStat := range parentStats {
		parentPath := gopath.Join(file.path, parentStat.Name())
		targets := fs.rule.targetsForSourcePath(parentPath)
		if targets == nil {
			results = append(results, parentStat)
			continue
		}

		for _, target := range targets {
			if knownTargets[target.path] {
				continue
			}

			targetFile, err := fs.Open(target.path)
			if err != nil {
				return nil, err
			}
			defer targetFile.Close()

			targetStat, err := targetFile.Stat()
			if err != nil {
				return nil, err
			}

			results = append(results, targetStat)
			knownTargets[target.path] = true
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

type Target struct {
	path string
}

func (t *Target) httpFile() http.File {
	return nil
}

type Source struct{}

func newTask(targets []*Target, sources []*Source) *Task {
	return nil
}

type Task struct {
	targets []*Target
}

func (t *Task) id() string {
	return ""
}

func (t *Task) current(other *Task) bool {
	return false
}

func (t *Task) targetFile(path string) http.File {
	return nil
}

func (t *Task) Target() io.Writer {
	return nil
}

func (t *Task) Source() io.Reader {
	return nil
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

func (r *rule) targetsForTargetPath(path string) []*Target  {
	return nil
}

func (r *rule) targetsForSourcePath(path string) []*Target  {
	return nil
}

func (r *rule) sourcesForTargets(targets []*Target, fs http.FileSystem) ([]*Source, error)  {
	return nil, nil
}

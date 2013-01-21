package makefs

import (
	"net/http"
	"os"
	gopath "path"
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
	task := newTask(path, sources, fs.rule.recipe)

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

type readdirProxy struct {
	http.File
	path   string
	ruleFs *ruleFs
}

func (file *readdirProxy) Readdir(count int) ([]os.FileInfo, error) {
	return file.ruleFs.readdir(file, count)
}

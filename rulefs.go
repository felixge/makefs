package makefs

import (
	"fmt"
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
	// Try to synthesize a task for the given path
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
	}

	// Note: This is different from len(sources) == 0, which is a valid task
	// that does not depend on any sources (.PHONY in make).
	if sources == nil {
		return nil, nil
	}

	// Synthesize task
	task := newTask(path, sources, fs.rule.recipe)

	// Check if we already synthesized this task before and can reuse it.
	if cachedTask, ok := fs.cache[path]; ok {
		if cachedTask.equal(task) {
			task = cachedTask
		}
	}

	// Update cache
	fs.cache[path] = task

	return task, nil
}

// BUG: For all stats produced by a rule, Readdir does not support count > 0 /
// returns an error in this case.

func (fs *ruleFs) readdir(file *readdirProxy, count int) ([]os.FileInfo, error) {
	if count > 0 {
		return nil, fmt.Errorf("makefs: Readdir with count > 0 not supported yet")
	}

	// Get items from parent file system
	stats, err := file.File.Readdir(0)
	if err != nil {
		return nil, err
	}

	// Get all targets created by this rule
	targets, err := fs.rule.findTargetPaths(fs.parent)
	if err != nil {
		return nil, err
	}

	// Get the canonical name of this dir
	dir := gopath.Clean(file.path)

	// Allocate a big enough results slice
	results := make([]os.FileInfo, 0, len(stats)+len(targets))
	// Keep track of all paths to remove duplicates
	knownPaths := make(map[string]bool, len(results))

	for _, path := range targets {
		if knownPaths[path] {
			continue
		}
		knownPaths[path] = true

		// Check if this target is inside the directory we are listing
		targetDir := gopath.Dir(path)
		if targetDir != dir {
			continue
		}

		targetFile, err := fs.Open(path)
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

	for _, stat := range stats {
		path := gopath.Join(dir, stat.Name())
		if knownPaths[path] {
			continue
		}
		knownPaths[path] = true

		results = append(results, stat)
	}
	
	// @TODO: We should probably sort the results before returning them.

	return results, nil
}

type readdirProxy struct {
	http.File
	path   string
	ruleFs *ruleFs
}

func (file *readdirProxy) Readdir(count int) ([]os.FileInfo, error) {
	return file.ruleFs.readdir(file, count)
}

package makefs

import (
	"fmt"
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
	if task, err := fs.task(path); err != nil {
		return nil, err
	} else if task != nil {
		return task.target.httpFile(), nil
	}

	// No task means we check in the parentFs
	file, err := fs.parent.Open(path)
	if file != nil {
		return &readdirProxy{File: file, fs: fs, path: path}, err
	}

	// If this is not a IsNotExist error, we can't handle it
	if !os.IsNotExist(err) {
		return nil, err
	}
	// keep a reference to this error, in case we can not recover from it
	notFoundErr := err

	// At this point, we need to check if one of the targets of our rule
	// indirectly creates the path we are looking for as a directory.
	targets, err := fs.rule.findTargetPaths(fs.parent)
	if err != nil {
		return nil, err
	}

	for _, target := range targets {
		if !strings.HasPrefix(target, path) {
			continue
		}

		file := &MemoryFile{Name: gopath.Base(path), IsDir: true}
		return &readdirProxy{File: file, fs: fs, path: path}, nil
	}

	return nil, notFoundErr
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

type readdirProxy struct {
	http.File
	fs   *ruleFs
	path string
}

func (file *readdirProxy) Readdir(count int) ([]os.FileInfo, error) {
	if count > 0 {
		return nil, fmt.Errorf("makefs: Readdir with count > 0 not supported yet")
	}

	// Get items from parent file system
	stats, err := file.File.Readdir(0)
	if err != nil && err != io.EOF {
		return nil, err
	}

	// Get all targets created by this rule
	targets, err := file.fs.rule.findTargetPaths(file.fs.parent)
	if err != nil {
		return nil, err
	}

	// Get the canonical name of this dir
	dir := gopath.Clean(file.path)

	// Allocate a big enough results slice
	results := make([]os.FileInfo, 0, len(stats)+len(targets))
	// Keep track of all paths to remove duplicates
	knownPaths := make(map[string]bool, len(results))

	// First we look for targets created by this rule that belong into this
	// directory and add them to our results
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

		targetFile, err := file.fs.Open(path)
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

	// Next we consider the items for the actual directory itself, and add them
	// as well (unless they were already added as a target).
	for _, stat := range stats {
		path := gopath.Join(dir, stat.Name())
		if knownPaths[path] {
			continue
		}
		knownPaths[path] = true

		results = append(results, stat)
	}

	// At last, we add result entries for directories that are indirectly created
	// by a target.
	for _, path := range targets {
		if !strings.HasPrefix(path, dir) {
			continue
		}

		parts := strings.Split(path[len(dir):], "/")
		name := parts[0]
		if dir != "/" {
			name = parts[1]
		}

		path := gopath.Join(dir, name)
		if knownPaths[path] {
			continue
		}
		knownPaths[path] = true

		file := &MemoryFile{Name: name, IsDir: true}
		stat, err := file.Stat()
		if err != nil {
			return nil, err
		}

		results = append(results, stat)
	}


	// @TODO: We should probably sort the results before returning them.

	return results, nil
}

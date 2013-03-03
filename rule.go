package makefs

import (
	"fmt"
	"net/http"
	gopath "path"
	"regexp"
	"sort"
	"strings"
)

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

	// @TODO: Do not allow rules with % on only one side

	return nil
}

func (r *rule) findSources(targetPath string, fs http.FileSystem) ([]*Source, error) {
	var stem = ""
	if targetPath == r.target {
		// exact match, no stem / prefix
	} else if isPattern(r.target) {
		stem = findStem(targetPath, r.target)
		if stem == "" {
			return nil, nil
		}
	} else {
		return nil, nil
	}

	sources := make([]*Source, 0)
	for _, source := range r.sources {
		sourcePath := insertStem(source, stem)
		globSources, err := Glob(sourcePath, fs)
		if err != nil {
			return nil, err
		}

		globPaths := make(sort.StringSlice, 0, len(globSources))
		for path, _ := range globSources {
			globPaths = append(globPaths, path)
		}

		sort.Sort(globPaths)

		for _, path := range globPaths {
			sources = append(sources, &Source{
				path: path,
				fs:   fs,
				stat: globSources[path],
			})
		}
	}

	if len(sources) == 0 {
		return nil, nil
	}
	return sources, nil
}

func (r *rule) findTargetPaths(fs http.FileSystem) ([]string, error) {
	var (
		dirs    = []string{"/"}
		results = []string{}
	)

	for len(dirs) > 0 {
		dir := dirs[0]
		dirs = dirs[1:]

		// Open the dir
		file, err := fs.Open(dir)
		if err != nil {
			return nil, err
		}
		defer file.Close()

		stat, err := file.Stat()
		if err != nil {
			return nil, err
		}

		// This should never happen / might be panic-worthy.
		if !stat.IsDir() {
			return nil, fmt.Errorf("makefs: unexpected file in findTargetPaths at: %s", dir)
		}

		stats, err := file.Readdir(-1)
		if err != nil {
			return nil, err
		}

		for _, stat := range stats {
			path := gopath.Join(dir, stat.Name())
			if !stat.IsDir() {
				target, err := r.findTargetPath(path, fs)
				if err != nil {
					return nil, err
				}

				if target != "" {
					results = append(results, target)
				}
			} else {
				dirs = append(dirs, path)
			}
		}
	}

	return results, nil
}

func (r *rule) findTargetPath(sourcePath string, fs http.FileSystem) (string, error) {
	targetPath := r.target
	if isPattern(targetPath) {
		var stem = ""
		for _, source := range r.sources {
			if isPattern(source) {
				stem = findStem(sourcePath, source)

				// Use the first stem we find in a source
				if stem != "" {
					break
				}
			}
		}

		// Cannot resolve pattern target without stem
		if stem == "" {
			return "", nil
		}

		// But if we got a stem, let's insert it
		targetPath = insertStem(r.target, stem)
	}

	// For this targetPath to be valid, *all* sources need to exist
	sources, err := r.findSources(targetPath, fs)
	if err != nil {
		return "", err
	}
	if sources == nil {
		return "", nil
	}

	return targetPath, nil
}

func findStem(path string, pattern string) string {
	pattern = regexp.QuoteMeta(pattern)
	pattern = "^" + strings.Replace(pattern, "%", "(.+)", 1) + "$"

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

func isPattern(str string) bool {
	return strings.Contains(str, "%")
}

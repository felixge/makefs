package makefs

import (
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
	var (
		stem = ""
		dir  = ""
	)

	if targetPath == r.target {
		// exact match, no stem / prefix
	} else if isPattern(r.target) {
		stem, dir = findStem(targetPath, r.target)
		if stem == "" {
			return nil, nil
		}
	} else {
		return nil, nil
	}

	sources := make([]*Source, 0)
	for _, source := range r.sources {
		sourcePath := gopath.Join(dir, insertStem(source, stem))
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

func (r *rule) resolveTargetPath(sourcePath string, fs http.FileSystem) (string, error) {
	targetPath := r.target
	if isPattern(targetPath) {
		var (
			stem = ""
			dir  = ""
		)

		for _, source := range r.sources {
			if isPattern(source) {
				stem, dir = findStem(sourcePath, source)

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
		targetPath = gopath.Join(dir, insertStem(r.target, stem))
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

func findStem(path string, pattern string) (string, string) {
	var (
		pathDir     = gopath.Dir(path)
		pathBase    = gopath.Base(path)
		patternDir  = gopath.Dir(pattern)
		patternBase = gopath.Base(pattern)
	)

	if patternDir != "." && !strings.HasSuffix(pathDir, patternDir) {
		return "", ""
	}

	patternBase = regexp.QuoteMeta(patternBase)
	patternBase = "^" + strings.Replace(patternBase, "%", "(.+)", 1) + "$"

	matcher, err := regexp.Compile(patternBase)
	if err != nil {
		panic("unreachable")
	}

	match := matcher.FindStringSubmatch(pathBase)
	if len(match) != 2 {
		return "", ""
	}

	return match[1], pathDir
}

func insertStem(pattern string, stem string) string {
	return strings.Replace(pattern, "%", stem, -1)
}

func isPattern(str string) bool {
	return strings.Contains(str, "%")
}

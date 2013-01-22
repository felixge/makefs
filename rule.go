package makefs

import (
	"net/http"
	"os"
	gopath "path"
	"regexp"
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
		dir = ""
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
		sourceFile, err := fs.Open(sourcePath)
		if err != nil {
			if os.IsNotExist(err) {
				return nil, nil
			}
			return nil, err
		}
		defer sourceFile.Close()

		sourceStat, err := sourceFile.Stat()
		if err != nil {
			return nil, err
		}

		sources = append(sources, &Source{
			path: sourcePath,
			fs:   fs,
			stat: sourceStat,
		})
	}

	return sources, nil
}

func (r *rule) resolveTargetPath(sourcePath string, fs http.FileSystem) (string, error) {
	targetPath := r.target
	if isPattern(targetPath) {
		var (
			stem = ""
			dir = ""
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
	} else if sources == nil {
		return "", nil
	}

	return targetPath, nil
}

func findStem(path string, pattern string) (string, string) {
	dir := gopath.Dir(path)
	name := gopath.Base(path)

	pattern = regexp.QuoteMeta(pattern)
	pattern = "^" + strings.Replace(pattern, "%", "(.+)", 1) + "$"

	matcher, err := regexp.Compile(pattern)
	if err != nil {
		panic("unreachable")
	}

	match := matcher.FindStringSubmatch(name)
	if len(match) != 2 {
		return "", ""
	}

	return match[1], dir
}

func insertStem(pattern string, stem string) string {
	return strings.Replace(pattern, "%", stem, -1)
}

func isPattern(str string) bool {
	return strings.Contains(str, "%")
}

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

func (r *rule) resolveTargetPath(sourcePath string) string {
	if r.sources[0] == sourcePath {
		return r.target
	}

	return ""
}

func isPattern(str string) bool {
	return strings.Contains(str, "%")
}

func isAbs(str string) bool {
	return gopath.IsAbs(str)
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

func isGlob(str string) bool {
	return strings.Contains(str, "*")
}

func expand(pattern string, fs http.FileSystem) ([]*Source, error) {
	return nil, nil
}

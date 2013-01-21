package makefs

import (
	"net/http"
	"strings"
	gopath "path"
	"regexp"
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
	if targetPath != r.target {
		return nil, nil
	}

	sourcePath := r.sources[0]
	source := &Source{path: sourcePath, fs: fs}
	sources := []*Source{source}

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

func expand(pattern string, fs http.FileSystem) ([]*Source, error) {
	return nil, nil
}

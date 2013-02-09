package makefs

import (
	"net/http"
	"os"
	gopath "path"
	"regexp"
	"strings"
)

// Note: Go has filepath.Glob(), but unfortunately it does not operate on a
// virtual / http.FileSystem. It also does not have support for the double star
// which is regrettable.

// Glob implements a limited path expansion method. At this point only the *
// (star) wildcard is implemented, but support for the recursive ** (double
// star) wildcard is planned. Patches for other commonly found placeholders
// would be welcomed.
func Glob(pattern string, fs http.FileSystem) (map[string]os.FileInfo, error) {
	var (
		results   = map[string]os.FileInfo{}
		basePaths = []string{"/"}
		parts     = parseGlob(pattern)
	)

	// iterate over all '/' separated parts of our pattern
	for p, part := range parts {
		newBasePaths := make([]string, 0, len(basePaths))

		// iterate over all active basePaths, and match part against the children
		for _, basePath := range basePaths {
			dir, err := fs.Open(basePath)
			if os.IsNotExist(err) {
				continue
			} else if err != nil {
				return nil, err
			}
			defer dir.Close()

			stats, err := dir.Readdir(-1)
			if err != nil {
				return nil, err
			}

			for _, stat := range stats {
				name := stat.Name()
				if !part.MatchString(name) {
					continue
				}

				path := gopath.Join(basePath, name)
				if last := p+1 == len(parts); last {
					results[path] = stat
				} else if stat.IsDir() {
					newBasePaths = append(newBasePaths, path)
				}
			}
		}

		basePaths = newBasePaths
	}
	return results, nil
}

type stringMatcher interface {
	MatchString(name string) bool
}

type staticPart string

func (p staticPart) MatchString(name string) bool {
	return string(p) == name
}

func parseGlob(pattern string) []stringMatcher {
	var (
		results = []stringMatcher{}
		// the first character is always '/', we skip that
		parts = strings.Split(pattern[1:], "/")
	)

	for _, part := range parts {
		results = append(results, parseGlobPart(part))
	}
	return results
}

// parseGlobPart takes a '/' seperated part element of a pattern and returns
// a stringMatcher that recognized this pattern.
//
// BUG: escaped stars are not interpreted as stars, but are matched as '\*',
// rather than '*' by the resulting stringMatcher.
func parseGlobPart(part string) stringMatcher {
	// Offsets for all the stars found.
	stars := []int{}

	// Find all the stars in the part, ignore escaped stars.
	var prev uint8
	for i := 0; i < len(part); i++ {
		char := part[i]

		star := (char == '*' && prev != '\\')
		if star {
			stars = append(stars, i)
		}

		prev = char
	}

	// No stars means this part is static, can be compared more efficiently.
	if len(stars) == 0 {
		return staticPart(part)
	}

	// Create a regexp pattern according to the stars we found.
	// Example: `*.txt` -> `.+\.txt`
	start := 0
	pattern := "^"
	for i, offset := range stars {
		pattern += regexp.QuoteMeta(part[start:offset])
		pattern += ".+"
		start = offset + 1

		if last := i+1 == len(stars); last {
			pattern += regexp.QuoteMeta(part[start:]) + "$"
		}
	}

	// We use MustCompile because our regexp should always be valid.
	return regexp.MustCompile(pattern)
}

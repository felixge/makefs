package makefs

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"io"
	"net/http"
	"os"
	gopath "path"
	"testing"
)

// RuleFsTests is a simple list that declares the tests ruleFs should pass.
var RuleFsTests = []struct {
	Name   string
	Rule   *rule
	Checks []FsChecker
}{
	{
		Name: "path to path",
		Rule: &rule{
			target:  "/foo.sha1",
			sources: []string{"/foo.txt"},
			recipe:  Sha1Recipe,
		},
		Checks: []FsChecker{
			&ReadCheck{"/foo.sha1", "781b3017fe23bf261d65a6c3ed4d1af59dea790f"},
			&StatCheck{path: "/foo.sha1", size: 40, name: "foo.sha1"},
			//&ExistCheck{"/foo.txt", true},
			//&ExistCheck{"/foo.sha1", true},
		},
	},
}

// TestRuleFs_Tests executes the RuleFsTests declared above.
func TestRuleFs_Tests(t *testing.T) {
	for _, test := range RuleFsTests {
		t.Logf("test: %s", test.Name)

		fs, err := newRuleFs(http.Dir(fixturesDir), test.Rule)
		if err != nil {
			t.Errorf("could not create fs: %s", err)
			continue
		}

		for _, check := range test.Checks {
			if err := check.Check(fs); err != nil {
				t.Errorf("%s in %#v", err, check)
			}
		}
	}
}

// Sha1Recipe takes one source file and produces the sha1 sum as the target.
var Sha1Recipe = func(t *Task) error {
	hash := sha1.New()
	if _, err := io.Copy(hash, t.Source()); err != nil {
		return err
	}

	hex := fmt.Sprintf("%x", hash.Sum(nil))

	if _, err := io.WriteString(t.Target(), hex); err != nil {
		return err
	}
	return nil
}

// FsChecker is a simple interface for checking things inside a http.FileSystem
type FsChecker interface {
	Check(fs http.FileSystem) error
}

// ReadCheck checks the result of reading a file from the given path.
type ReadCheck struct {
	path     string
	expected string
}

func (check *ReadCheck) Check(fs http.FileSystem) error {
	file, err := fs.Open(check.path)
	if err != nil {
		return err
	}

	buf := new(bytes.Buffer)
	if _, err := io.Copy(buf, file); err != nil {
		return err
	}

	got := buf.String()
	if got != check.expected {
		return fmt.Errorf("unexpected: %#v", got)
	}
	return nil
}

// ExistCheck checks if a file is present or not. It verifies this by calling
// Open() on the path, as well as Readdir() on the parent directory.
type ExistCheck struct {
	path        string
	shouldExist bool
}

func (check *ExistCheck) Check(fs http.FileSystem) error {
	_, err := fs.Open(check.path)

	existErr := os.IsNotExist(err)
	if check.shouldExist {
		if existErr {
			return fmt.Errorf("should exist, but does not")
		} else if err != nil {
			return fmt.Errorf("should exist, but raises error: %#v", err)
		}
	} else {
		if err == nil {
			return fmt.Errorf("should not exist, but does")
		} else if !existErr {
			return fmt.Errorf("should not exist, but raises unexpected err: %#v", err)
		}
	}

	dirPath := gopath.Dir(check.path)
	dirFile, err := fs.Open(dirPath)
	if err != nil {
		return err
	}

	stats, err := dirFile.Readdir(0)
	if err != nil {
		return err
	}

	name := gopath.Base(check.path)
	listed := false

	for _, stat := range stats {
		if stat.Name() == name {
			listed = true
			break
		}
	}

	if !check.shouldExist && listed {
		return fmt.Errorf("should not be listed by Readdir, but is")
	} else if check.shouldExist && !listed {
		return fmt.Errorf("should be listed by Readdir, but is not")
	}
	return nil
}

type StatCheck struct {
	path string
	size int64
	name string
}

func (check *StatCheck) Check(fs http.FileSystem) error {
	file, err := fs.Open(check.path)
	if err != nil {
		return err
	}

	stat, err := file.Stat()
	if err != nil {
		return err
	}

	gotSize := stat.Size()
	if gotSize != check.size {
		return fmt.Errorf("unexpected size: %d", gotSize)
	}

	gotName := stat.Name()
	if gotName != check.name {
		return fmt.Errorf("unexpected name: %s", gotName)
	}

	return nil
}

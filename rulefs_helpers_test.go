package makefs

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"io"
	"net/http"
	"os"
	gopath "path"
)

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

// CatRecipe takes several source files and concatenates them into one target.
var CatRecipe = func(t *Task) error {
	for _, source := range t.Sources() {
		if _, err := io.Copy(t.Target(), source); err != nil {
			return err
		}
	}
	return nil
}

// Checker is a simple interface for checking things inside a http.FileSystem
type Checker interface {
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
	file, err := fs.Open(check.path)
	if file != nil {
		defer file.Close()
	}

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
	defer dirFile.Close()

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

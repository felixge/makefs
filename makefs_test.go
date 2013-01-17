package makefs

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// Get name/dir of this source file
var (
	_, __filename, _, _ = runtime.Caller(0)
	__dirname           = filepath.Dir(__filename)
	fixturesDir         = __dirname + "/fixtures"
)

func TestMakeFs_Make(t *testing.T) {
	fs := NewFs(http.Dir(fixturesDir))
	fs.Make("%.sha1", "%.txt", func(t *Task) error {
		hash := sha1.New()
		if _, err := io.Copy(hash, t.Source()); err != nil {
			return err
		}
		if _, err := t.Target().Write(hash.Sum(nil)); err != nil {
			return err
		}
		return nil
	})

	file, err := fs.Open("/foo.sha1")
	if err != nil {
		t.Fatal(err)
	}

	buf := new(bytes.Buffer)
	if _, err := io.Copy(buf, file); err != nil {
		t.Fatal(err)
	}

	expected := "781b3017fe23bf261d65a6c3ed4d1af59dea790f"
	got := fmt.Sprintf("%x", buf)
	if got != expected {
		fmt.Printf("unexpected: %s\n", got)
	}
}

func TestMakeFs_ExecMake(t *testing.T) {
	fs := NewFs(http.Dir(fixturesDir))
	fs.ExecMake("%.sha1", "%.txt", "cut", "-c", "1-11")

	file, err := fs.Open("/foo.sha1")
	if err != nil {
		t.Fatal(err)
	}

	buf := new(bytes.Buffer)
	if _, err := io.Copy(buf, file); err != nil {
		t.Fatal(err)
	}

	expected := "May the foo\n"
	got := buf.String()
	if got != expected {
		fmt.Printf("unexpected: %s\n", got)
	}
}

func TestMakeFs_SubFs(t *testing.T) {
	fs := NewFs(http.Dir(fixturesDir))
	subFs := fs.SubFs("/sub")

	if _, err := fs.Open("/a.txt"); !os.IsNotExist(err) {
		t.Fatal("unexpected error", err)
	}

	if _, err := subFs.Open("/a.txt"); err != nil {
		t.Fatal("unexpected error", err)
	}
}

func TestRuleFs_Read(t *testing.T) {
	fs := strongRuleFs()
	file, err := fs.Open("/foo.strong")
	if err != nil {
		t.Fatal(err)
	}

	buf := &bytes.Buffer{}
	if _, err := io.Copy(buf, file); err != nil {
		t.Fatal(err)
	}

	expected := "<strong>May the foo be with you.\n</strong>"
	if buf.String() != expected {
		t.Fatalf("unexpected result: %s", buf)
	}
}

func TestRuleFs_Readdir(t *testing.T) {
	fs := strongRuleFs()
	dir, err := fs.Open("/")
	if err != nil {
		t.Fatal(err)
	}

	stats, err := dir.Readdir(0)
	if err != nil {
		t.Fatal(err)
	}

	expected := map[string]int64{
		"foo.strong": 42,
		"sub":        -1,
	}

	for _, stat := range stats {
		name := stat.Name()
		if name[0] == '.' {
			continue
		}

		size := stat.Size()
		if expectedSize, ok := expected[name]; !ok {
			t.Errorf("unexpected file: %s", name)
			continue
		} else if expectedSize == -1 {
			// -1 means don't check the size on this file (used for dirs)
		} else if expectedSize != size {
			t.Errorf("got size: %d, expected: %d for: %s", size, expectedSize, name)
		}

		delete(expected, name)
	}

	for name, _ := range expected {
		t.Errorf("missing file: %s", name)
	}
}

func TestRuleFs_Stat(t *testing.T) {
	fs := strongRuleFs()
	file, err := fs.Open("/foo.strong")
	if err != nil {
		t.Fatal(err)
	}

	stat, err := file.Stat()
	if err != nil {
		t.Fatal(err)
	}

	name := stat.Name()
	if name != "foo.strong" {
		t.Errorf("bad name: %s", name)
	}
}

var FindStemTests = []struct {
	Str     string
	Pattern string
	Expect  string
}{
	{Str: "foo.txt", Pattern: "%.txt", Expect: "foo"},
	{Str: "foo.txt", Pattern: "foo.%", Expect: "txt"},
	{Str: "a.b.c", Pattern: "a.%.c", Expect: "b"},
	{Str: "foo.txt", Pattern: ".txt", Expect: ""},
	{Str: "/", Pattern: "%.txt", Expect: ""},
}

func Test_findStem(t *testing.T) {
	for _, test := range FindStemTests {
		stem := findStem(test.Str, test.Pattern)
		if stem != test.Expect {
			t.Errorf("expected stem: %s, got: %s (%+v)", test.Expect, stem, test)
		}
	}
}

func strongRuleFs() http.FileSystem {
	fs := &ruleFs{
		parent: http.Dir(fixturesDir),
		rule: &rule{
			targets: []string{"%.strong"},
			sources: []string{"%.txt"},
			recipe: func(task *Task) error {
				target := task.Target()
				source := task.Source()

				if _, err := target.Write([]byte("<strong>")); err != nil {
					return err
				}
				if _, err := io.Copy(target, source); err != nil {
					return err
				}
				if _, err := target.Write([]byte("</strong>")); err != nil {
					return err
				}

				return nil
			},
		},
	}
	return fs
}

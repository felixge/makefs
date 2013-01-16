package makefs

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"runtime"
	"testing"
)

// Get name/dir of this source file
var (
	_, __filename, _, _ = runtime.Caller(0)
	__dirname           = filepath.Dir(__filename)
)

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

	expected := "<strong>May the foo be with you!\n</strong>"
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

	fmt.Printf("dir: %#v\n", dir)

	stats, err := dir.Readdir(0)
	if err != nil {
		t.Fatal(err)
	}

	expected := map[string]int64{
		"foo.strong": -1,
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
		} else if expectedSize != size {
			t.Errorf("got size: %d, expected: %d for: %s", size, expectedSize, name)
		}

		delete(expected, name)
	}

	for name, _ := range expected {
		t.Errorf("missing file: %s", name)
	}
}

func ExampleStat() {
	fs := strongRuleFs()
	file, err := fs.Open("/foo.strong")
	if err != nil {
		fmt.Printf("Open: %#v", err)
		return
	}

	stat, err := file.Stat()
	if err != nil {
		fmt.Printf("Err: %#v", err)
		return
	}

	fmt.Print(stat)

	// Output:
	// fuck
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
		parent: http.Dir(__dirname + "/fixtures"),
		rule: &rule{
			targets: []string{"%.strong"},
			sources: []string{"%.txt"},
			recipe: func(task *task) error {
				target := task.Target()
				if _, err := target.Write([]byte("<strong>")); err != nil {
					return err
				}
				if _, err := io.Copy(target, task.Source()); err != nil {
					return err
				}
				if _, err := target.Write([]byte("</strong>")); err != nil {
					return err
				}
				target.Close()
				return nil
			},
		},
	}
	return fs
}

package makefs

import (
	"bytes"
	"io"
	"net/http"
	"testing"
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
	{Str: "/pages", Pattern: "/public/%.html", Expect: ""},
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
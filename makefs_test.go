package makefs

import (
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
)

func ExampleOpen() {
	fs := strongRuleFs()
	file, err := fs.Open("/foo.strong")
	if err != nil {
		fmt.Printf("Open: %#v", err)
		return
	}

	io.Copy(os.Stdout, file)

	// Output:
	// <strong>May the foo be with you!
	// </strong>
}

func ExampleReaddir() {
	fs := strongRuleFs()
	dir, err := fs.Open("/")
	if err != nil {
		fmt.Printf("Open: %#v", err)
		return
	}

	stats, err := dir.Readdir(0)
	if err != nil {
		fmt.Printf("Readdir: %#v", err)
		return
	}

	for _, stat := range stats {
		name := stat.Name()
		if name[0] == '.' {
			continue
		}
		fmt.Printf("%s\n", name)
	}

	// Output:
	// foo.strong
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
			recipe: func(task *task) {
				target := task.Target()
				target.Write([]byte("<strong>"))
				io.Copy(target, task.Source())
				target.Write([]byte("</strong>"))
				target.Close()
			},
		},
	}
	return fs
}

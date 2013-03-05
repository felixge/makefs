package makefs

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"sync"
	"testing"
	"time"
)

const FooSha1 = "781b3017fe23bf261d65a6c3ed4d1af59dea790f"

var RuleFsTests = []struct {
	Name   string
	Rule   *rule
	Checks []Checker
}{
	{
		Name: "1 abs target, 1 abs source",
		Rule: &rule{
			target:  "/foo.sha1",
			sources: []string{"/foo.txt"},
			recipe:  Sha1Recipe,
		},
		Checks: []Checker{
			&ReadCheck{"/foo.sha1", FooSha1},
			&StatCheck{path: "/foo.sha1", size: 40, name: "foo.sha1"},
			&ExistCheck{"/foo.txt", true},
			&ExistCheck{"/foo.sha1", true},
			&ExistCheck{"/does-not-exist.sha1", false},
		},
	},
	{
		Name: "1 abs target, 1 abs source (in sub dir)",
		Rule: &rule{
			target:  "/%.sha1",
			sources: []string{"/sub/%.txt"},
			recipe:  Sha1Recipe,
		},
		Checks: []Checker{
			&ExistCheck{"/a.sha1", true},
		},
	},
	{
		Name: "1 abs target (creating new dir), 1 abs source",
		Rule: &rule{
			target:  "/some/new/dir/foo.sha1",
			sources: []string{"/foo.txt"},
			recipe:  Sha1Recipe,
		},
		Checks: []Checker{
			&ReadCheck{"/some/new/dir/foo.sha1", FooSha1},
			&ExistCheck{"/some/new/dir/foo.sha1", true},
			&ExistCheck{"/some", true},
			&ExistCheck{"/some/new", true},
		},
	},
	{
		Name: "1 abs target, 2 abs sources",
		Rule: &rule{
			target:  "/yin-yang.txt",
			sources: []string{"/yin.txt", "/yang.txt"},
			recipe:  CatRecipe,
		},
		Checks: []Checker{
			&ReadCheck{"/yin-yang.txt", "yin\nyang\n"},
			&StatCheck{path: "/yin-yang.txt", size: 9, name: "yin-yang.txt"},
		},
	},
	{
		Name: "1 pattern target, 1 pattern source",
		Rule: &rule{
			target:  "%.sha1",
			sources: []string{"%.txt"},
			recipe:  Sha1Recipe,
		},
		Checks: []Checker{
			&ReadCheck{"/foo.sha1", FooSha1},
			&ReadCheck{"/sub/a.sha1", "1fb217f037ece180e41303a2ac55aed51e3e473f"},
			&ExistCheck{"/foo.txt", true},
			&ExistCheck{"/foo.sha1", true},
			&ExistCheck{"/sub/a.txt", true},
			&ExistCheck{"/sub/a.sha1", true},
		},
	},
	{
		Name: "1 pattern target, 1 pattern source, 1 abs source",
		Rule: &rule{
			target:  "%.txt",
			sources: []string{"%.txt", "/yang.txt"},
			recipe:  CatRecipe,
		},
		Checks: []Checker{
			&ReadCheck{"/yin.txt", "yin\nyang\n"},
			&ReadCheck{"/yang.txt", "yang\nyang\n"},
			&ExistCheck{"/yin.txt", true},
			&ExistCheck{"/yang.txt", true},
		},
	},
	{
		Name: "1 abs target, 1 wildcard source",
		Rule: &rule{
			target:  "/all.txt",
			sources: []string{"/wild/*.txt"},
			recipe:  CatRecipe,
		},
		Checks: []Checker{
			&ReadCheck{"/all.txt", "1\n2\n3\n"},
		},
	},
}

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

func TestRuleFs_RecipeCaching(t *testing.T) {
	countLock := new(sync.Mutex)
	count := 0

	rule := &rule{
		target:  "%.sha1",
		sources: []string{"%.txt"},
		recipe: func(t *Task) error {
			countLock.Lock()
			count++
			countLock.Unlock()

			return Sha1Recipe(t)
		},
	}

	fs, err := newRuleFs(http.Dir(fixturesDir), rule)
	if err != nil {
		t.Fatal(err)
	}

	readTarget := func() {
		file, err := fs.Open("/foo.sha1")
		if err != nil {
			t.Fatal(err)
		}
		defer file.Close()

		buf := new(bytes.Buffer)
		if _, err := io.Copy(buf, file); err != nil {
			t.Error(err)
		}
		if buf.String() != FooSha1 {
			t.Error(buf.String())
		}
	}

	for i := 0; i < 3; i++ {
		readTarget()
		if count != 1 {
			t.Error(count)
		}
	}

	sourcePath := fixturesDir + "/foo.txt"
	stat, err := os.Stat(sourcePath)
	if err != nil {
		t.Fatal(err)
	}

	atime := time.Now()
	modtime := stat.ModTime().Add(1 * time.Second)
	if err := os.Chtimes(sourcePath, atime, modtime); err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 3; i++ {
		readTarget()
		if count != 2 {
			t.Error(count)
		}
	}
}

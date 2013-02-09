package makefs

import (
	"net/http"
	"testing"
)

var GlobTests = []struct {
	Pattern string
	Expected []string
}{
	{
		Pattern: "/foo.txt",
		Expected: []string{"/foo.txt"},
	},
	{
		Pattern: "/wild/*.txt",
		Expected: []string{"/wild/1.txt", "/wild/2.txt", "/wild/3.txt"},
	},
	{
		Pattern: "/wild/1.*xt",
		Expected: []string{"/wild/1.txt"},
	},
	{
		Pattern: "/wild/*.*xt",
		Expected: []string{"/wild/1.txt", "/wild/2.txt", "/wild/3.txt"},
	},
	{
		Pattern: "/wild/*/*.txt",
		Expected: []string{"/wild/a/4.txt", "/wild/a/5.txt", "/wild/b/6.txt"},
	},
	//{
		//Pattern: "/wild/**.txt",
		//Expected: []string{
			//"/wild/1.txt",
			//"/wild/2.txt",
			//"/wild/3.txt",
			//"/wild/a/4.txt",
			//"/wild/a/5.txt",
			//"/wild/b/6.txt",
		//},
	//},
}

func TestGlob(t *testing.T) {
	dir := http.Dir(fixturesDir)

	for _, test := range GlobTests {
		t.Logf("testing: %s", test.Pattern)

		stats, err := Glob(test.Pattern, dir)
		if err != nil {
			t.Error(err)
			continue
		}

		if got, expected := len(stats), len(test.Expected); got != expected {
			t.Errorf("%d instead of %d stats", got, expected)
		}

		for _, path := range test.Expected {
			if _, ok := stats[path]; !ok {
				t.Errorf("missing result: %s", path)
			}
		}

		for path, _ := range stats {
			found := false
			for _, expectedPath := range test.Expected {
				if path == expectedPath {
					found = true
					break
				}
			}

			if !found {
				t.Errorf("unexpected result: %s", path)
			}
		}
	}
}

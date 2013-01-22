package makefs

import (
	"testing"
)

var FindStemTests = []struct {
	Path       string
	Pattern    string
	ExpectDir  string
	ExpectStem string
}{
	{"/foo.txt", "%.txt", "/", "foo"},
	{"/bar/foo.txt", "%.txt", "/bar", "foo"},
	{"/bar/foo.txt", "prefix-%.txt", "", ""},
	{"/bar/prefix-foo.txt", "prefix-%.txt", "/bar", "foo"},
	{"/bar/foo.txt", "%.txt-suffix", "", ""},
	{"/bar/foo.txt-suffix", "%.txt-suffix", "/bar", "foo"},
	{"/foo/bar/some.txt", "bar/%.txt", "/foo/bar", "some"},
	{"/foo/not/some.txt", "bar/%.txt", "", ""},
}

func Test_findStem(t *testing.T) {
	for _, test := range FindStemTests {
		stem, dir := findStem(test.Path, test.Pattern)
		if stem != test.ExpectStem {
			t.Errorf("expected stem: %s, got: %s (%+v)", test.ExpectStem, stem, test)
		}
		if dir != test.ExpectDir {
			t.Errorf("expected base: %s, got: %s (%+v)", test.ExpectDir, dir, test)
		}
	}
}

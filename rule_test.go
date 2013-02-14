package makefs

import (
	"testing"
)

var FindStemTests2 = []struct {
	Path       string
	Pattern    string
	ExpectStem string
}{
	{"/foo.txt", "/%.txt", "foo"},
	{"/foo.txt", "%.txt", "/foo"},
	{"/public/2013/03/24/fun.html", "/public/%.html", "2013/03/24/fun"},
}

func Test_findStem(t *testing.T) {
	for _, test := range FindStemTests2 {
		stem := findStem(test.Path, test.Pattern)
		if stem != test.ExpectStem {
			t.Errorf("expected stem: %#v, got: %#v (%+v)", test.ExpectStem, stem, test)
		}
	}
}

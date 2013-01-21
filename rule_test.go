package makefs

import (
	"testing"
)

var FindStemTests = []struct {
	Str     string
	Pattern string
	Expect  string
}{
	{Str: "/foo.txt", Pattern: "%.txt", Expect: "/foo"},
	{Str: "/foo.txt", Pattern: "/%.txt", Expect: "foo"},
	{Str: "/bar/foo.txt", Pattern: "/%.txt", Expect: "bar/foo"},
	{Str: "/foo.txta", Pattern: "%.txt", Expect: ""},
	{Str: "/foo-bar.txt", Pattern: "%-bar.txt", Expect: "/foo"},
	{Str: "/pages", Pattern: "/public/%.html", Expect: ""},
	{Str: "/foo.txt", Pattern: ".txt", Expect: ""},
	{Str: "/", Pattern: "%.txt", Expect: ""},
	{Str: "/bar/foo.txt", Pattern: "/bar/%.txt", Expect: "foo"},
}

func Test_findStem(t *testing.T) {
	for _, test := range FindStemTests {
		stem := findStem(test.Str, test.Pattern)
		if stem != test.Expect {
			t.Errorf("expected stem: %s, got: %s (%+v)", test.Expect, stem, test)
		}
	}
}

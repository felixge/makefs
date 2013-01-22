package makefs

import (
	"net/http"
	"testing"
)

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
			&ReadCheck{"/foo.sha1", "781b3017fe23bf261d65a6c3ed4d1af59dea790f"},
			&StatCheck{path: "/foo.sha1", size: 40, name: "foo.sha1"},
			&ExistCheck{"/foo.txt", true},
			&ExistCheck{"/foo.sha1", true},
			&ExistCheck{"/does-not-exist.sha1", false},
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
			&ReadCheck{"/foo.sha1", "781b3017fe23bf261d65a6c3ed4d1af59dea790f"},
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


package makefs

import (
	"net/http"
	"os"
	"testing"
)

func TestIncludeFs_Open(t *testing.T) {
	fs := NewIncludeFs(http.Dir(fixturesDir), "/sub/", "/sub3/")

	// included, should exist
	if file, err := fs.Open("/sub/a.txt"); err != nil {
		t.Fatal(err)
	} else {
		file.Close()
	}

	// included, should exist
	if file, err := fs.Open("/sub3/sub3.txt"); err != nil {
		t.Fatal(err)
	} else {
		file.Close()
	}

	// not included, should not exist
	if _, err := fs.Open("/sub2/sub2.txt"); !os.IsNotExist(err) {
		t.Fatal(err)
	}

	// check readdir, should only list the included dirs
	file, err := fs.Open("/")
	if err != nil {
		t.Fatal(err)
	}

	stats, err := file.Readdir(-1)
	if err != nil {
		t.Fatal(err)
	}

	expected := map[string]bool{"sub": true, "sub3": true}
	if len(stats) != len(expected) {
		t.Fatal(len(stats))
	}

	for _, stat := range stats {
		if name := stat.Name(); !expected[name] {
			t.Errorf("Unexpected: %s", name)
		}
	}
}

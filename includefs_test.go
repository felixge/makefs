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
}

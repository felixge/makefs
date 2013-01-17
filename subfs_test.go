package makefs

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"testing"
)

func TestSubFs_Open(t *testing.T) {
	fs := NewSubFs(http.Dir(fixturesDir), "/sub")
	file, err := fs.Open("/a.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	buf := new(bytes.Buffer)
	if _, err := io.Copy(buf, file); err != nil {
		t.Fatal(err)
	}

	expected := "a.txt in subfs\n"
	got := buf.String()
	if got != expected {
		t.Fatalf("unexpected: %s", got)
	}
}

func TestSubFs_Open_Jail(t *testing.T) {
	fs := NewSubFs(http.Dir(fixturesDir), "/sub")
	_, err := fs.Open("../foo.txt")
	if err != os.ErrPermission {
		t.Fatal("wrong error", err)
	}
}

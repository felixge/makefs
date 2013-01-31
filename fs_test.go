package makefs

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
)

func TestFs_Make(t *testing.T) {
	fs := NewFs(http.Dir(fixturesDir))
	fs.Make("%.sha1", []string{"%.txt"}, func(t *Task) error {
		hash := sha1.New()
		if _, err := io.Copy(hash, t.Source()); err != nil {
			return err
		}
		if _, err := t.Target().Write(hash.Sum(nil)); err != nil {
			return err
		}
		return nil
	})

	file, err := fs.Open("/foo.sha1")
	if err != nil {
		t.Fatal(err)
	}

	buf := new(bytes.Buffer)
	if _, err := io.Copy(buf, file); err != nil {
		t.Fatal(err)
	}

	expected := "781b3017fe23bf261d65a6c3ed4d1af59dea790f"
	got := fmt.Sprintf("%x", buf)
	if got != expected {
		fmt.Printf("unexpected: %s\n", got)
	}
}

func TestFs_ExecMake(t *testing.T) {
	fs := NewFs(http.Dir(fixturesDir))
	fs.ExecMake("%.sha1", "%.txt", "cut", "-c", "1-11")

	file, err := fs.Open("/foo.sha1")
	if err != nil {
		t.Fatal(err)
	}

	buf := new(bytes.Buffer)
	if _, err := io.Copy(buf, file); err != nil {
		t.Fatal(err)
	}

	expected := "May the foo\n"
	got := buf.String()
	if got != expected {
		fmt.Printf("unexpected: %s\n", got)
	}
}

func TestFs_SubFs(t *testing.T) {
	fs := NewFs(http.Dir(fixturesDir))
	subFs := fs.SubFs("/sub")

	if _, err := fs.Open("/a.txt"); !os.IsNotExist(err) {
		t.Fatal("unexpected error", err)
	}

	if _, err := subFs.Open("/a.txt"); err != nil {
		t.Fatal("unexpected error", err)
	}
}

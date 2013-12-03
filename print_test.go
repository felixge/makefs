package makefs

import (
	"bytes"
	"go/parser"
	goprinter "go/printer"
	"go/token"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"testing"
)

func TestFs_Fprint(t *testing.T) {
	fs := http.Dir(fixturesDir)
	output := new(bytes.Buffer)

	if err := Fprint(output, fs, "testpackage", "testVar"); err != nil {
		t.Fatal(err)
	}

	fset := &token.FileSet{}
	filename := "some.go"
	src := output.Bytes()

	file, err := parser.ParseFile(fset, filename, src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}

	pretty := new(bytes.Buffer)
	if err := (&goprinter.Config{Tabwidth: 8}).Fprint(pretty, fset, file); err != nil {
		t.Fatal(err)
	}

	diffOutput, err := diff(output.Bytes(), pretty.Bytes())
	if err != nil {
		t.Fatal(err)
	}

	if len(diffOutput) > 0 {
		t.Fatalf("output not gofmt formatted:\n%s", diffOutput)
	}
}

// from go's gofmt.go (MIT license)
func diff(b1, b2 []byte) (data []byte, err error) {
	f1, err := ioutil.TempFile("", "gofmt")
	if err != nil {
		return
	}
	defer os.Remove(f1.Name())
	defer f1.Close()

	f2, err := ioutil.TempFile("", "gofmt")
	if err != nil {
		return
	}
	defer os.Remove(f2.Name())
	defer f2.Close()

	f1.Write(b1)
	f2.Write(b2)

	data, err = exec.Command("diff", "-u", f1.Name(), f2.Name()).CombinedOutput()
	if len(data) > 0 {
		// diff exits with a non-zero status when the files don't match.
		// Ignore that failure as long as we get output.
		err = nil
	}
	return
}

package makefs

import (
	"bytes"
	"fmt"
	"net/http"
	"testing"
)

func TestFs_Fprint(t *testing.T) {
	fs := http.Dir(fixturesDir)
	buf := new(bytes.Buffer)

	if err := Fprint(buf, fs, "testpackage", "TestVar"); err != nil {
		t.Fatal(err)
	}

	fmt.Printf("%s\n", buf.Bytes())
}

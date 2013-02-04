package makefs

import (
	"io"
	"testing"
)

func TestMemoryFile_Read(t *testing.T) {
	data := "12345"
	file := &MemoryFile{data: data}
	buf := make([]byte, len(data))

	if n, err := file.Read(buf[0:2]); err != nil {
		t.Fatal(err)
	} else if n != 2 {
		t.Fatal(n)
	} else if string(buf[0:2]) != data[0:2] {
		t.Fatal(string(buf[0:2]))
	}

	if n, err := file.Read(buf[2:]); err != nil {
		t.Fatal(err)
	} else if n != 3 {
		t.Fatal(n)
	} else if string(buf[2:]) != data[2:] {
		t.Fatal(string(buf[2:]))
	}

	if n, err := file.Read(buf); err != io.EOF {
		t.Fatal(err)
	} else if n != 0 {
		t.Fatal(n)
	}
}

func TestMemoryFile_Close(t *testing.T) {
	file := &MemoryFile{data: "foobar"}
	buf := make([]byte, 100)

	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	if n, err := file.Read(buf); err == nil {
		t.Fatal(err)
	} else if n != 0 {
		t.Fatal(err)
	}

	if err := file.Close(); err == nil {
		t.Fatal(err)
	}
}

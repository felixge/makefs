package makefs

import (
	"testing"
)

func TestBroadcast(t *testing.T) {
	broadcast := newBroadcast()
	broadcast.Write([]byte("abc"))

	a := broadcast.Client()
	aData := make([]byte, 512)
	if n, err := a.Read(aData); err != nil {
		t.Fatal(err)
	} else if n != 3 {
		t.Fatalf("unexpected n: %d", n)
	} else if (string(aData[0:n]) != "abc") {
		t.Fatalf("unexpected data: %s", aData[0:n])
	}

	broadcast.Write([]byte("de"))

	b := broadcast.Client()
	bData := make([]byte, 512)
	if n, err := b.Read(bData); err != nil {
		t.Fatal(err)
	} else if n != 5 {
		t.Fatalf("unexpected n: %d", n)
	} else if (string(bData[0:n]) != "abcde") {
		t.Fatalf("unexpected data: %s", bData[0:n])
	}

	go func() {
		broadcast.Write([]byte("fghi"))
	}()

	if n, err := b.Read(bData); err != nil {
		t.Fatal(err)
	} else if n != 4 {
		t.Fatalf("unexpected n: %d", n)
	} else if (string(bData[0:n]) != "fghi") {
		t.Fatalf("unexpected data: %s", aData[0:n])
	}

	if n, err := a.Read(aData); err != nil {
		t.Fatal(err)
	} else if n != 6 {
		t.Fatalf("unexpected n: %d", n)
	} else if (string(aData[0:n]) != "defghi") {
		t.Fatalf("unexpected data: %s", aData[0:n])
	}


	//b := broadcast.Client()
	//bb := make([]byte, 512)
	//if n, err := b.Read(bb); err != nil {
		//t.Fatal(err)
	//} else if n != 5 {
		//t.Fatalf("unexpected n: %d", n)
	//} else if (string(bb[0:n]) != "abcde") {
		//t.Fatalf("unexpected data: %s", bb[0:n])
	//}


}

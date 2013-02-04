package makefs

import (
	"io"
	"os"
	"testing"
	"time"
)

func TestMemoryFile_Read(t *testing.T) {
	data := "12345"
	file := &MemoryFile{Data: data}
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

func TestMemoryFile_Seek_SET(t *testing.T) {
	data := "12345"
	file := &MemoryFile{Data: data}
	buf := make([]byte, len(data))

	readByte2To4 := func() {
		if n, err := file.Seek(2, os.SEEK_SET); err != nil {
			t.Fatal(err)
		} else if n != 2 {
			t.Fatal(n)
		}

		if n, err := file.Read(buf[0:2]); err != nil {
			t.Fatal(err)
		} else if n != 2 {
			t.Fatal(n)
		} else if str := string(buf[0:2]); str != data[2:4] {
			t.Fatal(str)
		}
	}

	// 2x - makes sure SEEK_SET is absolute
	readByte2To4()
	readByte2To4()

	// negative seek should give error
	if _, err := file.Seek(-1, os.SEEK_SET); err == nil {
		t.Fatal(err)
	}
}

func TestMemoryFile_Seek_CUR(t *testing.T) {
	data := "12345"
	file := &MemoryFile{Data: data}
	buf := make([]byte, len(data))

	if n, err := file.Seek(1, os.SEEK_CUR); err != nil {
		t.Fatal(err)
	} else if n != 1 {
		t.Fatal(n)
	}

	if n, err := file.Read(buf[0:2]); err != nil {
		t.Fatal(err)
	} else if n != 2 {
		t.Fatal(n)
	} else if str := string(buf[0:2]); str != data[1:3] {
		t.Fatal(str)
	}

	if n, err := file.Seek(1, os.SEEK_CUR); err != nil {
		t.Fatal(err)
	} else if n != 4 {
		t.Fatal(n)
	}

	if n, err := file.Read(buf[0:2]); err != nil {
		t.Fatal(err)
	} else if n != 1 {
		t.Fatal(n)
	} else if str := string(buf[0:n]); str != data[4:5] {
		t.Fatal(str)
	}

	if n, err := file.Seek(-1, os.SEEK_CUR); err != nil {
		t.Fatal(err)
	} else if n != 4 {
		t.Fatal(n)
	}

	if n, err := file.Read(buf[0:2]); err != nil {
		t.Fatal(err)
	} else if n != 1 {
		t.Fatal(n)
	} else if str := string(buf[0:n]); str != data[4:5] {
		t.Fatal(str)
	}

	if _, err := file.Seek(-100, os.SEEK_CUR); err == nil {
		t.Fatal(err)
	}
}

func TestMemoryFile_Seek_END(t *testing.T) {
	data := "12345"
	file := &MemoryFile{Data: data}
	buf := make([]byte, len(data))

	if n, err := file.Seek(-2, os.SEEK_END); err != nil {
		t.Fatal(err)
	} else if n != 3 {
		t.Fatal(n)
	}

	if n, err := file.Read(buf[0:2]); err != nil {
		t.Fatal(err)
	} else if n != 2 {
		t.Fatal(n)
	} else if str := string(buf[0:2]); str != data[3:5] {
		t.Fatal(str)
	}

	if _, err := file.Seek(-100, os.SEEK_END); err == nil {
		t.Fatal(err)
	}
}

func TestMemoryFile_Close(t *testing.T) {
	file := &MemoryFile{Data: "foobar"}
	buf := make([]byte, 100)

	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	if n, err := file.Read(buf); err == nil {
		t.Fatal(err)
	} else if n != -1 {
		t.Fatal(err)
	}

	if n, err := file.Seek(2, os.SEEK_SET); err == nil {
		t.Fatal(err)
	} else if n != -1 {
		t.Fatal(err)
	}

	if err := file.Close(); err == nil {
		t.Fatal(err)
	}
}

func TestMemoryFile_Stat_File(t *testing.T) {
	data := "foobar"
	modTime := time.Unix(time.Now().Unix(), 0)
	file := &MemoryFile{Data: data, ModTime: modTime.Unix()}

	stat, err := file.Stat()
	if err != nil {
		t.Fatal(err)
	}

	if size := stat.Size(); size != int64(len(data)) {
		t.Error(size)
	}

	if isDir := stat.IsDir(); isDir != false {
		t.Error(isDir)
	}

	if mTime := stat.ModTime(); !mTime.Equal(modTime) {
		t.Errorf("expected: %s, got; %s", modTime, mTime)
	}

	if sys := stat.Sys(); sys != nil {
		t.Error(sys)
	}

	mode := stat.Mode()
	if mode&os.ModeDir > 0 {
		t.Error("dir mode should not be set")
	}

	if mode&0444 != 0444 {
		t.Errorf("unexpected permission: 0%o", mode)
	}
}

func TestMemoryFile_Stat_Dir(t *testing.T) {
	file := &MemoryFile{IsDir: true}

	stat, err := file.Stat()
	if err != nil {
		t.Fatal(err)
	}

	if isDir := stat.IsDir(); isDir != true {
		t.Error(isDir)
	}

	mode := stat.Mode()
	if mode&os.ModeDir != os.ModeDir {
		t.Error("dir mode should be set")
	}

	if mode&0555 != 0555 {
		t.Errorf("unexpected permission: 0%o", mode)
	}

}

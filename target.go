package makefs

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	gopath "path"
	"time"
)

func newTarget(path string, startTaskOnce func()) *Target {
	return &Target{
		path:          path,
		startTaskOnce: startTaskOnce,
		broadcast:     newBroadcast(),
	}
}

type Target struct {
	path          string
	broadcast     *broadcast
	task          *Task
	startTaskOnce func()
}

func (t *Target) Write(buf []byte) (int, error) {
	return t.broadcast.Write(buf)
}

func (t *Target) closeWithError(err error) error {
	return t.broadcast.CloseWithError(err)
}

func (t *Target) httpFile() http.File {
	return newTargetFile(t)
}

func newTargetFile(target *Target) *targetFile {
	return &targetFile{target: target}
}

type targetFile struct {
	target *Target
	reader io.Reader
}

func (file *targetFile) Close() error {
	// @TODO make future read calls fail
	return nil
}

func (file *targetFile) Read(buf []byte) (int, error) {
	if file.reader == nil {
		file.reader = file.client()
	}
	return file.reader.Read(buf)
}

func (file *targetFile) Seek(offset int64, whence int) (int64, error) {
	return 0, fmt.Errorf("not done yet: Seek()")
}

func (file *targetFile) Readdir(count int) ([]os.FileInfo, error) {
	// @TODO is there something more idomatic we can return here that makes sense
	// cross-plattform?
	return nil, fmt.Errorf("readdir: target file is not a dir")
}

func (file *targetFile) Stat() (os.FileInfo, error) {
	stat := &targetStat{targetFile: file}
	return stat, nil
}

func (file *targetFile) client() io.Reader {
	file.target.startTaskOnce()
	return file.target.broadcast.Client()
}

type targetStat struct {
	targetFile *targetFile
}

func (s *targetStat) IsDir() bool {
	// @TODO support targets that are directories
	return false
}

func (s *targetStat) ModTime() time.Time {
	// @TODO finish
	return time.Now()
}

func (s *targetStat) Mode() os.FileMode {
	// @TODO Finish
	return 0
}

func (s *targetStat) Name() string {
	return gopath.Base(s.targetFile.target.path)
}

// Size determines the size of the target file by creating a new broadcast
// client, and counting the bytes until EOF. It returns -1 if the broadcast
// client returns an error other than EOF from read.
//
// This means that calling this methods requires executing the recipe.
func (s *targetStat) Size() int64 {
	client := s.targetFile.client()
	n, err := io.Copy(ioutil.Discard, client)
	if err != nil {
		return -1
	}
	return n
}

func (s *targetStat) Sys() interface{} {
	return nil
}

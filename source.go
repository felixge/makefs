package makefs

import (
	"fmt"
	"net/http"
)

type Source struct {
	path string
	fs   http.FileSystem
	file http.File
}

func (s *Source) open() error {
	if s.file != nil {
		return fmt.Errorf("makefs: source already open: %s", s.path)
	}

	file, err := s.fs.Open(s.path)
	if err != nil {
		return err
	}

	s.file = file
	return nil
}

func (s *Source) close() error {
	if s.file == nil {
		return fmt.Errorf("makefs: source not open: %s", s.path)
	}

	return s.file.Close()
}

func (s *Source) Read(buf []byte) (int, error) {
	return s.file.Read(buf)
}

package makefs

import (
	"io"
)

func newTask(targetPath string, sources []*Source) *Task {
	target := &Target{path: targetPath}
	return &Task{target: target, sources: sources}
}

type Task struct {
	target  *Target
	sources []*Source
}

func (t *Task) id() string {
	return ""
}

func (t *Task) current(other *Task) bool {
	return false
}

func (t *Task) Target() io.Writer {
	return nil
}

func (t *Task) Source() io.Reader {
	return nil
}

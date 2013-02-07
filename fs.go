package makefs

import (
	"fmt"
	"io"
	"net/http"
	"os/exec"
)

func NewFs(base http.FileSystem) *Fs {
	return &Fs{head: base}
}

type Fs struct {
	head http.FileSystem
}

func (fs *Fs) Open(path string) (http.File, error) {
	return fs.head.Open(path)
}

func (fs *Fs) Make(target string, sources []string, recipe Recipe) error {
	rule := &rule{
		target:  target,
		sources: sources,
		recipe:  recipe,
	}

	ruleFs, err := newRuleFs(fs.head, rule)
	if err != nil {
		return err
	}

	fs.head = ruleFs
	return nil
}

func (fs *Fs) ExecMake(target string, source string, command string, args ...string) {
	fs.Make(target, []string{source}, func(task *Task) error {
		cmd := exec.Command(command, args...)
		cmd.Stdin = task.Source()
		cmd.Stdout = task.Target()
		cmd.Stderr = task.Target()

		err := cmd.Run()
		if err == nil {
			return nil
		}
		return fmt.Errorf("%s: %s", command, err.Error())
	})
}

func (fs *Fs) SubFs(newRoot string) http.FileSystem {
	return NewSubFs(fs.head, newRoot)
}

func (fs *Fs) Include(includes... string) {
	fs.head = NewIncludeFs(fs.head, includes...)
}

func (fs *Fs) Fprint(w io.Writer, pkg string, varName string) error {
	return Fprint(w, fs, pkg, varName)
}

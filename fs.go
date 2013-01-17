package makefs

/*
Vocabulary:

* pattern: a string containing exactly one % sign.
* glob: a string containing one or more * character. Single stars expand to any
	character except for the path separator. Double stars expand to any character,
	including the separator.


*/

import (
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

func (fs *Fs) MultiMake(targets []string, sources []string, recipe Recipe) {
	rule := &rule{
		targets: targets,
		sources: sources,
		recipe:  recipe,
	}

	fs.head = &ruleFs{
		parent: fs.head,
		rule:   rule,
	}
}

func (fs *Fs) Make(target string, source string, recipe Recipe) {
	fs.MultiMake([]string{target}, []string{source}, recipe)
}

func (fs *Fs) ExecMake(target string, source string, command string, args ...string) {
	fs.Make(target, source, func(task *Task) error {
		cmd := exec.Command(command, args...)
		cmd.Stdin = task.Source()
		cmd.Stdout = task.Target()
		cmd.Stderr = task.Target()
		return cmd.Run()
	})
}

func (fs *Fs) SubFs(newRoot string) http.FileSystem {
	return NewSubFs(fs.head, newRoot)
}

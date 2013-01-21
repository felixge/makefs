package makefs

import (
	"fmt"
	"sync"
)

func newTask(targetPath string, sources []*Source, recipe Recipe) *Task {
	task := &Task{sources: sources, recipe: recipe}
	task.target = newTarget(targetPath, func() { task.startOnce() })
	return task
}

type Task struct {
	target  *Target
	sources []*Source
	recipe  Recipe
	once    sync.Once
}

func (t *Task) current(other *Task) bool {
	return false
}

func (t *Task) Target() *Target {
	return t.target
}

func (t *Task) Source() *Source {
	return t.sources[0]
}

func (t *Task) Sources() []*Source {
	return t.sources
}

func (t *Task) startOnce() {
	t.once.Do(func() { go t.run() })
}

func (t *Task) run() {
	// Open all source files
	for _, source := range t.sources {
		if err := source.open(); err != nil {
			// @TODO: handle this
			panic("could not open source")
		}
		defer source.close()
	}

	// Execute recipe
	err := t.recipe(t)
	t.target.closeWithError(err)

	// TODO: what should we really do with this?
	if err != nil {
		fmt.Printf("RECIPE ERROR: %s\n", err)
	}
}

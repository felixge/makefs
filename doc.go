/*
Package makefs provides tools for processing and serializing virtual file
systems. A typical use case is processing assets (js, css, etc.) for a web
application, and serializing the resulting file system into a .go file for
production.

Makefs is inspired by the venerable GNU Make
(http://www.gnu.org/software/make/manual/make.html), and aims for a compatible
pattern syntax for defining targets and sources.

For example, this code will create a http.FileSystem that converts .less files
to .css files:

	fs := makefs.NewFs(http.Dir("/path/to/webroot"))
	fs.Make("%.css", []string{"%.less"}, func(t *Task) error {
		cmd := exec.Command("/path/to/less-processor.js", "-")
		cmd.Stdin = task.Source()
		cmd.Stdout = task.Target()
		cmd.Stderr = task.Target()
		return cmd.Run()
	})

The first argument ("%.css") defines the target to be created. In this case the
target is a pattern, matching any path ending in .css.

The second argument ([]string{"%.less"}) defines the sources (called
prerequisites by GNU Make) required to build the given target. In this case
each .css target requires one .less target with the same name / path prefix to
be present.

The third argument (func(t *Task) error) defined the recipe. The recipe
describes how the target is created from the given sources. A recipe is only
re-executed if one or more matching source files have changed since the last
execution.

Together, these arguments form what is called a Rule.

*/
package makefs

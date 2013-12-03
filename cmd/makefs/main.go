package main

import (
	"flag"
	"fmt"
	"github.com/felixge/makefs"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
)

const (
	name = "makefs"
)

func main() {
	var (
		flags    = flag.NewFlagSet(name, flag.ExitOnError)
		help     = flags.Bool("h", false, "Display help")
		pkgName  = flags.String("p", "", "Package name. Defaults to directory basename.")
		varName  = flags.String("n", "Fs", "Name of the variable to export.")
		fileName = flags.String("f", "fs.go", "Name of the file to create.")
	)

	flags.Usage = func() {
		fmt.Printf("%s [-p <pkgName>] [-n <varName>] <dir>\n", name)
		flags.PrintDefaults()
	}

	flags.Parse(os.Args[1:])
	if *help {
		flags.Usage()
		os.Exit(0)
	}

	args := flags.Args()
	if len(args) != 1 {
		flags.Usage()
		os.Exit(1)
	}

	dir := args[0]
	if *pkgName == "" {
		*pkgName = filepath.Base(dir)
	}

	if err := writeFs(dir, *pkgName, *varName, *fileName); err != nil {
		fmt.Printf("%s\n", err)
		os.Exit(1)
	}
}

func writeFs(dir, pkgName, varName, fileName string) (err error) {
	var (
		filePath = filepath.Join(dir, fileName)
		file     *os.File
	)

	os.Remove(filePath)

	// Create temporary file, otherwise Fprint() will pick up our output file
	// itself.
	file, err = ioutil.TempFile(os.TempDir(), "makefs")
	if err != nil {
		return
	}
	defer file.Close()

	if err = makefs.Fprint(file, http.Dir(dir), pkgName, varName); err != nil {
		return
	}
	if err = os.Rename(file.Name(), filePath); err != nil {
		return
	}
	return
}

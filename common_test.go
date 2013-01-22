package makefs

import (
	"path/filepath"
	"runtime"
)

// This file holds things that are shared across multiple test files inside the
// test suite.

// Get name/dir of this source file
var (
	_, __filename, _, _ = runtime.Caller(0)
	__dirname           = filepath.Dir(__filename)
	fixturesDir         = __dirname + "/fixtures"
)

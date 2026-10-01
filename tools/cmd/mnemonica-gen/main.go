// Command mnemonica-gen is the Go analogue of the JS tactica: it scans a
// package's mnemonica.Define/mnemonica.Sub calls and generates, next to the
// hand-written code, a typed wire func per subtype (attached in a generated
// init, replacing the reflect setter) and a typed construction method per
// subtype on the PARENT struct, so `user.Admin(args)` reads like
// mnemonica. The Define/Sub call sites stay hand-written; mnemonica-gen
// only attaches. Run it with //go:generate mnemonica-gen in the package's
// directory.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// exitFunc is swappable so tests can cover main without exiting the test
// binary.
var exitFunc = os.Exit

// writeFile is os.WriteFile under a name, so tests can force the write
// error path without filesystem gymnastics.
var writeFile = os.WriteFile

func main() {
	result := run(os.Args[1:])
	exitFunc(result)
}

func run(args []string) int {
	// Skip flag-shaped args: in a test binary os.Args carries -test.*
	// flags; production invocations pass plain directories.
	dir := "."
	for _, arg := range args {
		if !strings.HasPrefix(arg, "-") {
			dir = arg
			break
		}
	}
	generated, err := Generate(dir)
	if err != nil {
		fmt.Fprintln(os.Stderr, "mnemonica-gen:", err)
		return 1
	}
	writeErr := writeFile(filepath.Join(dir, outputName), generated, 0o644)
	if writeErr != nil {
		fmt.Fprintln(os.Stderr, "mnemonica-gen:", writeErr)
		return 1
	}
	return 0
}

// Package runtimepin pins the runtime module requirement. The tools'
// testdata packages import mnemonica/mnemonica, which `go mod tidy` cannot
// see, and without this blank import tidy would drop the require and every
// testdata package would stop compiling.
package runtimepin

import _ "mnemonica/mnemonica"

// Package broken does not compile: Generate must surface the package error
// rather than panic or emit garbage.
package broken

var Oops = undefinedReference

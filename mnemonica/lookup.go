package mnemonica

import "strings"

// Lookup resolves a type by its dotted path ("User", "User.Admin") in the
// collection (C1.4). The result is untyped by necessity — only the
// UserT/AdminT values know their struct type; use it for identity and
// dynamic construction. An unknown path yields ok == false; Lookup itself
// never fails.
func Lookup(collection *Collection, path string) (Type, bool) {
	result, ok := collection.Lookup(path)
	return result, ok
}

// Lookup resolves a dotted path in the collection; see the package-level
// Lookup. Unknown path → ok == false, no panic.
func (c *Collection) Lookup(path string) (Type, bool) {
	segments := strings.Split(path, ".")
	c.mu.RLock()
	defer c.mu.RUnlock()
	record, ok := c.roots[segments[0]]
	if !ok {
		var zero Type
		return zero, false
	}
	for _, segment := range segments[1:] {
		record, ok = record.children[segment]
		if !ok {
			var zero Type
			return zero, false
		}
	}
	result := Type{record: record}
	return result, true
}

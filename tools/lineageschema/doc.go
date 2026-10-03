// Package lineageschema validates the runtime's lineage export against
// the shared JSON Schema. The schema and the canonical fixture are
// embedded by github.com/mythographica/lethe — the single source of
// truth for the cross-language contract — so this package never reads
// files that could drift. The validator dependency lives HERE, in the
// tools module, never in the stdlib-only runtime. The tests import the
// runtime, produce graphs of every shape the format allows, and assert
// they validate — plus one byte-for-byte reproduction of the fixture.
package lineageschema

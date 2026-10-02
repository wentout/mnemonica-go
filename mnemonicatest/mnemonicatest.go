// Package mnemonicatest is the C2.4 adapted guard for tests: Go cannot
// shadow promoted writes (admin.Name = x writes the shared parent), so a
// test that must prove "construction changed nothing upstream" needs a
// baseline. Define types with mnemonica.WithParentSnapshots(true) and
// assert with AssertParentsUnchanged: the runtime snapshots every
// ancestor's user-visible fields at construction time, and the assertion
// compares the live lineage against that baseline. The P5 mnemonicavet
// analyzer is the static counterpart of this guard.
package mnemonicatest

import (
	"reflect"
	"testing"

	"github.com/wentout/mnemonica-go/mnemonica"
)

// AssertParentsUnchanged reports every ancestor whose user-visible fields
// differ from the construction-time snapshot: it is the test-time guard
// for the adapted C2.4 write-local rule. x may be any value; anything
// without a construction record, or defined without
// mnemonica.WithParentSnapshots(true), is reported via t.Errorf.
func AssertParentsUnchanged(t testing.TB, x any) {
	t.Helper()
	record, err := mnemonica.Props(x)
	if err != nil {
		t.Errorf("AssertParentsUnchanged: %v", err)
		return
	}
	if record.Snapshot == nil {
		t.Errorf("AssertParentsUnchanged: type %q was defined without mnemonica.WithParentSnapshots(true)", record.Type.Path())
		return
	}
	current := currentAncestorFields(record.Parent)
	for _, want := range record.Snapshot {
		key := want.Type + "." + want.Field
		got := current[key]
		if !reflect.DeepEqual(got, want.Value) {
			t.Errorf("AssertParentsUnchanged: ancestor field %s changed after construction: was %v, now %v", key, want.Value, got)
		}
	}
}

// currentAncestorFields re-walks the lineage and maps "Type.Field" to the
// live value, with the same visibility rules as the runtime's snapshot:
// exported, non-embedded fields only.
func currentAncestorFields(parent mnemonica.Instance) map[string]any {
	fields := make(map[string]any)
	for cursor := parent; cursor != nil; {
		// Lineage nodes are always constructed instances, so Props cannot
		// fail here; the error is ignored on that invariant.
		record, _ := mnemonica.Props(cursor)
		value := reflect.ValueOf(cursor).Elem()
		typeOf := value.Type()
		for index := 0; index < value.NumField(); index++ {
			field := typeOf.Field(index)
			if field.PkgPath != "" || field.Anonymous {
				continue
			}
			fields[typeOf.Name()+"."+field.Name] = value.Field(index).Interface()
		}
		cursor = record.Parent
	}
	return fields
}

package mnemonica

import (
	"fmt"
	"time"
)

// Record is the construction context of an instance (C3) — the Go
// counterpart of the JS getProps() result. Adaptations from the JS
// internal property table: Timestamp is a time.Time rather than epoch
// milliseconds; Subtypes is a sorted snapshot rather than the live map
// (thread-safety; the JS map is live); Creator is the constructing type
// itself (JS InstanceCreatorContext can differ from __type__ in
// chained/async construction, which has no Go counterpart); the
// prototype-specific __proto_proto__ does not exist in Go; setProps is P3.
type Record struct {
	// Type is the type this instance was constructed as (JS __type__).
	Type Type
	// Parent is the specific parent instance (JS __parent__); nil for roots.
	Parent Instance
	// Args is the construction argument value (JS __args__ — an array
	// there; a single typed value here), as passed to New/From. It is one
	// of the two any-typed surfaces of the API, by design.
	Args any
	// Timestamp is the construction time (JS __timestamp__, epoch
	// milliseconds there; a time.Time here).
	Timestamp time.Time
	// Creator is the type that constructed the instance (JS __creator__).
	Creator Type
	// Collection is the types collection (JS __collection__).
	Collection *Collection
	// Subtypes are the types declared under the instance's type
	// (JS __subtypes__), a sorted snapshot.
	Subtypes []Type
	// Self is the instance this record belongs to (JS __self__).
	Self Instance
	// Stack is the construction stack (JS __stack__), recorded only when
	// the type's submitStack config is on (C4.3); empty otherwise.
	Stack string
	// Snapshot is the construction-time copy of every ancestor's
	// user-visible fields — the baseline for the C2.4 adapted guard
	// (mnemonicatest.AssertParentsUnchanged). nil unless the type was
	// defined with WithParentSnapshots(true).
	Snapshot []SnapshotField
}

// Props returns the construction record of a mnemonica instance (C3).
// It returns ErrNotAnInstance for any value that does not carry a genuine
// construction record — the seal on Instance is what makes that answer
// exact.
func Props(x any) (Record, error) {
	var zero Record
	instance, ok := x.(Instance)
	if !ok {
		err := fmt.Errorf("%w: %T", ErrNotAnInstance, x)
		return zero, err
	}
	node := instance.mnemonicaNode()
	record := node.record
	if record == nil {
		// A Node that never went through construction carries no record.
		err := fmt.Errorf("%w: %T was never constructed", ErrNotAnInstance, x)
		return zero, err
	}
	result := Record{
		Type:       Type{record: record},
		Parent:     node.parent,
		Args:       node.args,
		Timestamp:  node.timestamp,
		Creator:    Type{record: record},
		Collection: record.collection,
		Subtypes:   record.subtypeHandles(),
		Self:       instance,
		Stack:      node.stack,
		Snapshot:   node.snapshot,
	}
	return result, nil
}

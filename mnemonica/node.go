package mnemonica

import (
	"context"
	"time"
)

// Instance is the seal every mnemonica instance satisfies. The method is
// unexported on purpose: unexported method identity includes the defining
// package, so no type outside package mnemonica can implement Instance —
// embedding Node is the ONLY way to get the method into a method set (the
// same sealing trick the standard library uses inside http). This is what
// guarantees Props, Is and As only ever see real mnemonica instances with
// a genuine lineage header, never a hand-rolled fake.
type Instance interface {
	mnemonicaNode() *Node
}

// SnapshotField is one captured ancestor field — the construction-time
// baseline for the C2.4 adapted guard. Only user-visible state is
// captured: exported, non-embedded fields.
type SnapshotField struct {
	Type  string // ancestor type name
	Field string // field name
	Value any    // shallow copy of the value at construction time
}

// Node is the lineage header every mnemonica type embeds (by value). It
// carries the construction record (C3): which type, constructed from which
// parent instance, with which args, when, in which collection. The fields
// are unexported and read through Props; users interact with Node only by
// embedding it.
type Node struct {
	// idCounter is the instance's lineage id counter: 0 means "not
	// assigned yet" — the global counter starts at 1, so 0 is never a
	// real id. Kept FIRST so the word is 64-bit aligned for the atomic
	// CAS in ID on every platform (on 32-bit arches Go guarantees
	// alignment only for a struct's first word; embed Node as the first
	// field — the documented idiom in every example). The counter rides
	// in the instance itself, so an id'd instance stays collectible: no
	// registry pins it. Copies of a Node carry its counter and so share
	// its id — a copy IS the same construction record.
	idCounter  uint64
	record     *typeRecord
	parent     Instance
	args       any
	timestamp  time.Time
	collection *Collection
	// err marks an errored instance (C4.2): set when the constructor
	// failed. It is what blockErrors checks in the lineage, and it is
	// never visible through Props — errored state surfaces as the
	// *ErroredInstance error and as ErrBlocked.
	err error
	// stack is the construction stack when submitStack is on (C4.3),
	// empty otherwise.
	stack string
	// ctx is the detached construction context (P4): the values of the
	// context passed to NewCtx/FromCtx, without cancellation or deadline —
	// the data outlives the request. nil means no context was attached;
	// it is the only state, so hook-less, ctx-less construction allocates
	// nothing extra.
	ctx context.Context
	// snapshot is the ancestor-field baseline when WithParentSnapshots is
	// on, nil otherwise (the C2.4 adapted guard).
	snapshot []SnapshotField
}

// mnemonicaNode implements Instance. Defined on *Node so that only pointer
// instances satisfy Instance: a copied User value must not silently carry
// an independent lineage header.
func (n *Node) mnemonicaNode() *Node {
	result := n
	return result
}

// As walks the lineage of x — x itself, then its parent, then the
// grandparent, exactly like errors.As walks a wrapped error chain — and
// returns the first instance assertable to T. An Admin constructed from a
// User IS a *User along its chain (C2.5), so As[*User](admin) returns the
// parent user and true; if no ancestor matches, As returns the zero T and
// false. T must be a pointer type (instances are always pointers).
func As[T any](x Instance) (T, bool) {
	var zero T
	for current := x; current != nil; current = current.mnemonicaNode().parent {
		if typed, ok := current.(T); ok {
			return typed, true
		}
	}
	return zero, false
}

// Is reports whether T appears anywhere in x's lineage — x itself or any
// ancestor — the nominal-identity counterpart of errors.Is (C2.5).
func Is[T any](x Instance) bool {
	_, ok := As[T](x)
	return ok
}

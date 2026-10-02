package mnemonica

import "reflect"

// config is the resolved per-type configuration (C4). It is computed once
// at Define/Sub time and immutable afterwards: every construction of a
// type sees the same config, which is what makes construction itself
// lock-free. Values mirror the JS defaults and semantics (FOR_HUMANS,
// Configuration Options).
type config struct {
	// strictChain (default true): only the type's own direct subtypes may
	// be constructed from its instances; false also allows subtypes found
	// up the chain (C4.1).
	strictChain bool
	// blockErrors (default true): a failing constructor produces an
	// *ErroredInstance carrying the half-built instance, and construction
	// from an instance whose lineage errored is refused with ErrBlocked
	// (C4.2).
	blockErrors bool
	// submitStack (default false): Record gets the construction stack
	// (C4.3).
	submitStack bool
	// snapshot is Go-specific (no JS counterpart): capture the user-visible
	// fields of every ancestor at construction time, so
	// mnemonicatest.AssertParentsUnchanged can detect the promoted-write
	// mutation that the adapted C2.4 warns about.
	snapshot bool
}

// builtinDefaults mirrors the JS defaults: strictChain and blockErrors on,
// submitStack off.
func builtinDefaults() config {
	result := config{
		strictChain: true,
		blockErrors: true,
	}
	return result
}

// optionState accumulates explicit per-key overrides. Pointers distinguish
// "not set" from "set to the zero value": a subtype must be able to turn a
// parent's true back to false while inheriting every other key.
type optionState struct {
	strictChain *bool
	blockErrors *bool
	submitStack *bool
	snapshot    *bool

	// collectionName names a collection for the lineage export. Only
	// NewCollection consumes it; Define/Sub options silently ignore it.
	collectionName *string

	// wire is an explicit parent-wiring func (WithWireFunc, P5): it
	// replaces the default cached reflect setter. wireChild/wireParent
	// remember the func's declared pointer types so Sub can validate them
	// against its own instead of failing per construction.
	wire       func(child, parent any)
	wireChild  reflect.Type
	wireParent reflect.Type
}

// Option configures a collection (NewCollection) or a type (Define/Sub).
// Options are the Go counterpart of the JS config object argument.
type Option func(*optionState)

// WithStrictChain sets the strictChain config key (default true). See C4.1.
func WithStrictChain(value bool) Option {
	return func(state *optionState) {
		state.strictChain = &value
	}
}

// WithBlockErrors sets the blockErrors config key (default true). See C4.2.
func WithBlockErrors(value bool) Option {
	return func(state *optionState) {
		state.blockErrors = &value
	}
}

// WithSubmitStack sets the submitStack config key (default false). See C4.3.
func WithSubmitStack(value bool) Option {
	return func(state *optionState) {
		state.submitStack = &value
	}
}

// WithParentSnapshots enables construction-time ancestor snapshots for the
// C2.4 adapted guard (default false). Go-specific: JS needs no snapshot
// because write-local is native there; in Go, mnemonicatest needs a
// construction-time baseline to compare against.
func WithParentSnapshots(value bool) Option {
	return func(state *optionState) {
		state.snapshot = &value
	}
}

// WithCollectionName names a collection for the lineage export (L1). It
// is meaningful only as a NewCollection option; Define and Sub ignore it.
func WithCollectionName(name string) Option {
	return func(state *optionState) {
		state.collectionName = &name
	}
}

// WithWireFunc wires the subtype with an explicit typed func instead of
// the default cached reflect setter (the P0 decision's alternative, opened
// for hand-written and generated wiring). Sub validates the func's types
// against its own and rejects a mismatch at define time. With a wire func,
// Sub skips the embedded-parent-field validation: the func asserts the
// shape itself. Documented any-exception: Option is not generic, so the
// typed func travels erased, with its reflect types alongside for the
// Sub-time validation.
func WithWireFunc[C, P any](wire func(*C, *P)) Option {
	return func(state *optionState) {
		state.wire = func(child, parent any) {
			wire(child.(*C), parent.(*P))
		}
		state.wireChild = reflect.TypeOf((*C)(nil))
		state.wireParent = reflect.TypeOf((*P)(nil))
	}
}

// collectOptions runs the variadic options into a fresh state.
func collectOptions(options []Option) optionState {
	var state optionState
	for _, option := range options {
		option(&state)
	}
	return state
}

// resolve applies the explicit overrides onto a base config and returns
// the result. Roots base on the collection defaults; subtypes base on the
// parent's RESOLVED config — the JS subtype config inheritance (a subtype
// inherits the parent definition's configuration, overridden per key).
func (state *optionState) resolve(base config) config {
	result := base
	if state.strictChain != nil {
		result.strictChain = *state.strictChain
	}
	if state.blockErrors != nil {
		result.blockErrors = *state.blockErrors
	}
	if state.submitStack != nil {
		result.submitStack = *state.submitStack
	}
	if state.snapshot != nil {
		result.snapshot = *state.snapshot
	}
	return result
}

// Package mnemonica is the Go port of mnemonica core: instance-level
// inheritance. A subtype is not constructed from a class but FROM an
// existing parent instance — admin, err := AdminT.From(user, args) — and
// the new instance keeps a live link to that parent. The chain from any
// instance back to the root instance is the data's own history: what it
// was made from, with which arguments, in which order. That chain is the
// identity, checked nominally by type, not by shape.
//
// Go maps the model onto embedding: a subtype struct embeds a POINTER to
// its parent type (type Admin struct { mnemonica.Node; *User; Role string }),
// so reads promote natively — admin.Name reads through to the shared
// *User (C2.3 read-through).
//
// Write-local (C2.4) is NOT native to Go: admin.Name = x writes into the
// shared parent, because Go cannot shadow promoted field writes. This port
// records C2.4 as adapted: define types with WithParentSnapshots(true) and
// assert with mnemonicatest.AssertParentsUnchanged in tests; the mnemonicavet
// analyzer (P5) is the static guard. See docs/conformance.md.
//
// Construction never panics: it returns (T, error), and failures are
// sentinel values checkable with errors.Is — what %w did for errors, for
// your data. Configuration (C4) is per type with collection defaults and
// subtype inheritance; lifecycle hooks (C5) register on a type or a
// collection.
//
// Context binds to data (P4): Go binds context to CALLS, mnemonica keeps
// what must outlive the call. NewCtx/FromCtx record a construction ctx's
// VALUES — detached from cancellation and deadline, because the data
// outlives the request — and ContextOf(x) derives a lineage context for
// work that outlives it. See the ContextOf documentation.
package mnemonica

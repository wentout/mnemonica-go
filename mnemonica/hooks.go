package mnemonica

import (
	"context"
	"slices"
)

// HookKind identifies a lifecycle hook (C5). The values are the JS hook
// type literals.
type HookKind string

const (
	// HookPreCreation fires before construction; returning an error aborts
	// it, and no creationError hooks fire for the abort.
	HookPreCreation HookKind = "preCreation"
	// HookPostCreation fires after a successful construction.
	HookPostCreation HookKind = "postCreation"
	// HookCreationError fires when a constructor failed with blockErrors
	// enabled — an errored instance was produced (C4.2).
	HookCreationError HookKind = "creationError"
)

// HookData is the payload every hook receives (C5), the idiomatic Go
// counterpart of the JS HookData:
//
//	TypeName/type        → Type
//	args                 → Args (the single typed construction args value)
//	existentInstance     → Parent
//	inheritedInstance    → Instance (nil in preCreation: JS fires preCreation
//	                       before the memory layer exists, so there is no
//	                       instance yet)
//
// Err is the constructor's original error, set only for creationError.
//
// Ctx is the P4 construction context: the detached ctx passed to
// NewCtx/FromCtx (or inherited by Clone/Fork and friends), or
// context.Background() when none was attached — always non-nil, so hooks
// can rely on it without a nil check.
type HookData struct {
	Type     Type
	Args     any
	Parent   Instance
	Instance Instance
	Err      error
	Ctx      context.Context
}

// HookFunc is a lifecycle hook. Returning an error aborts construction and
// propagates unwrapped: for preCreation before anything is built (JS: "throw
// normally to abort construction; the error propagates unwrapped"), for
// postCreation/creationError the hook error is what New/From return.
type HookFunc func(*HookData) error

// validHookKind reports whether kind is one of the three lifecycle hooks.
func validHookKind(kind HookKind) bool {
	switch kind {
	case HookPreCreation, HookPostCreation, HookCreationError:
		return true
	}
	return false
}

// hooksFor returns the hooks to run for kind in contract order: preCreation
// runs collection hooks first, then type hooks (the collection wraps the
// type on the pre side); postCreation and creationError run type hooks
// first, then collection hooks (invokePostHooks in the JS calls the type
// first). Slices are copied under the collection lock so hooks run outside
// any lock and may themselves define types or register hooks. Callers gate
// on hasHooks first, so the result is never empty.
func (r *typeRecord) hooksFor(kind HookKind) []HookFunc {
	collection := r.collection
	collection.mu.RLock()
	typeHooks := slices.Clone(r.hooks[kind])
	collectionHooks := slices.Clone(collection.hooks[kind])
	collection.mu.RUnlock()
	var result []HookFunc
	if kind == HookPreCreation {
		result = append(result, collectionHooks...)
		result = append(result, typeHooks...)
	} else {
		result = append(result, typeHooks...)
		result = append(result, collectionHooks...)
	}
	return result
}

// hasHooks reports quickly whether any hook is registered for kind, on the
// type or the collection. Construction builds the HookData payload only
// when hooks exist, so hook-less construction allocates nothing extra.
func (r *typeRecord) hasHooks(kind HookKind) bool {
	collection := r.collection
	collection.mu.RLock()
	found := len(r.hooks[kind])+len(collection.hooks[kind]) > 0
	collection.mu.RUnlock()
	return found
}

// runHooks invokes the hooks for kind in order; the first error aborts.
// Callers build the HookData only after hasHooks(kind) is true.
func (r *typeRecord) runHooks(kind HookKind, data *HookData) error {
	for _, hook := range r.hooksFor(kind) {
		if err := hook(data); err != nil {
			return err
		}
	}
	return nil
}

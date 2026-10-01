package mnemonica

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

// This file is the C7 utils port: standalone functions over instances, the
// Go counterparts of the JS `utils` export (the JS core's docs/UTILS.md).
// Every util that reconstructs (Clone, Fork, ForkOnto, Sibling, Merge) runs
// the SAME construction pipeline as From — hooks, config, blockErrors and
// errored instances included — because that is what the JS fork/merge do:
// they re-enter InstanceCreator, they do not shallow-copy.

// nodeOf resolves the lineage header of x, mapping a typed-nil pointer to
// nil — the JS utils guard their null the same way before walking.
func nodeOf(x Instance) *Node {
	if x == nil {
		return nil
	}
	value := reflect.ValueOf(x)
	if value.Kind() == reflect.Pointer && value.IsNil() {
		return nil
	}
	result := x.mnemonicaNode()
	return result
}

// resolveInstance validates x for the utils that read or reconstruct: it
// must be an Instance (something embedding Node), non-nil, and constructed.
// It returns the instance and its type record together, so callers never
// re-assert.
func resolveInstance(x any) (Instance, *typeRecord, error) {
	instance, ok := x.(Instance)
	if !ok {
		return nil, nil, fmt.Errorf("%w: %T", ErrNotAnInstance, x)
	}
	node := nodeOf(instance)
	if node == nil {
		return nil, nil, fmt.Errorf("%w: nil instance", ErrNotAnInstance)
	}
	record := node.record
	if record == nil {
		return nil, nil, fmt.Errorf("%w: %T was never constructed", ErrNotAnInstance, x)
	}
	return instance, record, nil
}

// recordOf resolves the type record alone, for callers that do not need
// the instance back.
func recordOf(x any) (*typeRecord, error) {
	_, record, err := resolveInstance(x)
	if err != nil {
		return nil, err
	}
	return record, nil
}

// Extract returns a flat map of the user-visible (exported, non-embedded)
// fields along x's lineage: x first, then its parent, then the
// grandparent, to the root. A nearer field shadows an ancestor's
// same-named field (nearest wins), matching the JS prototype shadowing.
// Values are untyped at runtime — the honest any, as in the JS
// Extracted<T>.
func Extract(x Instance) map[string]any {
	result := make(map[string]any)
	if nodeOf(x) == nil {
		return result
	}
	for cursor := x; cursor != nil; cursor = cursor.mnemonicaNode().parent {
		value := reflect.ValueOf(cursor).Elem()
		typeOf := value.Type()
		for index := 0; index < value.NumField(); index++ {
			field := typeOf.Field(index)
			if field.PkgPath != "" || field.Anonymous {
				continue
			}
			if _, exists := result[field.Name]; exists {
				continue // a nearer instance already won this name
			}
			result[field.Name] = value.Field(index).Interface()
		}
	}
	return result
}

// Pick returns the named fields along x's lineage, nearest occurrence
// winning — the subset semantics of the JS utils.pick. Names with no
// matching field are simply absent.
func Pick(x Instance, keys ...string) map[string]any {
	all := Extract(x)
	result := make(map[string]any, len(keys))
	for _, key := range keys {
		if value, exists := all[key]; exists {
			result[key] = value
		}
	}
	return result
}

// Parent returns the nearest parent of x. Without a path it is x's direct
// parent; with a single segment, the nearest ANCESTOR (x itself is never a
// candidate, per the JS rule) whose type name matches; with a dotted path
// ("GrandParent.Parent"), the segments must match contiguously upwards and
// the instance matched by the LAST segment is returned — the JS
// disambiguation for lineages where the same type name appears twice.
//
// The second result reports whether a parent was found: a root (or an
// unparented value) yields (nil, false), as does a path with no match. The
// nil is a nil Instance interface — there is deliberately no typed nil, so
// the result compares directly against nil. An empty path string means
// "no filter", the same as omitting the path.
func Parent(x Instance, path ...string) (Instance, bool) {
	node := nodeOf(x)
	if node == nil {
		return nil, false
	}
	if len(path) == 0 || path[0] == "" {
		if node.parent == nil {
			return nil, false
		}
		result := node.parent
		return result, true
	}
	segments := strings.Split(path[0], ".")
	last := len(segments) - 1
	for candidate := node.parent; candidate != nil; candidate = candidate.mnemonicaNode().parent {
		if !contiguousMatch(candidate, segments, last) {
			continue
		}
		result := candidate
		return result, true
	}
	return nil, false
}

// contiguousMatch reports whether candidate matches segments[last] and the
// segments before it match contiguously upwards from candidate's parent.
// Candidates come from a real lineage, so every node carries a record.
func contiguousMatch(candidate Instance, segments []string, last int) bool {
	cursor := candidate
	for index := last; index >= 0; index-- {
		if cursor == nil {
			return false
		}
		if cursor.mnemonicaNode().record.name != segments[index] {
			return false
		}
		cursor = cursor.mnemonicaNode().parent
	}
	return true
}

// Clone returns a new instance of x's type, constructed FROM THE SAME
// PARENT with THE SAME ARGS — the JS clone (fork(x).call(x)), which
// re-runs the constructor, not a shallow copy: hooks fire, and a fresh
// Node header means the clone shares the parent but owns its own record.
// Clone is "the same construction again", so it inherits x's recorded ctx
// (P4): same construction, same context.
func Clone[T any](x *T) (*T, error) {
	instance, record, err := resolveInstance(x)
	if err != nil {
		var zero *T
		return zero, err
	}
	node := instance.mnemonicaNode()
	constructed, err := record.construct(node.parent, node.args, node.ctx)
	if err != nil {
		var zero *T
		return zero, err
	}
	result := constructed.(*T)
	return result, nil
}

// Fork returns a new instance of x's type from x's parent with NEW args —
// the JS utils.fork called with arguments. The args value is any for the
// same reason the untyped Type.New is: the static args type is not
// recoverable from *T; the handler's own check rejects a mismatched value
// with ErrWrongArgumentsUsed. Fork inherits x's recorded ctx (P4): a fork
// keeps the source's request lineage. ForkCtx attaches an explicit ctx
// instead; a nil ctx there inherits, same as Fork.
func Fork[T any](x *T, args any) (*T, error) {
	result, err := ForkCtx(x, noCtx, args)
	if err != nil {
		var zero *T
		return zero, err
	}
	return result, nil
}

// ForkCtx is Fork with an explicit construction context (P4); a nil ctx
// inherits x's recorded ctx, an explicit one replaces it.
func ForkCtx[T any](x *T, ctx context.Context, args any) (*T, error) {
	instance, record, err := resolveInstance(x)
	if err != nil {
		var zero *T
		return zero, err
	}
	node := instance.mnemonicaNode()
	recorded := detachedContext(ctx)
	if recorded == nil {
		recorded = node.ctx
	}
	constructed, err := record.construct(node.parent, args, recorded)
	if err != nil {
		var zero *T
		return zero, err
	}
	result := constructed.(*T)
	return result, nil
}

// ForkOnto returns a new instance of x's type constructed FROM A DIFFERENT
// PARENT with the given args — the JS fork().call(thisArg) form, the DAG
// construction. The parent is validated exactly as by From: nominally,
// against the type's declared parent (with the C4.1 strictChain rules) —
// except that a ROOT-typed instance may be re-parented onto any instance,
// as in the JS merge of two roots. ForkOnto inherits x's recorded ctx (P4);
// ForkOntoCtx attaches an explicit ctx instead (nil inherits).
func ForkOnto[T any](x *T, parent Instance, args any) (*T, error) {
	result, err := ForkOntoCtx(x, parent, noCtx, args)
	if err != nil {
		var zero *T
		return zero, err
	}
	return result, nil
}

// ForkOntoCtx is ForkOnto with an explicit construction context (P4); a
// nil ctx inherits x's recorded ctx, an explicit one replaces it.
func ForkOntoCtx[T any](x *T, parent Instance, ctx context.Context, args any) (*T, error) {
	instance, record, err := resolveInstance(x)
	if err != nil {
		var zero *T
		return zero, err
	}
	recorded := detachedContext(ctx)
	if recorded == nil {
		recorded = instance.mnemonicaNode().ctx
	}
	constructed, err := record.constructOnto(parent, args, recorded)
	if err != nil {
		var zero *T
		return zero, err
	}
	result := constructed.(*T)
	return result, nil
}

// Sibling returns a new instance of x's type constructed FROM x's parent
// with the given args — another child of the same parent. (The JS
// SiblingAccessor type-lookup form has no extra value in Go, where Lookup
// already resolves types by name.) Sibling inherits x's recorded ctx (P4):
// the new child belongs to the same request; SiblingCtx attaches an
// explicit ctx instead (nil inherits).
func Sibling[T any](x *T, args any) (*T, error) {
	result, err := SiblingCtx(x, noCtx, args)
	if err != nil {
		var zero *T
		return zero, err
	}
	return result, nil
}

// SiblingCtx is Sibling with an explicit construction context (P4); a nil
// ctx inherits x's recorded ctx, an explicit one replaces it.
func SiblingCtx[T any](x *T, ctx context.Context, args any) (*T, error) {
	instance, record, err := resolveInstance(x)
	if err != nil {
		var zero *T
		return zero, err
	}
	node := instance.mnemonicaNode()
	recorded := detachedContext(ctx)
	if recorded == nil {
		recorded = node.ctx
	}
	constructed, err := record.construct(node.parent, args, recorded)
	if err != nil {
		var zero *T
		return zero, err
	}
	result := constructed.(*T)
	return result, nil
}

// Merge returns a new instance of a's type whose parent is b — the JS
// utils.merge, defined there as fork(a).call(b, ...args). a provides the
// fields; b's fields are visible through read-through for everything a
// does not define (b fills non-overlapping keys). b is validated as a
// parent exactly like From, so with the default strictChain b must be an
// instance of a's declared parent type. Merge inherits a's recorded ctx
// (P4) — the merged result stays in the source's request; MergeCtx
// attaches an explicit ctx instead (nil inherits).
func Merge[T any](a *T, b Instance, args any) (*T, error) {
	result, err := MergeCtx(a, b, noCtx, args)
	if err != nil {
		var zero *T
		return zero, err
	}
	return result, nil
}

// MergeCtx is Merge with an explicit construction context (P4); a nil ctx
// inherits a's recorded ctx, an explicit one replaces it.
func MergeCtx[T any](a *T, b Instance, ctx context.Context, args any) (*T, error) {
	instance, record, err := resolveInstance(a)
	if err != nil {
		var zero *T
		return zero, err
	}
	recorded := detachedContext(ctx)
	if recorded == nil {
		recorded = instance.mnemonicaNode().ctx
	}
	constructed, err := record.constructOnto(b, args, recorded)
	if err != nil {
		var zero *T
		return zero, err
	}
	result := constructed.(*T)
	return result, nil
}

// Exception is the Go counterpart of the JS utils.exception: an error
// value carrying the instance it was made from. Adaptation: the JS
// exception is an error INSTANCE of the instance's own type (its data
// lives in getProps); Go cannot mint an arbitrary struct type at runtime,
// so the carrier is this dedicated type and the instance travels with it —
// Instance() is the props.instance equivalent.
type Exception struct {
	instance Instance
	original error
	args     []any
}

// NewException wraps original in an error that carries x, mirroring
// `new utils.exception(instance, error, ...args)`. original may be nil.
func NewException(x Instance, original error, args ...any) *Exception {
	result := &Exception{
		instance: x,
		original: original,
		args:     args,
	}
	return result
}

// Error describes the exception and the wrapped error.
func (e *Exception) Error() string {
	result := fmt.Sprintf("mnemonica: exception from %T: %v", e.instance, e.original)
	return result
}

// Unwrap exposes the wrapped error to errors.Is and errors.As.
func (e *Exception) Unwrap() error {
	result := e.original
	return result
}

// Instance returns the instance the exception was made from.
func (e *Exception) Instance() Instance {
	result := e.instance
	return result
}

// Args returns the extra arguments passed after the error (the JS
// props.args).
func (e *Exception) Args() []any {
	result := e.args
	return result
}

// Parsed is the one-level snapshot of an instance (C7 utils.parse).
// Adaptations from the JS Parsed<T>: Parent is the parent INSTANCE (the JS
// fix — not a name), nil for roots; proto/joint are prototype internals
// and have no Go counterpart; deep recursive parsing is not implemented,
// same as the JS.
type Parsed struct {
	Name   string         // the instance's type name
	Props  map[string]any // Extract of the instance (flat, nearest wins)
	Self   Instance       // the instance itself
	Parent Instance       // the parent instance; nil for a root
}

// Parse returns the one-level snapshot of x. An unconstructed value (or
// nil) yields the zero Parsed, the introspective counterpart of
// ErrNotAnInstance.
func Parse(x Instance) Parsed {
	var result Parsed
	record, err := recordOf(x)
	if err != nil {
		return result
	}
	node := x.mnemonicaNode()
	result = Parsed{
		Name:   record.name,
		Props:  Extract(x),
		Self:   x,
		Parent: node.parent,
	}
	return result
}

// ToJSON renders the Extract of x as JSON — always a valid object,
// "{}" when there are no user-visible fields. Keys are escaped by
// encoding/json. Values that encoding/json cannot marshal (functions,
// channels) produce an error rather than invalid output.
func ToJSON(x Instance) (string, error) {
	encoded, err := json.Marshal(Extract(x))
	if err != nil {
		return "", err
	}
	result := string(encoded)
	return result, nil
}

// ConstructorSequence returns the lineage's type names, nearest first:
// x's own type, then the parent's, to the root. The JS walk also appends
// the Mnemonica/Mnemosyne base constructor names; there are no base
// constructors in the Go port, so the sequence stops at the root. The
// walk stops at the first node without a construction record.
func ConstructorSequence(x Instance) []string {
	var result []string
	if nodeOf(x) == nil {
		return result
	}
	for cursor := x; cursor != nil; cursor = cursor.mnemonicaNode().parent {
		record := cursor.mnemonicaNode().record
		if record == nil {
			break
		}
		result = append(result, record.name)
	}
	return result
}

// CollectConstructors returns the constructor-name lookup set along the
// lineage — the JS default (non-sequence) form of collectConstructors.
func CollectConstructors(x Instance) map[string]bool {
	sequence := ConstructorSequence(x)
	result := make(map[string]bool, len(sequence))
	for _, name := range sequence {
		result[name] = true
	}
	return result
}

package mnemonica

import (
	"context"
	"fmt"
	"reflect"
	"runtime/debug"
	"strings"
	"time"
)

// Root is the phantom parent type of every root type. A *Root is not an
// Instance (it embeds no Node), so a root handle's From always fails at the
// Instance assertion: roots are constructed with New. Root exists so
// TypeDef can carry the parent type statically — Go methods cannot take
// their own type parameters, so the parent type must be a type parameter
// of the handle itself.
type Root struct{}

// Type is an untyped handle to a declared type — the result of Lookup.
// It cannot construct typed values (construction needs the struct type),
// but it carries the type's identity and metadata, and New/From on it
// build untyped Instance values for dynamic use, accepting the args as
// any (the one place any appears in the public API, with the Args record).
type Type struct {
	record *typeRecord
}

// Name returns the type's own segment ("Admin" for "User.Admin").
func (t Type) Name() string {
	result := t.record.name
	return result
}

// Path returns the type's dotted path in its collection ("User.Admin").
func (t Type) Path() string {
	result := t.record.path
	return result
}

// Collection returns the collection this type is declared in.
func (t Type) Collection() *Collection {
	result := t.record.collection
	return result
}

// Subtypes returns a sorted snapshot of the types declared directly under
// this type (JS __subtypes__).
func (t Type) Subtypes() []Type {
	result := t.record.subtypeHandles()
	return result
}

// New constructs a root instance of the type, without static typing.
// Prefer the typed TypeDef.New when the type is known at compile time.
func (t Type) New(args any) (Instance, error) {
	constructed, err := t.record.construct(nil, args, nil)
	if err != nil {
		return nil, err
	}
	result := constructed.(Instance)
	return result, nil
}

// From constructs a subtype instance FROM a parent instance, without
// static typing. The parent must be an instance of this type's declared
// parent type, exactly as for the typed TypeDef.From.
func (t Type) From(parent Instance, args any) (Instance, error) {
	constructed, err := t.record.construct(parent, args, nil)
	if err != nil {
		return nil, err
	}
	result := constructed.(Instance)
	return result, nil
}

// TypeDef is the typed handle to a declared type; the value to keep in
// package-level vars (UserT, AdminT). T is the struct type, P the declared
// parent struct type (Root for roots), A the construction args type.
// Construction through it is fully typed: New(args A) returns *T, and a
// subtype handle's From(parent *P, args A) takes the parent BY ITS POINTER
// TYPE — wrong parent kinds do not compile — while the runtime still
// checks parent identity nominally, per type, not just per Go type (C2.2).
type TypeDef[T, P, A any] struct {
	Type
}

// New constructs a root instance of T and runs its handler (C2.1). The
// handler's error propagates according to the type's blockErrors config
// (C4.2): wrapped in an *ErroredInstance when enabled, as-is when not.
// New on a subtype handle fails: subtypes are constructed with From.
func (td *TypeDef[T, P, A]) New(args A) (*T, error) {
	// New records no context (P4): noCtx is nil, named in context.go.
	constructed, err := td.record.construct(nil, args, noCtx)
	if err != nil {
		var zero *T
		return zero, err
	}
	result := constructed.(*T)
	return result, nil
}

// NewCtx is New with a construction context (P4): the ctx's VALUES are
// recorded on the instance, detached from cancellation and deadline, so
// work on the data can outlive the request that built it. A nil ctx is
// exactly New: nothing is recorded, nothing extra is allocated. See
// ContextOf for the read side.
func (td *TypeDef[T, P, A]) NewCtx(ctx context.Context, args A) (*T, error) {
	constructed, err := td.record.construct(nil, args, detachedContext(ctx))
	if err != nil {
		var zero *T
		return zero, err
	}
	result := constructed.(*T)
	return result, nil
}

// From constructs a subtype instance FROM an existing parent instance
// (C2.2): the parent must be an instance of the subtype's declared parent
// type, and the child's embedded *Parent field is wired to that exact
// instance (or, with strictChain off on both sides, to the nearest
// matching ancestor found up the parent's chain — C4.1). Later mutation of
// the parent is visible through the child (C2.6: the parent is shared, not
// copied, and reads promote natively).
func (td *TypeDef[T, P, A]) From(parent *P, args A) (*T, error) {
	// From records no context (P4): noCtx is nil, named in context.go.
	result, err := td.fromDetached(parent, args, noCtx)
	if err != nil {
		var zero *T
		return zero, err
	}
	return result, nil
}

// FromCtx is From with a construction context (P4); see NewCtx for the
// recording semantics.
func (td *TypeDef[T, P, A]) FromCtx(ctx context.Context, parent *P, args A) (*T, error) {
	result, err := td.fromDetached(parent, args, detachedContext(ctx))
	if err != nil {
		var zero *T
		return zero, err
	}
	return result, nil
}

// fromDetached is the shared body of From and FromCtx: validate the parent,
// then construct with the already-detached ctx (nil records nothing).
func (td *TypeDef[T, P, A]) fromDetached(parent *P, args A, recorded context.Context) (*T, error) {
	if parent == nil {
		var zero *T
		err := fmt.Errorf("%w: parent is nil", ErrWrongModificationPattern)
		return zero, err
	}
	parentInstance, ok := any(parent).(Instance)
	if !ok {
		// Unreachable for real parent types (Define/Sub require Node) —
		// only for the Root sentinel, where it keeps root From calls
		// compile-time possible but runtime-rejected.
		var zero *T
		err := fmt.Errorf("%w: parent is not a mnemonica instance", ErrWrongModificationPattern)
		return zero, err
	}
	constructed, err := td.record.construct(parentInstance, args, recorded)
	if err != nil {
		var zero *T
		return zero, err
	}
	result := constructed.(*T)
	return result, nil
}

// SetHandler replaces the constructor body (C1.5). The handler is read at
// every construction, so the swap affects only constructions that happen
// after it — instances built earlier keep whatever their handler wrote.
func (td *TypeDef[T, P, A]) SetHandler(handler func(*T, A) error) {
	wrapped := func(instance any, args any) error {
		err := handler(instance.(*T), args.(A))
		return err
	}
	td.record.handlerMu.Lock()
	td.record.handler = wrapped
	td.record.handlerMu.Unlock()
}

// AttachWire replaces the parent-wiring func with a typed one. The
// generated wire funcs (cmd/mnemonica-gen, P5) attach through this from a
// generated init: package-init ordering runs init before main or any
// test, and construction of package-level handles cannot race it — which
// is why no lock is taken on the per-construction wire read.
func (td *TypeDef[T, P, A]) AttachWire(wire func(*T, *P)) {
	wrapped := func(child, parent any) {
		wire(child.(*T), parent.(*P))
	}
	td.record.wire = wrapped
}

// RegisterHook registers a type-level lifecycle hook (C5.1). See the
// Collection.RegisterHook comment for the execution order.
func (td *TypeDef[T, P, A]) RegisterHook(kind HookKind, hook HookFunc) error {
	if !validHookKind(kind) {
		err := fmt.Errorf("%w: %q", ErrWrongHookType, kind)
		return err
	}
	if hook == nil {
		err := fmt.Errorf("%w: %q", ErrMissingHookCallback, kind)
		return err
	}
	collection := td.record.collection
	collection.mu.Lock()
	td.record.hooks[kind] = append(td.record.hooks[kind], hook)
	collection.mu.Unlock()
	return nil
}

// Must wraps a Define/Sub result for package-level declarations, the way
// template.Must does: it panics on the error and returns the TypeDef
// otherwise. Construction itself never panics — Must is only for
// declaration-time setup that cannot fail in a correct program.
func Must[T, P, A any](td *TypeDef[T, P, A], err error) *TypeDef[T, P, A] {
	if err != nil {
		panic(err)
	}
	return td
}

// Define declares a root type named name in collection (C1.2). T must be a
// struct type embedding mnemonica.Node; handler is the constructor body,
// run on every construction with the single typed args value. The options,
// if any, override the collection's config defaults for this type.
// Declaring the same name twice in one collection fails with
// ErrAlreadyDeclared (C1.3).
func Define[T, A any](collection *Collection, name string, handler func(*T, A) error, options ...Option) (*TypeDef[T, Root, A], error) {
	if err := checkTypeName(name); err != nil {
		return nil, err
	}
	// Every instance must carry the lineage header; a *T that does not
	// implement Instance does not embed Node and can never be constructed.
	var probe *T
	if _, ok := any(probe).(Instance); !ok {
		err := fmt.Errorf("%w: %v must embed mnemonica.Node", ErrWrongTypeDefinition, reflect.TypeOf(probe))
		return nil, err
	}
	state := collectOptions(options)
	record := &typeRecord{
		name:       name,
		path:       name,
		collection: collection,
		children:   make(map[string]*typeRecord),
		config:     state.resolve(collection.defaults),
		alloc: func() any {
			result := new(T)
			return result
		},
		handler: wrapHandler(handler),
		hooks:   make(map[HookKind][]HookFunc),
	}
	collection.mu.Lock()
	defer collection.mu.Unlock()
	if _, exists := collection.roots[name]; exists {
		err := fmt.Errorf("%w: %q", ErrAlreadyDeclared, name)
		return nil, err
	}
	collection.roots[name] = record
	result := &TypeDef[T, Root, A]{Type: Type{record: record}}
	return result, nil
}

// Sub declares a subtype of parent, named name (C1.2). T must be a struct
// type that embeds mnemonica.Node AND embeds *P — the embedded pointer to
// the parent struct — so the parent instance can be wired in and reads
// promote through it. A value embed of P is rejected: it would copy the
// parent and break the shared-history aliasing that makes the lineage real
// (C2.6) — unless WithWireFunc is used, in which case the explicit func
// asserts the shape and the embed validation is skipped. The options, if
// any, override the parent's resolved config per key — the JS subtype
// config inheritance.
func Sub[T, P, GP, PA, A any](parent *TypeDef[P, GP, PA], name string, handler func(*T, A) error, options ...Option) (*TypeDef[T, P, A], error) {
	if err := checkTypeName(name); err != nil {
		return nil, err
	}
	var probe *T
	if _, ok := any(probe).(Instance); !ok {
		err := fmt.Errorf("%w: subtype %v must embed mnemonica.Node", ErrWrongTypeDefinition, reflect.TypeOf(probe))
		return nil, err
	}
	parentRecord := parent.record
	parentType := reflect.TypeOf((*P)(nil)) // *P, e.g. *User; always a pointer here
	state := collectOptions(options)
	var wire func(child, parentInstance any)
	if state.wire != nil {
		childType := reflect.TypeOf(probe)
		if state.wireChild != childType || state.wireParent != parentType {
			err := fmt.Errorf("%w: WithWireFunc has func(%s, %s), but this Sub wires %s under parent %s", ErrWrongTypeDefinition, state.wireChild, state.wireParent, childType, parentType)
			return nil, err
		}
		wire = state.wire
	} else {
		index, err := findParentField(reflect.TypeOf(probe).Elem(), parentType)
		if err != nil {
			return nil, err
		}
		wire = func(child, parentInstance any) {
			// One cached reflect.Set per construction — the accepted P0
			// wiring decision (plain setter, no unsafe), replaced by an
			// explicit wire func when WithWireFunc is given.
			reflect.ValueOf(child).Elem().Field(index).Set(reflect.ValueOf(parentInstance))
		}
	}
	record := &typeRecord{
		name:       name,
		path:       parentRecord.path + "." + name,
		parent:     parentRecord,
		collection: parentRecord.collection,
		children:   make(map[string]*typeRecord),
		config:     state.resolve(parentRecord.config),
		alloc: func() any {
			result := new(T)
			return result
		},
		wire:    wire,
		handler: wrapHandler(handler),
		hooks:   make(map[HookKind][]HookFunc),
	}
	collection := parentRecord.collection
	collection.mu.Lock()
	defer collection.mu.Unlock()
	if _, exists := parentRecord.children[name]; exists {
		err := fmt.Errorf("%w: %q", ErrAlreadyDeclared, record.path)
		return nil, err
	}
	parentRecord.children[name] = record
	result := &TypeDef[T, P, A]{Type: Type{record: record}}
	return result, nil
}

// findParentField reflects ONCE, at Sub time, over the child struct and
// returns the index of the anonymous *Parent field. There is deliberately
// no struct-kind check above: a *T that satisfies Instance is necessarily
// a struct (embedding is struct-only), so one more check could never fire.
func findParentField(childType reflect.Type, parentType reflect.Type) (int, error) {
	for index := 0; index < childType.NumField(); index++ {
		field := childType.Field(index)
		if !field.Anonymous {
			continue
		}
		if field.Type == parentType {
			if field.PkgPath != "" {
				// An unexported embedded field is read-only to reflect:
				// wiring it would panic per construction, so the type
				// definition is rejected here, once.
				err := fmt.Errorf("%w: embedded parent field %s is unexported and cannot be wired", ErrWrongTypeDefinition, field.Name)
				return 0, err
			}
			return index, nil
		}
		if field.Type == parentType.Elem() {
			err := fmt.Errorf("%w: embed *%s, not %s — a value embed copies the parent and the lineage stops being shared", ErrWrongTypeDefinition, parentType.Elem().Name(), parentType.Elem())
			return 0, err
		}
	}
	err := fmt.Errorf("%w: struct has no embedded field of type %v", ErrWrongTypeDefinition, parentType)
	return 0, err
}

// resolveParent validates the parent for construction (C2.2) and returns
// the instance to actually wire in. The runtime check is nominal, per type
// identity — a *User from another collection, or a hand-made &User{} that
// never went through New, does not qualify.
func (r *typeRecord) resolveParent(parent Instance) (Instance, error) {
	if parent == nil {
		if r.parent == nil {
			return nil, nil // root construction
		}
		err := fmt.Errorf("%w: %q is a subtype, construct it with From", ErrWrongModificationPattern, r.path)
		return nil, err
	}
	if r.parent == nil {
		err := fmt.Errorf("%w: %q is a root type, use New", ErrWrongModificationPattern, r.path)
		return nil, err
	}
	parentRecord := parent.mnemonicaNode().record
	if parentRecord == nil {
		// A Node that never went through construction: it has no type
		// identity, so it cannot parent anything.
		err := fmt.Errorf("%w: parent carries no construction record", ErrWrongModificationPattern)
		return nil, err
	}
	if parentRecord == r.parent {
		return parent, nil
	}
	// C4.1, checked at both independent points like the JS: the target
	// type's config AND the given parent's type config must BOTH allow the
	// chain walk-up ("requires both of them to be off"). Go adaptation: a
	// parent of a completely different kind still cannot be wired into the
	// static *P field — strictChain false relaxes to ancestors of the
	// declared parent type, which become the wired parent.
	if r.config.strictChain {
		err := fmt.Errorf("%w: parent of %q must be an instance of %q, not %q (strictChain)", ErrWrongModificationPattern, r.path, r.parent.path, parentRecord.path)
		return nil, err
	}
	if parentRecord.config.strictChain {
		err := fmt.Errorf("%w: parent type %q disallows chain walk-up (strictChain)", ErrWrongModificationPattern, parentRecord.path)
		return nil, err
	}
	for ancestor := parent.mnemonicaNode().parent; ancestor != nil; ancestor = ancestor.mnemonicaNode().parent {
		if ancestor.mnemonicaNode().record == r.parent {
			return ancestor, nil
		}
	}
	err := fmt.Errorf("%w: no instance of %q found up the chain of %q", ErrWrongModificationPattern, r.parent.path, parentRecord.path)
	return nil, err
}

// checkLineageHealthy implements blockErrors (C4.2): refuse to construct
// from any instance whose lineage carries an errored node — the JS config
// table's "prevents construction when errors exist in the prototype
// chain". The walk starts at the parent as given (before any strictChain
// walk-up resolution), so a chain relaxed by strictChain is still checked
// in full.
func checkLineageHealthy(parent Instance) error {
	for cursor := parent; cursor != nil; cursor = cursor.mnemonicaNode().parent {
		if cursor.mnemonicaNode().err != nil {
			err := fmt.Errorf("%w: %q failed to construct", ErrBlocked, cursor.mnemonicaNode().record.path)
			return err
		}
	}
	return nil
}

// construct builds an instance of r through the normal construction rules:
// the parent is validated and resolved first (C2.2, C4.1). The blockErrors
// health check runs on the parent AS GIVEN: a strictChain walk-up may
// resolve to an ancestor, but an errored node anywhere in the given chain
// still blocks. recorded is the detached construction ctx (P4) or nil.
func (r *typeRecord) construct(parent Instance, args any, recorded context.Context) (any, error) {
	resolvedParent, err := r.resolveParent(parent)
	if err != nil {
		return nil, err
	}
	result, err := r.build(resolvedParent, parent, args, recorded)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// constructOnto is the fork().call(thisArg) / merge construction path: a
// ROOT type may be re-parented onto any instance there (the JS .call
// branch bypasses the constructor's parent requirement — that is how two
// roots merge), while a SUBTYPE type validates the parent exactly as From
// does. recorded is the inherited or explicit detached ctx (P4) or nil.
func (r *typeRecord) constructOnto(parent Instance, args any, recorded context.Context) (any, error) {
	if r.parent != nil {
		result, err := r.construct(parent, args, recorded)
		if err != nil {
			return nil, err
		}
		return result, nil
	}
	result, err := r.build(parent, parent, args, recorded)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// build runs the construction pipeline: refuse errored lineages
// (blockErrors, C4.2 — walked from healthRoot, the parent as given), run
// preCreation hooks, allocate and wire with the resolved parent, record
// the detached ctx (P4), run the handler read at that moment (C1.5), then
// run the post hooks. The lineage header is complete before the handler
// runs, so the handler may inspect its own instance with Props.
func (r *typeRecord) build(resolvedParent Instance, healthRoot Instance, args any, recorded context.Context) (any, error) {
	if r.config.blockErrors {
		if err := checkLineageHealthy(healthRoot); err != nil {
			return nil, err
		}
	}
	if r.hasHooks(HookPreCreation) {
		preData := &HookData{
			Type:   Type{record: r},
			Args:   args,
			Parent: resolvedParent,
			Ctx:    hookContext(recorded),
		}
		if err := r.runHooks(HookPreCreation, preData); err != nil {
			return nil, err
		}
	}
	instance := r.alloc()
	inst := instance.(Instance)
	node := inst.mnemonicaNode()
	node.record = r
	node.parent = resolvedParent
	node.args = args
	node.timestamp = time.Now()
	node.collection = r.collection
	node.ctx = recorded
	if r.config.submitStack {
		// Kept as Go gives it; the JS trim removes node internals, and
		// there are none here to remove.
		node.stack = string(debug.Stack())
	}
	if r.wire != nil {
		r.wire(instance, resolvedParent)
	}
	if r.config.snapshot {
		node.snapshot = captureParentSnapshot(resolvedParent)
	}
	handler := r.currentHandler()
	if err := handler(instance, args); err != nil {
		node.err = err
		if !r.config.blockErrors {
			return nil, err
		}
		if r.hasHooks(HookCreationError) {
			errorData := &HookData{
				Type:     Type{record: r},
				Args:     args,
				Parent:   resolvedParent,
				Instance: inst,
				Err:      err,
				Ctx:      hookContext(recorded),
			}
			if hookErr := r.runHooks(HookCreationError, errorData); hookErr != nil {
				return nil, hookErr
			}
		}
		errored := &ErroredInstance{
			Type:     Type{record: r},
			Instance: inst,
			Err:      err,
		}
		return nil, errored
	}
	if r.hasHooks(HookPostCreation) {
		postData := &HookData{
			Type:     Type{record: r},
			Args:     args,
			Parent:   resolvedParent,
			Instance: inst,
			Ctx:      hookContext(recorded),
		}
		if err := r.runHooks(HookPostCreation, postData); err != nil {
			return nil, err
		}
	}
	return instance, nil
}

// captureParentSnapshot copies the user-visible (exported, non-embedded)
// fields of every ancestor at construction time — the baseline for the C2.4
// adapted guard. Shallow: values are copied as they are; mutation through
// pointers nested inside them is not tracked.
func captureParentSnapshot(parent Instance) []SnapshotField {
	var result []SnapshotField
	for cursor := parent; cursor != nil; cursor = cursor.mnemonicaNode().parent {
		value := reflect.ValueOf(cursor).Elem()
		typeOf := value.Type()
		for index := 0; index < value.NumField(); index++ {
			field := typeOf.Field(index)
			if field.PkgPath != "" {
				continue // unexported
			}
			if field.Anonymous {
				continue // the Node header and the parent pointer are not user state
			}
			entry := SnapshotField{
				Type:  typeOf.Name(),
				Field: field.Name,
				Value: value.Field(index).Interface(),
			}
			result = append(result, entry)
		}
	}
	return result
}

// wrapHandler adapts the typed constructor to the untyped storage shared
// with the Lookup-returned Type handle. The args assertion is checked: the
// untyped bridge is exactly where a wrong args value can reach a typed
// handler, and it must fail with a sentinel, not panic.
func wrapHandler[T, A any](handler func(*T, A) error) func(any, any) error {
	argsType := reflect.TypeOf((*A)(nil)).Elem()
	wrapped := func(instance any, args any) error {
		typed, ok := args.(A)
		if !ok {
			err := fmt.Errorf("%w: args is %T, want %v for this type", ErrWrongArgumentsUsed, args, argsType)
			return err
		}
		err := handler(instance.(*T), typed)
		return err
	}
	return wrapped
}

// checkTypeName validates one path segment: non-empty and dot-free, so
// dotted paths stay unambiguous (C1.2).
func checkTypeName(name string) error {
	if name == "" || strings.Contains(name, ".") {
		err := fmt.Errorf("%w: invalid type name %q (non-empty, no dots)", ErrWrongTypeDefinition, name)
		return err
	}
	return nil
}

package mnemonica

import (
	"fmt"
	"sort"
	"sync"
)

// Collection holds the root types of one type tree (C1.1). Create one per
// application domain with NewCollection, optionally passing Option values
// to set collection-level config defaults (C4) or a collection name (used
// by the lineage export), or use the package-level Default. A Collection
// is safe for concurrent use: definition and hook registration mutate the
// registry under a write lock, lookup and construction read it under read
// locks.
type Collection struct {
	mu       sync.RWMutex
	name     string
	roots    map[string]*typeRecord
	defaults config
	hooks    map[HookKind][]HookFunc
}

// NewCollection creates an empty type collection (C1.1). The options, if
// any, become the collection-level config defaults: a root type resolves
// its config against them, and a subtype against its parent's resolved
// config (JS subtype config inheritance). WithCollectionName sets the
// name the lineage export records for the collection's types.
func NewCollection(options ...Option) *Collection {
	state := collectOptions(options)
	result := &Collection{
		roots:    make(map[string]*typeRecord),
		defaults: state.resolve(builtinDefaults()),
		hooks:    make(map[HookKind][]HookFunc),
	}
	if state.collectionName != nil {
		result.name = *state.collectionName
	}
	return result
}

// Name returns the collection's name: the WithCollectionName option given
// to NewCollection, or "default" for the package-level Default. Unnamed
// collections return "".
func (c *Collection) Name() string {
	result := c.name
	return result
}

// RegisterHook registers a collection-level lifecycle hook (C5.1), applied
// to every type in the collection. Order at construction: collection
// preCreation hooks run before type ones; type postCreation/creationError
// hooks run before collection ones (C5.2).
func (c *Collection) RegisterHook(kind HookKind, hook HookFunc) error {
	if !validHookKind(kind) {
		err := fmt.Errorf("%w: %q", ErrWrongHookType, kind)
		return err
	}
	if hook == nil {
		err := fmt.Errorf("%w: %q", ErrMissingHookCallback, kind)
		return err
	}
	c.mu.Lock()
	c.hooks[kind] = append(c.hooks[kind], hook)
	c.mu.Unlock()
	return nil
}

// Default is the package-level default collection, the Go counterpart of
// the JS defaultTypes collection (C1.1). It is named "default".
var Default = NewCollection(WithCollectionName("default"))

// typeRecord is the untyped identity of a declared type, shared by the
// typed handle TypeDef[T, P, A] and the untyped handle Type. It is the
// registry node: one record per dotted path, children keyed by name.
type typeRecord struct {
	name       string
	path       string
	parent     *typeRecord // nil for roots; the declared parent identity
	collection *Collection
	children   map[string]*typeRecord
	config     config

	// alloc builds a fresh *T. Defined once at Define/Sub time: generic
	// functions can new(T) here because TypeDef is parameterised by the
	// struct type itself, not by a pointer type parameter.
	alloc func() any

	// wire connects the child's embedded *Parent field to the parent
	// instance. nil for roots. Built ONCE at Sub time by reflecting on the
	// child struct and caching the field index (the accepted P0 decision):
	// per-construction cost is one cached reflect.Value.Set, no unsafe.
	wire func(child, parent any)

	// handler is the constructor body, read at EVERY construction (C1.5)
	// so SetHandler can swap it at runtime. The mutex keeps a swap from
	// racing a concurrent construction; the handler itself runs unlocked.
	handlerMu sync.RWMutex
	handler   func(instance any, args any) error

	hooks map[HookKind][]HookFunc
}

// currentHandler reads the handler under a read lock; the returned func is
// invoked after the lock is released.
func (r *typeRecord) currentHandler() func(any, any) error {
	r.handlerMu.RLock()
	handler := r.handler
	r.handlerMu.RUnlock()
	return handler
}

// subtypeHandles returns a sorted snapshot of the type's declared
// subtypes. A snapshot (not the live map, as in JS) keeps the registry
// race-free for readers: Sub may add a child while Props is reading.
func (r *typeRecord) subtypeHandles() []Type {
	collection := r.collection
	collection.mu.RLock()
	result := make([]Type, 0, len(r.children))
	for _, child := range r.children {
		result = append(result, Type{record: child})
	}
	collection.mu.RUnlock()
	sort.Slice(result, func(i, j int) bool {
		return result[i].record.name < result[j].record.name
	})
	return result
}

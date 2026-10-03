package mnemonica

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"strconv"
	"sync"
	"sync/atomic"
)

// This file is the L1 lineage export: instance ids, the deep parse walk,
// and the cross-language lineage graph (the FIRST DRAFT of the shared
// format; see github.com/mythographica/lethe, whose lineage.schema.json
// is the contract, and the JS core's plans/lineage-export.md). The
// runtime stays stdlib-only; schema validation of the output lives in
// the tools module.

// ---- instance ids ----

// The id counter is global and atomic; ids are formatted on demand from
// the per-instance counter, so nothing pins an instance: an id'd node is
// garbage-collectable like any other. The counter never resets, so ids
// are never reused, even across GC.
var globalIDCtr atomic.Uint64

// idPrefix is the per-process random prefix, generated once (the JS core
// avoids crypto for front-end bundling; Go has no such constraint, so it
// comes from crypto/rand).
var idPrefix struct {
	sync.Once
	value string
}

// ID returns the instance's stable, process-unique id: a per-process
// random prefix plus the instance's own counter ("a1b2c3d4:17"), assigned
// lazily on the first call — the counter field always rides in Node, but
// no assignment, no prefix generation, and no allocation happen until the
// first ask. A nil (or typed-nil) instance yields "".
func ID(x Instance) string {
	node := nodeOf(x)
	if node == nil {
		return ""
	}
	counter := atomic.LoadUint64(&node.idCounter)
	if counter == 0 {
		counter = assignIDCounter(node)
	}
	result := fmt.Sprintf("%s:%d", idPrefixValue(), counter)
	return result
}

// assignIDCounter publishes a fresh counter into the node with a CAS
// loop: exactly one goroutine wins; losers re-read and return the winner.
// A loser's fresh counter is burned — a harmless, tiny gap in the
// sequence.
func assignIDCounter(node *Node) uint64 {
	for {
		existing := atomic.LoadUint64(&node.idCounter)
		if existing != 0 {
			return existing
		}
		fresh := globalIDCtr.Add(1)
		if atomic.CompareAndSwapUint64(&node.idCounter, 0, fresh) {
			return fresh
		}
	}
}

// idPrefixValue generates the process prefix on first use.
func idPrefixValue() string {
	idPrefix.Do(func() {
		idPrefix.value = newIDPrefix()
	})
	result := idPrefix.value
	return result
}

// newIDPrefix draws 4 random bytes for an 8-hex-char process prefix.
func newIDPrefix() string {
	var bytes [4]byte
	if _, err := processRandomRead(bytes[:]); err != nil {
		// crypto/rand failure is not recoverable; ids only need
		// uniqueness within the process, so fall back to a fixed prefix
		// rather than crash the program.
		return "00000000"
	}
	result := hex.EncodeToString(bytes[:])
	return result
}

// processRandomRead is crypto/rand's Read under a name, so tests can
// exercise the failure fallback.
var processRandomRead = rand.Read

// ---- DeepParse ----

// DeepParse walks the lineage from x to the root, returning one Parsed
// per level — x first, root last — each with Parent set to the parent
// INSTANCE (nil at the root), the shape the JS deepParse settled on. A
// non-instance (or unconstructed) x yields nil.
func DeepParse(x Instance) []Parsed {
	var result []Parsed
	for cursor := x; cursor != nil; {
		if _, _, err := resolveInstance(cursor); err != nil {
			return nil
		}
		result = append(result, Parse(cursor))
		parent, ok := Parent(cursor)
		if !ok {
			break
		}
		cursor = parent
	}
	return result
}

// ---- the lineage graph ----

// GraphTypeRef identifies a node level's type WITHOUT relying on names
// alone (they collide across levels and collections): the collection name
// plus the full dotted path.
type GraphTypeRef struct {
	Collection string `json:"collection"`
	Path       string `json:"path"`
}

// GraphNode is one exported instance. Parent is nil (JSON null) at the
// root. Args and Props appear only when the corresponding options are
// given.
type GraphNode struct {
	Type   GraphTypeRef   `json:"type"`
	Own    map[string]any `json:"own"`
	Parent *string        `json:"parent"`
	Args   any            `json:"args,omitempty"`
	Props  map[string]any `json:"props,omitempty"`
}

// Graph is the exported lineage: the heads (the instances passed to
// Lineage, in argument order) plus every reachable instance, deduplicated
// at any depth, keyed by id.
type Graph struct {
	Version string               `json:"version"`
	Heads   []string             `json:"heads"`
	Nodes   map[string]GraphNode `json:"nodes"`
}

// LineageOption configures what Lineage includes beyond the defaults.
type LineageOption func(*lineageConfig)

type lineageConfig struct {
	withArgs bool
	props    []string
}

// WithArgs includes each node's construction args, sanitised through the
// same placeholder machinery as field values. Off by default.
func WithArgs() LineageOption {
	return func(config *lineageConfig) {
		config.withArgs = true
	}
}

// WithProps includes per-node metadata: currently "timestamp" (RFC 3339).
// Off by default. Unknown keys are an error — silent skips would hide
// typos in a contract-facing format.
func WithProps(keys ...string) LineageOption {
	return func(config *lineageConfig) {
		config.props = append(config.props, keys...)
	}
}

// Lineage exports xs and their ancestors as a deduplicated lineage graph,
// the draft of the cross-language shared format. Instances reachable
// through more than one head — shared parents, siblings, forks — appear
// once, at any depth. It fails with ErrNotAnInstance for a nil or
// unconstructed element.
//
// The signature takes a slice rather than a variadic because Go cannot
// mix variadic instances with variadic options; heads stay in xs order.
func Lineage(xs []Instance, options ...LineageOption) (Graph, error) {
	var config lineageConfig
	for _, option := range options {
		option(&config)
	}
	exporter := lineageExporter{
		graph: Graph{
			Version: "1",
			Heads:   make([]string, 0, len(xs)),
			Nodes:   make(map[string]GraphNode),
		},
		seen: make(map[*Node]bool),
		cfg:  config,
	}
	for _, x := range xs {
		id, err := exporter.exportInstance(x)
		if err != nil {
			return Graph{}, err
		}
		exporter.graph.Heads = append(exporter.graph.Heads, id)
	}
	result := exporter.graph
	return result, nil
}

// lineageExporter carries the graph-under-construction and the dedup set.
type lineageExporter struct {
	graph Graph
	seen  map[*Node]bool
	cfg   lineageConfig
}

// exportInstance assigns the id, encodes own fields (which may pull $ref
// targets in), then exports the parent link — so ids are minted in
// first-encounter order: the node itself, its $ref targets in field
// declaration order, then its parent. That order is part of the shared
// fixture convention (see the recipe README in
// github.com/mythographica/lethe).
func (e *lineageExporter) exportInstance(x Instance) (string, error) {
	_, record, err := resolveInstance(x)
	if err != nil {
		return "", err
	}
	node := x.mnemonicaNode()
	id := ID(x)
	if e.seen[node] {
		return id, nil
	}
	e.seen[node] = true

	own, err := e.encodeStruct(reflect.ValueOf(x).Elem(), make(map[visit]bool))
	if err != nil {
		return "", err
	}
	var parentRef *string
	if node.parent != nil {
		// Parents are constructed by construction (resolveInstance ran
		// for this node; its parent went through build), so the export
		// cannot fail here.
		parentID, _ := e.exportInstance(node.parent)
		parentRef = &parentID
	}
	graphNode := GraphNode{
		Type: GraphTypeRef{
			Collection: record.collection.Name(),
			Path:       record.path,
		},
		Own:    own,
		Parent: parentRef,
	}
	if e.cfg.withArgs {
		args, err := e.encodeValue(reflect.ValueOf(node.args), make(map[visit]bool))
		if err != nil {
			return "", err
		}
		graphNode.Args = args
	}
	if len(e.cfg.props) > 0 {
		props, err := e.encodeProps(node)
		if err != nil {
			return "", err
		}
		graphNode.Props = props
	}
	e.graph.Nodes[id] = graphNode
	return id, nil
}

// encodeProps resolves the supported prop keys. The supported set is
// deliberately small: anything richer belongs to the plugin system, not
// to format options.
func (e *lineageExporter) encodeProps(node *Node) (map[string]any, error) {
	props := make(map[string]any, len(e.cfg.props))
	for _, key := range e.cfg.props {
		switch key {
		case "timestamp":
			props["timestamp"] = node.timestamp
		default:
			err := fmt.Errorf("mnemonica: unknown lineage prop %q (supported: timestamp)", key)
			return nil, err
		}
	}
	return props, nil
}

// visit identities a reference-kind value for cycle detection.
type visit struct {
	typ reflect.Type
	ptr unsafePointer
}

// unsafePointer aliases the pointer word without importing unsafe into
// the API surface; reflect.Value.Pointer returns uintptr which is
// comparable and sufficient as a map key.
type unsafePointer = uintptr

// encodeStruct maps a struct's exported, non-embedded fields — the values
// THAT level set itself, so shadowed fields survive with their writer.
func (e *lineageExporter) encodeStruct(structure reflect.Value, visited map[visit]bool) (map[string]any, error) {
	result := make(map[string]any)
	structType := structure.Type()
	for index := 0; index < structure.NumField(); index++ {
		field := structType.Field(index)
		if field.PkgPath != "" || field.Anonymous {
			continue
		}
		encoded, err := e.encodeValue(structure.Field(index), visited)
		if err != nil {
			return nil, err
		}
		result[field.Name] = encoded
	}
	return result, nil
}

// encodeValue converts one field value into JSON-safe data: primitives
// pass through, instances become $refs (and join the graph), reference
// kinds are cycle-checked, and everything JSON cannot carry becomes a
// tagged placeholder — never an error, never a panic. Field values are
// always valid reflect.Values, so no zero-Value guard is needed.
func (e *lineageExporter) encodeValue(value reflect.Value, visited map[visit]bool) (any, error) {
	if value.CanInterface() {
		if marshaler, ok := value.Interface().(json.Marshaler); ok {
			return marshaler, nil // e.g. time.Time renders as RFC 3339
		}
		if instance, ok := value.Interface().(Instance); ok {
			// A typed-nil *T still satisfies Instance (method set) —
			// export it as null like any other nil pointer.
			if nodeOf(instance) == nil {
				return nil, nil
			}
			return e.encodeRef(instance)
		}
	}
	switch value.Kind() {
	case reflect.Bool:
		result := value.Bool()
		return result, nil
	case reflect.String:
		result := value.String()
		return result, nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		result := value.Int()
		return result, nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		result := value.Uint()
		return result, nil
	case reflect.Float32, reflect.Float64:
		result := encodeFloat(value.Float())
		return result, nil
	case reflect.Complex64, reflect.Complex128:
		return placeholder("complex"), nil
	case reflect.Func, reflect.Chan:
		return placeholder(value.Kind().String()), nil
	case reflect.UnsafePointer:
		if value.IsNil() {
			return nil, nil
		}
		return placeholder("unsafe.Pointer"), nil
	case reflect.Interface:
		if value.IsNil() {
			return nil, nil
		}
		result, err := e.encodeValue(value.Elem(), visited)
		return result, err
	case reflect.Pointer:
		if value.IsNil() {
			return nil, nil
		}
		marker := visit{typ: value.Type(), ptr: value.Pointer()}
		if visited[marker] {
			return placeholder("cycle"), nil
		}
		visited[marker] = true
		result, err := e.encodeValue(value.Elem(), visited)
		return result, err
	case reflect.Struct:
		return e.encodeStruct(value, visited)
	case reflect.Map:
		if value.IsNil() {
			return nil, nil
		}
		marker := visit{typ: value.Type(), ptr: value.Pointer()}
		if visited[marker] {
			return placeholder("cycle"), nil
		}
		visited[marker] = true
		return e.encodeMap(value, visited)
	case reflect.Slice:
		if value.IsNil() {
			return nil, nil
		}
		if value.Len() > 0 {
			marker := visit{typ: value.Type(), ptr: value.Pointer()}
			if visited[marker] {
				return placeholder("cycle"), nil
			}
			visited[marker] = true
		}
		result := make([]any, value.Len())
		for index := 0; index < value.Len(); index++ {
			encoded, err := e.encodeValue(value.Index(index), visited)
			if err != nil {
				return nil, err
			}
			result[index] = encoded
		}
		return result, nil
	case reflect.Array:
		result := make([]any, value.Len())
		for index := 0; index < value.Len(); index++ {
			encoded, err := e.encodeValue(value.Index(index), visited)
			if err != nil {
				return nil, err
			}
			result[index] = encoded
		}
		return result, nil
	default:
		return placeholder(value.Kind().String()), nil
	}
}

// encodeRef exports an instance-valued field as {"$ref": id} and joins
// the referenced instance — with its own chain — into the graph.
func (e *lineageExporter) encodeRef(instance Instance) (any, error) {
	id, err := e.exportInstance(instance)
	if err != nil {
		return nil, err
	}
	result := map[string]any{"$ref": id}
	return result, nil
}

// encodeMap renders string- and integer-keyed maps as objects; any other
// key kind would silently stringify in surprising ways, so the whole map
// becomes a placeholder.
func (e *lineageExporter) encodeMap(value reflect.Value, visited map[visit]bool) (any, error) {
	result := make(map[string]any, value.Len())
	iterator := value.MapRange()
	for iterator.Next() {
		key := iterator.Key()
		var keyString string
		switch key.Kind() {
		case reflect.String:
			keyString = key.String()
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			keyString = strconv.FormatInt(key.Int(), 10)
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			keyString = strconv.FormatUint(key.Uint(), 10)
		default:
			return placeholder("map-keys"), nil
		}
		encoded, err := e.encodeValue(iterator.Value(), visited)
		if err != nil {
			return nil, err
		}
		result[keyString] = encoded
	}
	return result, nil
}

// encodeFloat passes finite floats through and tags the non-finite ones:
// JSON has no NaN or infinities.
func encodeFloat(value float64) any {
	switch {
	case math.IsNaN(value):
		result := placeholder("nan")
		return result
	case math.IsInf(value, 1):
		result := placeholder("+inf")
		return result
	case math.IsInf(value, -1):
		result := placeholder("-inf")
		return result
	default:
		return value
	}
}

// placeholder is the tagged stand-in for values JSON cannot carry: a
// stable tag plus the value's kind. Documented in the schema (see
// github.com/mythographica/lethe).
func placeholder(kind string) map[string]any {
	result := map[string]any{"$mnemonica": "unsupported", "kind": kind}
	return result
}

package lineageschema

import (
	"bytes"
	"encoding/json"
	"math"
	"testing"
	"time"
	"unsafe"

	"github.com/mythographica/lethe"
	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/wentout/mnemonica-go/mnemonica"
)

// schema compiles the shared schema once per test run.
func schema(t *testing.T) *jsonschema.Schema {
	t.Helper()
	document, err := jsonschema.UnmarshalJSON(bytes.NewReader(lethe.LineageSchema))
	if err != nil {
		t.Fatalf("parse schema: %v", err)
	}
	compiler := jsonschema.NewCompiler()
	compiler.AddResource("lineage.schema.json", document)
	compiled, err := compiler.Compile("lineage.schema.json")
	if err != nil {
		t.Fatalf("compile schema: %v", err)
	}
	return compiled
}

// graphOf builds a graph from xs and returns its canonical JSON.
func graphOf(t *testing.T, options []mnemonica.LineageOption, xs ...mnemonica.Instance) []byte {
	t.Helper()
	graph, err := mnemonica.Lineage(xs, options...)
	if err != nil {
		t.Fatalf("Lineage: %v", err)
	}
	raw, err := json.Marshal(graph)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return raw
}

// validate asserts raw is a valid graph per the shared schema.
func validate(t *testing.T, raw []byte) {
	t.Helper()
	var document any
	if err := json.Unmarshal(raw, &document); err != nil {
		t.Fatalf("unmarshal graph: %v", err)
	}
	if err := schema(t).Validate(document); err != nil {
		t.Errorf("graph does not validate: %v\ngraph: %s", err, raw)
	}
}

// fixtureTypes mirrors the shared fixture's shape.
type LineageUser struct {
	mnemonica.Node
	Name string
}

type LineageAdmin struct {
	mnemonica.Node
	*LineageUser
	Role     string
	Attached *LineageUser
}

type LineageSuper struct {
	mnemonica.Node
	*LineageAdmin
	Level int
}

func buildFixture(t *testing.T) (*LineageUser, *LineageAdmin, *LineageAdmin, *LineageSuper) {
	t.Helper()
	collection := mnemonica.NewCollection(mnemonica.WithCollectionName("fixture"))
	userT, err := mnemonica.Define[LineageUser](collection, "User", func(u *LineageUser, name string) error {
		u.Name = name
		return nil
	})
	if err != nil {
		t.Fatalf("Define: %v", err)
	}
	adminT, err := mnemonica.Sub[LineageAdmin](userT, "Admin", func(a *LineageAdmin, role string) error {
		a.Role = role
		return nil
	})
	if err != nil {
		t.Fatalf("Sub: %v", err)
	}
	superT, err := mnemonica.Sub[LineageSuper](adminT, "SuperAdmin", func(s *LineageSuper, level int) error {
		s.Level = level
		return nil
	})
	if err != nil {
		t.Fatalf("Sub: %v", err)
	}
	user, err := userT.New("ada")
	if err != nil {
		t.Fatal(err)
	}
	adminOne, err := adminT.From(user, "root")
	if err != nil {
		t.Fatal(err)
	}
	adminTwo, err := adminT.From(user, "operator")
	if err != nil {
		t.Fatal(err)
	}
	super, err := superT.From(adminOne, 7)
	if err != nil {
		t.Fatal(err)
	}
	adminOne.Attached = user
	return user, adminOne, adminTwo, super
}

func TestSharedFixtureValidates(t *testing.T) {
	validate(t, bytes.TrimSpace(lethe.LineageFixture))
}

// TestSharedFixtureGraphMatches is the byte-for-byte half of the shared
// contract: this port's graph for the fixture script, ids mapped 1:1 in
// first-encounter order, must equal lethe.LineageFixture exactly.
func TestSharedFixtureGraphMatches(t *testing.T) {
	user, adminOne, adminTwo, super := buildFixture(t)
	graph, err := mnemonica.Lineage([]mnemonica.Instance{super, adminTwo})
	if err != nil {
		t.Fatalf("Lineage: %v", err)
	}
	mapping := map[string]string{
		mnemonica.ID(super):    "s",
		mnemonica.ID(adminOne): "a1",
		mnemonica.ID(user):     "u",
		mnemonica.ID(adminTwo): "a2",
	}
	raw, err := json.Marshal(graph)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	remapValue(decoded, mapping)
	got, err := json.Marshal(decoded)
	if err != nil {
		t.Fatalf("re-marshal: %v", err)
	}
	if !bytes.Equal(got, bytes.TrimSpace(lethe.LineageFixture)) {
		t.Errorf("the port's fixture graph differs from the shared fixture:\n%s", got)
	}
}

// remapValue renames ids (heads, node keys, parent, $ref) through the
// decoded graph, mirroring the runtime test's helper.
func remapValue(value any, mapping map[string]string) {
	switch typed := value.(type) {
	case map[string]any:
		for key, item := range typed {
			if key == "parent" || key == "$ref" {
				if id, ok := item.(string); ok {
					if mapped, found := mapping[id]; found {
						typed[key] = mapped
					}
				}
				continue
			}
			if mapped, found := mapping[key]; found {
				delete(typed, key)
				typed[mapped] = item
			}
			remapValue(item, mapping)
		}
	case []any:
		for index, item := range typed {
			if id, ok := item.(string); ok {
				if mapped, found := mapping[id]; found {
					typed[index] = mapped
				}
				continue
			}
			remapValue(item, mapping)
		}
	}
}

func TestProducedFixtureGraphValidates(t *testing.T) {
	_, _, adminTwo, super := buildFixture(t)
	validate(t, graphOf(t, nil, super, adminTwo))
}

// TestPlaceholderShapesValidate: every tagged placeholder kind the
// encoder can emit.
func TestPlaceholderShapesValidate(t *testing.T) {
	type kinds struct {
		mnemonica.Node
		NotANum  float64
		PlusInf  float64
		MinusInf float64
		Cplx     complex128
		Fn       func()
		Ch       chan int
		Ptr      unsafe.Pointer
		UintAddr uintptr
		BadKeys  map[bool]string
		When     time.Time
		Nested   map[string]any
	}
	collection := mnemonica.NewCollection(mnemonica.WithCollectionName("kinds"))
	marker := 0
	kindsT, err := mnemonica.Define[kinds](collection, "Kinds", func(k *kinds, _ string) error {
		k.NotANum = math.NaN()
		k.PlusInf = math.Inf(1)
		k.MinusInf = math.Inf(-1)
		k.Cplx = complex(1, 2)
		k.Fn = func() {}
		k.Ch = make(chan int)
		k.Ptr = unsafe.Pointer(&marker)
		k.UintAddr = 1
		k.BadKeys = map[bool]string{true: "x"}
		k.When = time.Now()
		k.Nested = map[string]any{"key": "value", "list": []any{float64(1), "two"}}
		return nil
	})
	if err != nil {
		t.Fatalf("Define: %v", err)
	}
	instance, err := kindsT.New("seed")
	if err != nil {
		t.Fatal(err)
	}
	validate(t, graphOf(t, nil, instance))
}

type cycleWrap struct {
	Name string
	Self *cycleWrap
}

func TestCycleShapeValidates(t *testing.T) {
	type holder struct {
		mnemonica.Node
		Payload *cycleWrap
	}
	collection := mnemonica.NewCollection(mnemonica.WithCollectionName("cycles"))
	holderT, err := mnemonica.Define[holder](collection, "Holder", func(h *holder, _ string) error {
		h.Payload = &cycleWrap{Name: "loop"}
		h.Payload.Self = h.Payload
		return nil
	})
	if err != nil {
		t.Fatalf("Define: %v", err)
	}
	instance, err := holderT.New("seed")
	if err != nil {
		t.Fatal(err)
	}
	validate(t, graphOf(t, nil, instance))
}

func TestArgsAndPropsValidate(t *testing.T) {
	_, admin, _, _ := buildFixture(t)
	validate(t, graphOf(t, []mnemonica.LineageOption{mnemonica.WithArgs(), mnemonica.WithProps("timestamp")}, admin))
}

// TestInvalidGraphRejected: the validator must actually reject — a graph
// with an extra top-level key and a missing parent must fail.
func TestInvalidGraphRejected(t *testing.T) {
	bad := []byte(`{"version":"1","heads":[],"nodes":{},"extra":true}`)
	var document any
	if err := json.Unmarshal(bad, &document); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if err := schema(t).Validate(document); err == nil {
		t.Error("a graph with an unknown top-level key validated — the schema must reject it")
	}
}

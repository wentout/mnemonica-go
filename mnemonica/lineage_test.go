package mnemonica_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"math"
	"os"
	"reflect"
	"runtime"
	"testing"
	"time"

	"github.com/wentout/mnemonica-go/mnemonica"
)

// Fixture types for the shared lineage fixture — the exact construction
// script is documented in testdata/lineage/README.md at the repo root.

type FixtureUser struct {
	mnemonica.Node
	Name string
}

type FixtureAdmin struct {
	mnemonica.Node
	*FixtureUser
	Role     string
	Attached *FixtureUser
}

type FixtureSuperAdmin struct {
	mnemonica.Node
	*FixtureAdmin
	Level int
}

type fixtureGraph struct {
	collection *mnemonica.Collection
	userT      *mnemonica.TypeDef[FixtureUser, mnemonica.Root, string]
	adminT     *mnemonica.TypeDef[FixtureAdmin, FixtureUser, string]
	superT     *mnemonica.TypeDef[FixtureSuperAdmin, FixtureAdmin, int]
}

// buildFixtureGraph constructs the shared-fixture script: collection
// "fixture", User → Admin → SuperAdmin, u; a1, a2 (siblings) from u; s
// from a1; a1.Attached = u (the $ref field).
func buildFixtureGraph(t *testing.T) (fixtureGraph, *FixtureUser, *FixtureAdmin, *FixtureAdmin, *FixtureSuperAdmin) {
	t.Helper()
	collection := mnemonica.NewCollection(mnemonica.WithCollectionName("fixture"))
	userT, err := mnemonica.Define[FixtureUser](collection, "User", func(u *FixtureUser, name string) error {
		u.Name = name
		return nil
	})
	if err != nil {
		t.Fatalf("Define: %v", err)
	}
	adminT, err := mnemonica.Sub[FixtureAdmin](userT, "Admin", func(a *FixtureAdmin, role string) error {
		a.Role = role
		return nil
	})
	if err != nil {
		t.Fatalf("Sub: %v", err)
	}
	superT, err := mnemonica.Sub[FixtureSuperAdmin](adminT, "SuperAdmin", func(s *FixtureSuperAdmin, level int) error {
		s.Level = level
		return nil
	})
	if err != nil {
		t.Fatalf("Sub: %v", err)
	}
	user, err := userT.New("ada")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	adminOne, err := adminT.From(user, "root")
	if err != nil {
		t.Fatalf("From: %v", err)
	}
	adminTwo, err := adminT.From(user, "operator")
	if err != nil {
		t.Fatalf("From: %v", err)
	}
	super, err := superT.From(adminOne, 7)
	if err != nil {
		t.Fatalf("From: %v", err)
	}
	adminOne.Attached = user
	return fixtureGraph{collection: collection, userT: userT, adminT: adminT, superT: superT},
		user, adminOne, adminTwo, super
}

func TestIDStableAndUnique(t *testing.T) {
	_, user, admin, _, super := buildFixtureGraph(t)
	first := mnemonica.ID(user)
	if first == "" {
		t.Fatal("ID is empty")
	}
	if second := mnemonica.ID(user); second != first {
		t.Errorf("ID not stable: %q then %q", first, second)
	}
	seen := map[string]bool{first: true, mnemonica.ID(admin): true, mnemonica.ID(super): true}
	if len(seen) != 3 {
		t.Error("distinct instances must get distinct ids")
	}
}

func TestIDNilInputs(t *testing.T) {
	if id := mnemonica.ID(nil); id != "" {
		t.Errorf("ID(nil) = %q, want empty", id)
	}
	var nilUser *FixtureUser
	if id := mnemonica.ID(nilUser); id != "" {
		t.Errorf("ID((*FixtureUser)(nil)) = %q, want empty", id)
	}
}

// TestIDLazy: the id counter rides in Node, but no assignment, no prefix
// generation, and no allocation happen until the first ID call —
// construction allocations stay at the pre-L1 count.
func TestIDLazy(t *testing.T) {
	fx := newFixture()
	allocs := testing.AllocsPerRun(100, func() {
		_, err := fx.userT.New("ada")
		if err != nil {
			t.Fatal(err)
		}
	})
	if allocs > 2 {
		t.Errorf("New allocates %v after the id feature, want <= 2 (ids are lazy)", allocs)
	}
}

// TestIDConcurrentAssign: many goroutines asking for a fresh instance's
// id at once, over many rounds — exactly one CAS wins per instance,
// everyone gets the same value, and the burn-a-counter loser path is
// exercised (a lost race is near-certain across 100 barrier-synced
// rounds on a multicore machine).
func TestIDConcurrentAssign(t *testing.T) {
	fx := newFixture()
	const workers = 64
	const rounds = 100
	start := make(chan struct{})
	ids := make(chan string, workers)
	for round := 0; round < rounds; round++ {
		user := lawMust(fx.userT.New("ada"))
		for w := 0; w < workers; w++ {
			go func() {
				<-start
				ids <- mnemonica.ID(user)
			}()
		}
		close(start)
		want := mnemonica.ID(user)
		seen := make(map[string]bool, workers)
		for w := 0; w < workers; w++ {
			seen[<-ids] = true
		}
		if len(seen) != 1 || !seen[want] {
			t.Fatalf("round %d: concurrent IDs = %v, want exactly %q for everyone", round, seen, want)
		}
		start = make(chan struct{})
	}
}

// TestIDCollectable is the L1 leak fix witness: an instance that GOT an
// id must still be garbage-collectable — no registry pins it. The
// instance is built inside a helper that returns only its id, so the
// test cannot hold it alive; runtime.AddCleanup signals collection.
func TestIDCollectable(t *testing.T) {
	done := make(chan struct{})
	id := idOfExportedInstance(t, done)
	if id == "" {
		t.Fatal("no id assigned")
	}
	deadline := time.After(10 * time.Second)
	for {
		runtime.GC()
		select {
		case <-done:
			return // the id'd instance was collected
		case <-deadline:
			t.Fatal("an instance that got an id was not collected within 10s — the id store pins instances")
		default:
			runtime.Gosched()
		}
	}
}

// idOfExportedInstance constructs an instance, assigns its id, and
// returns ONLY the id string — the caller never sees the instance, so
// nothing but the cleanup can keep it alive.
func idOfExportedInstance(t *testing.T, done chan struct{}) string {
	t.Helper()
	collection := mnemonica.NewCollection()
	userT := lawMust(mnemonica.Define[FixtureUser](collection, "GcUser", func(u *FixtureUser, name string) error {
		u.Name = name
		return nil
	}))
	user := lawMust(userT.New("ada"))
	id := mnemonica.ID(user)
	runtime.AddCleanup(user, func(signal chan struct{}) {
		close(signal)
	}, done)
	return id
}

func TestDeepParse(t *testing.T) {
	_, user, admin, _, super := buildFixtureGraph(t)
	levels := mnemonica.DeepParse(super)
	if len(levels) != 3 {
		t.Fatalf("DeepParse length = %d, want 3 (instance first, root last)", len(levels))
	}
	if levels[0].Self != mnemonica.Instance(super) || levels[0].Name != "SuperAdmin" {
		t.Errorf("level 0 = (%v, %q), want the super", levels[0].Self, levels[0].Name)
	}
	if levels[1].Self != mnemonica.Instance(admin) {
		t.Error("level 1 is not the admin")
	}
	if levels[2].Self != mnemonica.Instance(user) {
		t.Error("level 2 is not the user")
	}
	// Each level's Parent is the parent INSTANCE; the root's is nil.
	if levels[0].Parent != mnemonica.Instance(admin) {
		t.Error("super's Parent is not the admin instance")
	}
	if levels[1].Parent != mnemonica.Instance(user) {
		t.Error("admin's Parent is not the user instance")
	}
	if levels[2].Parent != nil {
		t.Error("root's Parent is not nil")
	}
}

func TestDeepParseNonInstance(t *testing.T) {
	if levels := mnemonica.DeepParse(nil); levels != nil {
		t.Errorf("DeepParse(nil) = %v, want nil", levels)
	}
	if levels := mnemonica.DeepParse(&FixtureUser{}); levels != nil {
		t.Errorf("DeepParse(unconstructed) = %v, want nil", levels)
	}
}

func TestLineageSharedFixture(t *testing.T) {
	_, user, adminOne, adminTwo, super := buildFixtureGraph(t)
	graph, err := mnemonica.Lineage([]mnemonica.Instance{super, adminTwo})
	if err != nil {
		t.Fatalf("Lineage: %v", err)
	}
	if graph.Version != "1" {
		t.Errorf("version = %q, want %q", graph.Version, "1")
	}
	// The user is shared by s, a1, and a2 — it appears once.
	if len(graph.Nodes) != 4 {
		t.Fatalf("nodes = %d, want 4 (the shared user appears once)", len(graph.Nodes))
	}
	// Heads are in argument order.
	if len(graph.Heads) != 2 || graph.Heads[0] != mnemonica.ID(super) || graph.Heads[1] != mnemonica.ID(adminTwo) {
		t.Errorf("heads = %v, want [super, adminTwo] ids", graph.Heads)
	}
	userID := mnemonica.ID(user)
	adminOneID := mnemonica.ID(adminOne)
	// s's parent is a1; a1's parent is the user; the user's parent is null.
	if graph.Nodes[mnemonica.ID(super)].Parent == nil || *graph.Nodes[mnemonica.ID(super)].Parent != adminOneID {
		t.Error("super's parent link is wrong")
	}
	if graph.Nodes[adminOneID].Parent == nil || *graph.Nodes[adminOneID].Parent != userID {
		t.Error("admin's parent link is wrong")
	}
	if graph.Nodes[userID].Parent != nil {
		t.Error("root's parent is not null")
	}
	// own carries that level's own fields: a1 has Role AND the $ref.
	own := graph.Nodes[adminOneID].Own
	if own["Role"] != "root" {
		t.Errorf("a1 own Role = %v, want root", own["Role"])
	}
	ref, ok := own["Attached"].(map[string]any)
	if !ok || ref["$ref"] != userID {
		t.Errorf("a1 own Attached = %v, want {$ref: %s}", own["Attached"], userID)
	}
	// u appears once although three nodes point at it.
	if graph.Nodes[userID].Own["Name"] != "ada" {
		t.Errorf("user own Name = %v, want ada", graph.Nodes[userID].Own["Name"])
	}
}

// TestLineageSharedFixtureBytes pins the byte-exact shared graph: ids
// are per-process, so the test maps its ids onto the fixture's semantic
// ids (documented in testdata/lineage/README.md) and compares JSON.
func TestLineageSharedFixtureBytes(t *testing.T) {
	_, user, adminOne, adminTwo, super := buildFixtureGraph(t)
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
	normalised := remapGraph(t, graph, mapping)
	got, err := json.Marshal(normalised)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	want, err := os.ReadFile("../testdata/lineage/fixture.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	if !bytes.Equal(got, bytes.TrimSpace(want)) {
		t.Errorf("lineage graph differs from the shared fixture:\n%s", got)
	}
}

// remapGraph renames ids (heads, node keys, parent, $ref, args, props)
// through a JSON round trip, so the comparison happens on plain data.
func remapGraph(t *testing.T, graph mnemonica.Graph, mapping map[string]string) map[string]any {
	t.Helper()
	raw, err := json.Marshal(graph)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	remapValue(decoded, mapping)
	return decoded
}

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
				// node ids are map keys under "nodes"
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

// TestLineageForkDedup: a fork is a sibling under the same parent — the
// parent and everything above it export once.
func TestLineageForkDedup(t *testing.T) {
	fx := newFixture()
	user := lawMust(fx.userT.New("ada"))
	admin := lawMust(fx.adminT.From(user, "root"))
	fork, err := mnemonica.Fork(admin, "operator")
	if err != nil {
		t.Fatalf("Fork: %v", err)
	}
	graph, err := mnemonica.Lineage([]mnemonica.Instance{admin, fork})
	if err != nil {
		t.Fatalf("Lineage: %v", err)
	}
	if len(graph.Nodes) != 3 {
		t.Errorf("nodes = %d, want 3 (user exported once for both siblings)", len(graph.Nodes))
	}
	if graph.Nodes[mnemonica.ID(fork)].Parent == nil ||
		*graph.Nodes[mnemonica.ID(fork)].Parent != mnemonica.ID(user) {
		t.Error("fork's parent link is wrong")
	}
}

// TestLineageRefChain: a $ref target brings its own chain with it.
func TestLineageRefChain(t *testing.T) {
	fx, user, _, _, _ := buildFixtureGraph(t)
	other, err := fx.userT.New("grace")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	carrierT, err := mnemonica.Sub[FixtureAdmin](fx.userT, "Carrier",
		func(a *FixtureAdmin, role string) error {
			a.Role = role
			return nil
		})
	if err != nil {
		t.Fatalf("Sub: %v", err)
	}
	carrier := lawMust(carrierT.From(user, "carrier"))
	carrier.Attached = other
	graph, err := mnemonica.Lineage([]mnemonica.Instance{carrier})
	if err != nil {
		t.Fatalf("Lineage: %v", err)
	}
	// nodes: carrier, user (carrier's parent), other (the $ref target).
	if len(graph.Nodes) != 3 {
		t.Fatalf("nodes = %d, want 3 (the $ref target joined with its own record)", len(graph.Nodes))
	}
	otherID := mnemonica.ID(other)
	if _, ok := graph.Nodes[otherID]; !ok {
		t.Error("the $ref target is not in nodes")
	}
	if ref := graph.Nodes[mnemonica.ID(carrier)].Own["Attached"].(map[string]any); ref["$ref"] != otherID {
		t.Errorf("carrier's Attached = %v, want $ref to the other user", ref)
	}
	// "other" is a root: its parent is null.
	if graph.Nodes[otherID].Parent != nil {
		t.Error("the $ref target's parent is not null")
	}
}

// placeholderKinds covers every non-JSON value the encoder must tag.
type placeholderKinds struct {
	mnemonica.Node
	Flag     bool
	Text     string
	Small    int8
	Count    int
	Big      uint64
	Ratio    float64
	NotANum  float64
	PlusInf  float64
	MinusInf float64
	Cplx     complex128
	Fn       func()
	Ch       chan int
	Any      any
	Tags     map[int]string
	Words    map[string]string
	Scores   []int
	Empty    []string
	Pairs    [2]string
	When     time.Time
	Nested   struct {
		Inner string
	}
}

func TestLineagePlaceholders(t *testing.T) {
	collection := mnemonica.NewCollection(mnemonica.WithCollectionName("kinds"))
	typesT, err := mnemonica.Define[placeholderKinds](collection, "Kinds", func(k *placeholderKinds, _ string) error {
		k.Flag = true
		k.Text = "hello"
		k.Small = -3
		k.Count = 42
		k.Big = 9
		k.Ratio = 1.5
		k.NotANum = math.NaN()
		k.PlusInf = math.Inf(1)
		k.MinusInf = math.Inf(-1)
		k.Cplx = complex(1, 2)
		k.Fn = func() {}
		k.Ch = make(chan int)
		k.Tags = map[int]string{7: "seven"}
		k.Words = map[string]string{"a": "b"}
		k.Scores = []int{1, 2}
		k.Empty = []string{}
		k.Pairs = [2]string{"x", "y"}
		k.When = time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)
		k.Nested.Inner = "deep"
		return nil
	})
	if err != nil {
		t.Fatalf("Define: %v", err)
	}
	instance := lawMust(typesT.New("seed"))
	graph, err := mnemonica.Lineage([]mnemonica.Instance{instance})
	if err != nil {
		t.Fatalf("Lineage: %v", err)
	}
	own := graph.Nodes[mnemonica.ID(instance)].Own
	if own["Flag"] != true || own["Text"] != "hello" || own["Small"] != int64(-3) ||
		own["Count"] != int64(42) || own["Big"] != uint64(9) || own["Ratio"] != 1.5 {
		t.Errorf("primitives lost in own: %v", own)
	}
	for field, kind := range map[string]string{
		"NotANum": "nan", "PlusInf": "+inf", "MinusInf": "-inf",
		"Cplx": "complex", "Fn": "func", "Ch": "chan",
	} {
		tag, ok := own[field].(map[string]any)
		if !ok || tag["$mnemonica"] != "unsupported" || tag["kind"] != kind {
			t.Errorf("own[%s] = %v, want unsupported/%s placeholder", field, own[field], kind)
		}
	}
	if own["Any"] != nil {
		t.Errorf("nil interface = %v, want null", own["Any"])
	}
	if tags, ok := own["Tags"].(map[string]any); !ok || tags["7"] != "seven" {
		t.Errorf("int-keyed map = %v, want {7: seven}", own["Tags"])
	}
	if own["Empty"] == nil {
		t.Error("non-nil empty slice must export as [], not null")
	}
	if len(own["Scores"].([]any)) != 2 || own["Pairs"].([]any)[1] != "y" {
		t.Errorf("slice/array lost: %v", own)
	}
	when, err := json.Marshal(own["When"])
	if err != nil || !bytes.Contains(when, []byte("2026-10-02T12:00:00")) {
		t.Errorf("time.Time = %s (%v), want an RFC 3339 string", when, err)
	}
	if nested, ok := own["Nested"].(map[string]any); !ok || nested["Inner"] != "deep" {
		t.Errorf("nested struct = %v", own["Nested"])
	}
}

// TestLineageCycles: pointer fields back to the level itself export as a
// cycle placeholder instead of recursing forever.
func TestLineageCycles(t *testing.T) {
	type wrap struct {
		Name string
		Self *wrap
	}
	type holder struct {
		mnemonica.Node
		Payload *wrap
		Spare   *wrap
		Missing []string
	}
	collection := mnemonica.NewCollection(mnemonica.WithCollectionName("cycles"))
	holderT, err := mnemonica.Define[holder](collection, "Holder", func(h *holder, name string) error {
		h.Payload = &wrap{Name: name}
		h.Payload.Self = h.Payload // pointer cycle
		return nil
	})
	if err != nil {
		t.Fatalf("Define: %v", err)
	}
	instance := lawMust(holderT.New("loopy"))
	graph, err := mnemonica.Lineage([]mnemonica.Instance{instance})
	if err != nil {
		t.Fatalf("Lineage: %v", err)
	}
	payload, ok := graph.Nodes[mnemonica.ID(instance)].Own["Payload"].(map[string]any)
	if !ok {
		t.Fatalf("payload = %v", graph.Nodes[mnemonica.ID(instance)].Own["Payload"])
	}
	if payload["Name"] != "loopy" {
		t.Errorf("payload.Name = %v", payload["Name"])
	}
	tag, ok := payload["Self"].(map[string]any)
	if !ok || tag["$mnemonica"] != "unsupported" || tag["kind"] != "cycle" {
		t.Errorf("cycle field = %v, want a cycle placeholder", payload["Self"])
	}
	if own := graph.Nodes[mnemonica.ID(instance)].Own; own["Spare"] != nil || own["Missing"] != nil {
		t.Errorf("nil pointer/slice fields = %v, want nulls", own)
	}
}

func TestLineageBadMapKeys(t *testing.T) {
	type keyed struct {
		mnemonica.Node
		Lookup map[bool]string
	}
	collection := mnemonica.NewCollection(mnemonica.WithCollectionName("keys"))
	keyedT, err := mnemonica.Define[keyed](collection, "Keyed", func(k *keyed, _ string) error {
		k.Lookup = map[bool]string{true: "yes"}
		return nil
	})
	if err != nil {
		t.Fatalf("Define: %v", err)
	}
	instance := lawMust(keyedT.New("seed"))
	graph, err := mnemonica.Lineage([]mnemonica.Instance{instance})
	if err != nil {
		t.Fatalf("Lineage: %v", err)
	}
	tag, ok := graph.Nodes[mnemonica.ID(instance)].Own["Lookup"].(map[string]any)
	if !ok || tag["$mnemonica"] != "unsupported" || tag["kind"] != "map-keys" {
		t.Errorf("bool-keyed map = %v, want a map-keys placeholder", graph.Nodes[mnemonica.ID(instance)].Own["Lookup"])
	}
}

func TestLineageRefViaInterface(t *testing.T) {
	fx, user, _, _, _ := buildFixtureGraph(t)
	carrier := lawMust(fx.adminT.From(user, "box"))
	var asAny any = user
	collection := mnemonica.NewCollection(mnemonica.WithCollectionName("iface"))
	ifaceT, err := mnemonica.Define[struct {
		mnemonica.Node
		Held any
	}](collection, "Iface", func(i *struct {
		mnemonica.Node
		Held any
	}, _ string) error {
		i.Held = asAny
		return nil
	})
	if err != nil {
		t.Fatalf("Define: %v", err)
	}
	carrierInstance := lawMust(ifaceT.New("carrier"))
	_ = carrier
	graph, err := mnemonica.Lineage([]mnemonica.Instance{carrierInstance})
	if err != nil {
		t.Fatalf("Lineage: %v", err)
	}
	held, ok := graph.Nodes[mnemonica.ID(carrierInstance)].Own["Held"].(map[string]any)
	if !ok || held["$ref"] != mnemonica.ID(user) {
		t.Errorf("interface-held instance = %v, want $ref", graph.Nodes[mnemonica.ID(carrierInstance)].Own["Held"])
	}
	if _, ok := graph.Nodes[mnemonica.ID(user)]; !ok {
		t.Error("interface-held instance did not join nodes")
	}
}

func TestLineageArgsAndProps(t *testing.T) {
	_, user, admin, _, _ := buildFixtureGraph(t)

	without, err := mnemonica.Lineage([]mnemonica.Instance{admin})
	if err != nil {
		t.Fatalf("Lineage: %v", err)
	}
	node := without.Nodes[mnemonica.ID(admin)]
	if node.Args != nil || node.Props != nil {
		t.Error("args/props must be off by default")
	}

	with, err := mnemonica.Lineage([]mnemonica.Instance{admin}, mnemonica.WithArgs(), mnemonica.WithProps("timestamp"))
	if err != nil {
		t.Fatalf("Lineage: %v", err)
	}
	node = with.Nodes[mnemonica.ID(admin)]
	if node.Args != "root" {
		t.Errorf("args = %v, want root", node.Args)
	}
	timestamp, err := json.Marshal(node.Props["timestamp"])
	if err != nil || !bytes.Contains(timestamp, []byte("T")) {
		t.Errorf("timestamp prop = %s (%v), want an RFC 3339 string", timestamp, err)
	}
	_ = user
}

// TestLineageArgsSanitised: args go through the same placeholder
// machinery — non-JSON args never fail the export.
func TestLineageArgsSanitised(t *testing.T) {
	type spicy struct {
		mnemonica.Node
		Note string
	}
	type spicyArgs struct {
		Fn func()
	}
	collection := mnemonica.NewCollection(mnemonica.WithCollectionName("args"))
	spicyT, err := mnemonica.Define[spicy](collection, "Spicy", func(s *spicy, args spicyArgs) error {
		s.Note = "x"
		_ = args
		return nil
	})
	if err != nil {
		t.Fatalf("Define: %v", err)
	}
	instance := lawMust(spicyT.New(spicyArgs{Fn: func() {}}))
	graph, err := mnemonica.Lineage([]mnemonica.Instance{instance}, mnemonica.WithArgs())
	if err != nil {
		t.Fatalf("Lineage: %v", err)
	}
	args, ok := graph.Nodes[mnemonica.ID(instance)].Args.(map[string]any)
	if !ok {
		t.Fatalf("args = %#v", graph.Nodes[mnemonica.ID(instance)].Args)
	}
	tag, ok := args["Fn"].(map[string]any)
	if !ok || tag["$mnemonica"] != "unsupported" || tag["kind"] != "func" {
		t.Errorf("args.Fn = %v, want a placeholder", args["Fn"])
	}
}

func TestLineageUnknownProp(t *testing.T) {
	_, _, admin, _, _ := buildFixtureGraph(t)
	_, err := mnemonica.Lineage([]mnemonica.Instance{admin}, mnemonica.WithProps("nosuch"))
	if err == nil {
		t.Fatal("unknown prop succeeded, want an error")
	}
}

func TestLineageErrors(t *testing.T) {
	if _, err := mnemonica.Lineage(nil); err != nil {
		t.Errorf("Lineage(nil slice) = %v, want an empty graph", err)
	}
	if _, err := mnemonica.Lineage([]mnemonica.Instance{nil}); !errors.Is(err, mnemonica.ErrNotAnInstance) {
		t.Errorf("Lineage([nil]) error = %v, want ErrNotAnInstance", err)
	}
	if _, err := mnemonica.Lineage([]mnemonica.Instance{&FixtureUser{}}); !errors.Is(err, mnemonica.ErrNotAnInstance) {
		t.Errorf("Lineage([unconstructed]) error = %v, want ErrNotAnInstance", err)
	}
}

// TestLineageRefUnconstructed: a $ref to an unconstructed instance is an
// error, not a panic — the graph is a contract.
func TestLineageRefUnconstructed(t *testing.T) {
	_, _, admin, _, _ := buildFixtureGraph(t)
	admin.Attached = &FixtureUser{}
	if _, err := mnemonica.Lineage([]mnemonica.Instance{admin}); !errors.Is(err, mnemonica.ErrNotAnInstance) {
		t.Errorf("Lineage with an unconstructed $ref error = %v, want ErrNotAnInstance", err)
	}
}

// TestLineageJSONRoundTrip: marshal → unmarshal → marshal is the identity
// — the graph survives a JSON boundary unchanged (compared canonically:
// both sides decoded to maps).
func TestLineageJSONRoundTrip(t *testing.T) {
	_, _, _, adminTwo, super := buildFixtureGraph(t)
	graph, err := mnemonica.Lineage([]mnemonica.Instance{super, adminTwo}, mnemonica.WithArgs(), mnemonica.WithProps("timestamp"))
	if err != nil {
		t.Fatalf("Lineage: %v", err)
	}
	first, err := json.Marshal(graph)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(first, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	second, err := json.Marshal(decoded)
	if err != nil {
		t.Fatalf("re-marshal: %v", err)
	}
	var again map[string]any
	if err := json.Unmarshal(second, &again); err != nil {
		t.Fatalf("re-unmarshal: %v", err)
	}
	if !reflect.DeepEqual(decoded, again) {
		t.Errorf("round trip changed the graph:\nfirst:  %s\nsecond: %s", first, second)
	}
}

// TestLineageOwnSkipsEmbedded: the embedded parent and the Node header
// never appear in own — shadowing keeps each writer's value with the
// writer.
func TestLineageOwnSkipsEmbedded(t *testing.T) {
	_, _, _, adminTwo, _ := buildFixtureGraph(t)
	graph, err := mnemonica.Lineage([]mnemonica.Instance{adminTwo})
	if err != nil {
		t.Fatalf("Lineage: %v", err)
	}
	own := graph.Nodes[mnemonica.ID(adminTwo)].Own
	if _, ok := own["FixtureUser"]; ok {
		t.Error("the embedded parent leaked into own")
	}
	if _, ok := own["Node"]; ok {
		t.Error("the Node header leaked into own")
	}
	if len(own) != 2 || own["Role"] != "operator" || own["Attached"] != nil {
		t.Errorf("own = %v, want Role plus a null Attached (nil pointer exported as null)", own)
	}
}

func TestCollectionName(t *testing.T) {
	named := mnemonica.NewCollection(mnemonica.WithCollectionName("fixture"))
	if name := named.Name(); name != "fixture" {
		t.Errorf("Name() = %q, want fixture", name)
	}
	unnamed := mnemonica.NewCollection()
	if name := unnamed.Name(); name != "" {
		t.Errorf("unnamed Name() = %q, want empty", name)
	}
	if name := mnemonica.Default.Name(); name != "default" {
		t.Errorf("Default Name() = %q, want default", name)
	}
}

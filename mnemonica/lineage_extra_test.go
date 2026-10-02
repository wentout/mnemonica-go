package mnemonica_test

import (
	"errors"
	"testing"
	"unsafe"

	"github.com/wentout/mnemonica-go/mnemonica"
)

// The remaining encoder branches: kinds and paths the main battery does
// not reach, and the error paths that must stay errors (not panics).

func TestLineageUintptrAndUnsafePointer(t *testing.T) {
	type raw struct {
		mnemonica.Node
		Addr      uintptr
		UnsafePtr unsafe.Pointer
	}
	collection := mnemonica.NewCollection(mnemonica.WithCollectionName("raw"))
	rawT, err := mnemonica.Define[raw](collection, "Raw", func(r *raw, _ string) error {
		r.Addr = 42
		r.UnsafePtr = nil
		return nil
	})
	if err != nil {
		t.Fatalf("Define: %v", err)
	}
	instance := lawMust(rawT.New("seed"))
	graph, err := mnemonica.Lineage([]mnemonica.Instance{instance})
	if err != nil {
		t.Fatalf("Lineage: %v", err)
	}
	own := graph.Nodes[mnemonica.ID(instance)].Own
	if tag, ok := own["Addr"].(map[string]any); !ok || tag["kind"] != "uintptr" {
		t.Errorf("uintptr field = %v, want a placeholder", own["Addr"])
	}
	if own["UnsafePtr"] != nil {
		t.Errorf("nil unsafe.Pointer = %v, want null", own["UnsafePtr"])
	}
}

func TestLineageUnsafePointerSet(t *testing.T) {
	type raw struct {
		mnemonica.Node
		UnsafePtr unsafe.Pointer
	}
	collection := mnemonica.NewCollection(mnemonica.WithCollectionName("rawset"))
	rawT, err := mnemonica.Define[raw](collection, "Raw", func(r *raw, _ string) error {
		marker := 0
		r.UnsafePtr = unsafe.Pointer(&marker)
		return nil
	})
	if err != nil {
		t.Fatalf("Define: %v", err)
	}
	instance := lawMust(rawT.New("seed"))
	graph, err := mnemonica.Lineage([]mnemonica.Instance{instance})
	if err != nil {
		t.Fatalf("Lineage: %v", err)
	}
	tag, ok := graph.Nodes[mnemonica.ID(instance)].Own["UnsafePtr"].(map[string]any)
	if !ok || tag["kind"] != "unsafe.Pointer" {
		t.Errorf("unsafe.Pointer field = %v, want a placeholder", graph.Nodes[mnemonica.ID(instance)].Own["UnsafePtr"])
	}
}

func TestLineageNilMap(t *testing.T) {
	type mapped struct {
		mnemonica.Node
		Lookup map[string]string
	}
	collection := mnemonica.NewCollection(mnemonica.WithCollectionName("nilmap"))
	mappedT, err := mnemonica.Define[mapped](collection, "Mapped", func(m *mapped, _ string) error {
		return nil
	})
	if err != nil {
		t.Fatalf("Define: %v", err)
	}
	instance := lawMust(mappedT.New("seed"))
	graph, err := mnemonica.Lineage([]mnemonica.Instance{instance})
	if err != nil {
		t.Fatalf("Lineage: %v", err)
	}
	if own := graph.Nodes[mnemonica.ID(instance)].Own; own["Lookup"] != nil {
		t.Errorf("nil map = %v, want null", own["Lookup"])
	}
}

func TestLineageUintKeyedMap(t *testing.T) {
	type keyed struct {
		mnemonica.Node
		Scores map[uint8]int
	}
	collection := mnemonica.NewCollection(mnemonica.WithCollectionName("uintkeys"))
	keyedT, err := mnemonica.Define[keyed](collection, "Keyed", func(k *keyed, _ string) error {
		k.Scores = map[uint8]int{3: 33}
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
	scores, ok := graph.Nodes[mnemonica.ID(instance)].Own["Scores"].(map[string]any)
	if !ok || scores["3"] != int64(33) {
		t.Errorf("uint-keyed map = %v, want {3: 33}", graph.Nodes[mnemonica.ID(instance)].Own["Scores"])
	}
}

// TestLineageMapValueError: an unconstructed instance nested inside a map
// value is an error from the whole export, not a panic.
func TestLineageMapValueError(t *testing.T) {
	type bag struct {
		mnemonica.Node
		Items map[string]*FixtureUser
	}
	collection := mnemonica.NewCollection(mnemonica.WithCollectionName("bag"))
	bagT, err := mnemonica.Define[bag](collection, "Bag", func(b *bag, _ string) error {
		return nil
	})
	if err != nil {
		t.Fatalf("Define: %v", err)
	}
	instance := lawMust(bagT.New("seed"))
	_, err = mnemonica.Lineage([]mnemonica.Instance{instance})
	if err != nil {
		t.Fatalf("empty bag: %v", err)
	}
	instance.Items = map[string]*FixtureUser{"bad": {}}
	if _, err := mnemonica.Lineage([]mnemonica.Instance{instance}); !errors.Is(err, mnemonica.ErrNotAnInstance) {
		t.Errorf("map value with an unconstructed instance error = %v, want ErrNotAnInstance", err)
	}
}

// TestLineageArgsError: args go through the same machinery — an
// unconstructed instance in the args is an export error.
func TestLineageArgsError(t *testing.T) {
	type box struct {
		mnemonica.Node
		Note string
	}
	collection := mnemonica.NewCollection(mnemonica.WithCollectionName("argsref"))
	boxT, err := mnemonica.Define[box](collection, "Box", func(b *box, held *FixtureUser) error {
		b.Note = held.Name // unconstructed field reads as its zero value
		return nil
	})
	if err != nil {
		t.Fatalf("Define: %v", err)
	}
	good := lawMust(boxT.New(lawFixtureUser(t)))
	if _, err := mnemonica.Lineage([]mnemonica.Instance{good}, mnemonica.WithArgs()); err != nil {
		t.Fatalf("Lineage with valid args: %v", err)
	}
	bad := lawMust(boxT.New(&FixtureUser{}))
	if _, err := mnemonica.Lineage([]mnemonica.Instance{bad}, mnemonica.WithArgs()); !errors.Is(err, mnemonica.ErrNotAnInstance) {
		t.Errorf("args with an unconstructed instance error = %v, want ErrNotAnInstance", err)
	}
}

// lawFixtureUser builds one constructed FixtureUser for args fixtures.
func lawFixtureUser(t *testing.T) *FixtureUser {
	t.Helper()
	collection := mnemonica.NewCollection()
	userT := lawMust(mnemonica.Define[FixtureUser](collection, "LawUser", func(u *FixtureUser, name string) error {
		u.Name = name
		return nil
	}))
	return lawMust(userT.New("ada"))
}

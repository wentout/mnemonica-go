package mnemonica_test

import (
	"errors"
	"testing"

	"github.com/wentout/mnemonica-go/mnemonica"
)

// The last encoder branches: interface shapes, collection cycles through
// any-typed fields, and error propagation out of slice/array elements.

func TestLineageNilInterfaceField(t *testing.T) {
	type holder struct {
		mnemonica.Node
		Held any
	}
	collection := mnemonica.NewCollection(mnemonica.WithCollectionName("niliface"))
	holderT, err := mnemonica.Define[holder](collection, "Holder", func(h *holder, _ string) error {
		return nil
	})
	if err != nil {
		t.Fatalf("Define: %v", err)
	}
	instance := lawMust(holderT.New("seed"))
	graph, err := mnemonica.Lineage([]mnemonica.Instance{instance})
	if err != nil {
		t.Fatalf("Lineage: %v", err)
	}
	if own := graph.Nodes[mnemonica.ID(instance)].Own; own["Held"] != nil {
		t.Errorf("nil interface field = %v, want null", own["Held"])
	}
}

func TestLineagePlainInterfaceField(t *testing.T) {
	type holder struct {
		mnemonica.Node
		Held any
	}
	collection := mnemonica.NewCollection(mnemonica.WithCollectionName("plainiface"))
	holderT, err := mnemonica.Define[holder](collection, "Holder", func(h *holder, _ string) error {
		h.Held = "plain"
		return nil
	})
	if err != nil {
		t.Fatalf("Define: %v", err)
	}
	instance := lawMust(holderT.New("seed"))
	graph, err := mnemonica.Lineage([]mnemonica.Instance{instance})
	if err != nil {
		t.Fatalf("Lineage: %v", err)
	}
	if own := graph.Nodes[mnemonica.ID(instance)].Own; own["Held"] != "plain" {
		t.Errorf("interface field = %v, want plain", own["Held"])
	}
}

// TestLineageSliceCycle: a slice that contains itself (through an any
// element) exports the inner occurrence as a cycle placeholder.
func TestLineageSliceCycle(t *testing.T) {
	type holder struct {
		mnemonica.Node
		Items []any
	}
	collection := mnemonica.NewCollection(mnemonica.WithCollectionName("slicecycle"))
	holderT, err := mnemonica.Define[holder](collection, "Holder", func(h *holder, _ string) error {
		h.Items = make([]any, 1)
		h.Items[0] = h.Items
		return nil
	})
	if err != nil {
		t.Fatalf("Define: %v", err)
	}
	instance := lawMust(holderT.New("seed"))
	graph, err := mnemonica.Lineage([]mnemonica.Instance{instance})
	if err != nil {
		t.Fatalf("Lineage: %v", err)
	}
	items, ok := graph.Nodes[mnemonica.ID(instance)].Own["Items"].([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("items = %v", graph.Nodes[mnemonica.ID(instance)].Own["Items"])
	}
	tag, ok := items[0].(map[string]any)
	if !ok || tag["kind"] != "cycle" {
		t.Errorf("self-containing slice = %v, want a cycle placeholder", items[0])
	}
}

// TestLineageMapCycle: the same for maps.
func TestLineageMapCycle(t *testing.T) {
	type holder struct {
		mnemonica.Node
		Lookup map[string]any
	}
	collection := mnemonica.NewCollection(mnemonica.WithCollectionName("mapcycle"))
	holderT, err := mnemonica.Define[holder](collection, "Holder", func(h *holder, _ string) error {
		h.Lookup = map[string]any{}
		h.Lookup["self"] = h.Lookup
		return nil
	})
	if err != nil {
		t.Fatalf("Define: %v", err)
	}
	instance := lawMust(holderT.New("seed"))
	graph, err := mnemonica.Lineage([]mnemonica.Instance{instance})
	if err != nil {
		t.Fatalf("Lineage: %v", err)
	}
	lookup, ok := graph.Nodes[mnemonica.ID(instance)].Own["Lookup"].(map[string]any)
	if !ok {
		t.Fatalf("lookup = %v", graph.Nodes[mnemonica.ID(instance)].Own["Lookup"])
	}
	tag, ok := lookup["self"].(map[string]any)
	if !ok || tag["kind"] != "cycle" {
		t.Errorf("self-containing map = %v, want a cycle placeholder", lookup["self"])
	}
}

// TestLineageSliceElementError: an unconstructed instance inside a slice
// propagates the error out of the element loop.
func TestLineageSliceElementError(t *testing.T) {
	type holder struct {
		mnemonica.Node
		Users []*FixtureUser
	}
	collection := mnemonica.NewCollection(mnemonica.WithCollectionName("sliceerr"))
	holderT, err := mnemonica.Define[holder](collection, "Holder", func(h *holder, _ string) error {
		return nil
	})
	if err != nil {
		t.Fatalf("Define: %v", err)
	}
	instance := lawMust(holderT.New("seed"))
	instance.Users = []*FixtureUser{{}}
	if _, err := mnemonica.Lineage([]mnemonica.Instance{instance}); !errors.Is(err, mnemonica.ErrNotAnInstance) {
		t.Errorf("slice element with an unconstructed instance error = %v, want ErrNotAnInstance", err)
	}
}

// TestLineageArrayElementError: and out of an array element.
func TestLineageArrayElementError(t *testing.T) {
	type holder struct {
		mnemonica.Node
		Users [1]*FixtureUser
	}
	collection := mnemonica.NewCollection(mnemonica.WithCollectionName("arrayerr"))
	holderT, err := mnemonica.Define[holder](collection, "Holder", func(h *holder, _ string) error {
		return nil
	})
	if err != nil {
		t.Fatalf("Define: %v", err)
	}
	instance := lawMust(holderT.New("seed"))
	instance.Users[0] = &FixtureUser{}
	if _, err := mnemonica.Lineage([]mnemonica.Instance{instance}); !errors.Is(err, mnemonica.ErrNotAnInstance) {
		t.Errorf("array element with an unconstructed instance error = %v, want ErrNotAnInstance", err)
	}
}

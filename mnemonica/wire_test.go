package mnemonica_test

import (
	"testing"

	"github.com/wentout/mnemonica-go/mnemonica"
)

// TestSubWithWireFunc: an explicit typed wire func replaces the cached
// reflect setter — the P0 probe's option 1b, opened for hand-written and
// generated wiring.
func TestSubWithWireFunc(t *testing.T) {
	fx := newFixture()
	wiredT, err := mnemonica.Sub[Admin](fx.userT, "WiredAdmin",
		func(a *Admin, role string) error {
			a.Role = role
			return nil
		},
		mnemonica.WithWireFunc(func(a *Admin, u *User) {
			a.User = u
		}),
	)
	if err != nil {
		t.Fatalf("Sub with WithWireFunc: %v", err)
	}
	user, err := fx.userT.New("ada")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	admin, err := wiredT.From(user, "root")
	if err != nil {
		t.Fatalf("From: %v", err)
	}
	if admin.User != user {
		t.Errorf("embedded *User = %p, want the parent %p (wire func wired it)", admin.User, user)
	}
	if admin.Name != "ada" {
		t.Errorf("promoted Name = %q, want %q (read-through intact)", admin.Name, "ada")
	}
}

// TestSubWithWireFuncTypeMismatch: a wire func for the wrong types is a
// define-time error, not a per-construction panic.
func TestSubWithWireFuncTypeMismatch(t *testing.T) {
	fx := newFixture()
	_, err := mnemonica.Sub[Admin](fx.userT, "Mismatched",
		func(a *Admin, role string) error { return nil },
		mnemonica.WithWireFunc(func(a *Admin, w *Widget) {
			t.Error("wire func must not run: its types do not match the Sub")
		}),
	)
	if err == nil {
		t.Fatal("Sub with mismatched wire func succeeded, want ErrWrongTypeDefinition")
	}
}

// TestSubWithWireFuncSkipsEmbedValidation: with an explicit wire func the
// shape is the func's own assertion — Sub does not require the embedded
// parent pointer (the Node header still records the real parent).
func TestSubWithWireFuncSkipsEmbedValidation(t *testing.T) {
	fx := newFixture()
	wiredT, err := mnemonica.Sub[Orphan](fx.userT, "Detached",
		func(o *Orphan, label string) error {
			return nil
		},
		mnemonica.WithWireFunc(func(o *Orphan, u *User) {
			// Intentionally no embedded pointer: the parent lives only in
			// the Node header.
		}),
	)
	if err != nil {
		t.Fatalf("Sub with wire func over a parentless struct: %v", err)
	}
	user, err := fx.userT.New("ada")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	detached, err := wiredT.From(user, "lonely")
	if err != nil {
		t.Fatalf("From: %v", err)
	}
	parent, ok := mnemonica.Parent(detached)
	if !ok || parent != user {
		t.Errorf("Parent = %v, %v; want the user via the Node header", parent, ok)
	}
}

// TestAttachWire: the generated-code hook — a typed wire func attached
// after Sub replaces the reflect setter before constructions begin.
func TestAttachWire(t *testing.T) {
	fx := newFixture()
	taggedT, err := mnemonica.Sub[Tagged](fx.userT, "Tagged",
		func(x *Tagged, label string) error {
			x.Label = label
			return nil
		},
	)
	if err != nil {
		t.Fatalf("Sub: %v", err)
	}
	wireRuns := 0
	taggedT.AttachWire(func(x *Tagged, u *User) {
		wireRuns++
		x.User = u
	})
	user, err := fx.userT.New("ada")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	tagged, err := taggedT.From(user, "hello")
	if err != nil {
		t.Fatalf("From: %v", err)
	}
	if wireRuns != 1 {
		t.Errorf("attached wire ran %d times, want 1", wireRuns)
	}
	if tagged.User != user {
		t.Errorf("attached wire did not wire the parent: %p", tagged.User)
	}
	if tagged.Name != "ada" {
		t.Errorf("promoted Name = %q, want %q", tagged.Name, "ada")
	}
}

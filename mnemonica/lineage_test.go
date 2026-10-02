package mnemonica_test

import (
	"testing"

	"github.com/wentout/mnemonica-go/mnemonica"
)

func TestIsWalksLineage(t *testing.T) {
	fx := newFixture()
	user, err := fx.userT.New("ada")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	admin, err := fx.adminT.From(user, "root")
	if err != nil {
		t.Fatalf("From: %v", err)
	}
	super, err := fx.superT.From(admin, 9)
	if err != nil {
		t.Fatalf("From super: %v", err)
	}

	// C2.5: an Admin built from a User IS a User along its chain.
	if !mnemonica.Is[*User](admin) {
		t.Error("Is[*User](admin) = false, want true")
	}
	if !mnemonica.Is[*Admin](admin) {
		t.Error("Is[*Admin](admin) = false, want true")
	}
	if !mnemonica.Is[*User](super) {
		t.Error("Is[*User](super) = false, want true (lineage walks to root)")
	}
	if !mnemonica.Is[*Admin](super) {
		t.Error("Is[*Admin](super) = false, want true")
	}
	if mnemonica.Is[*SuperAdmin](admin) {
		t.Error("Is[*SuperAdmin](admin) = true, want false (children are not ancestors)")
	}
	if mnemonica.Is[*Admin](user) {
		t.Error("Is[*Admin](user) = true, want false")
	}
	if mnemonica.Is[*Widget](admin) {
		t.Error("Is[*Widget](admin) = true, want false (unrelated type)")
	}
}

func TestIsNilInstance(t *testing.T) {
	if mnemonica.Is[*User](nil) {
		t.Error("Is[*User](nil) = true, want false")
	}
}

func TestAsWalksLineage(t *testing.T) {
	fx := newFixture()
	user, err := fx.userT.New("ada")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	admin, err := fx.adminT.From(user, "root")
	if err != nil {
		t.Fatalf("From: %v", err)
	}
	super, err := fx.superT.From(admin, 9)
	if err != nil {
		t.Fatalf("From super: %v", err)
	}

	// errors.As-style: x itself matches first.
	gotAdmin, ok := mnemonica.As[*Admin](admin)
	if !ok || gotAdmin != admin {
		t.Errorf("As[*Admin](admin) = %v, %v; want the admin itself", gotAdmin, ok)
	}
	// Then the ancestor: the exact parent instance comes back.
	gotUser, ok := mnemonica.As[*User](admin)
	if !ok || gotUser != user {
		t.Errorf("As[*User](admin) = %v, %v; want the parent user", gotUser, ok)
	}
	// Two levels up through a three-node chain.
	gotUser, ok = mnemonica.As[*User](super)
	if !ok || gotUser != user {
		t.Errorf("As[*User](super) = %v, %v; want the root user", gotUser, ok)
	}
	// No match: zero value and false, no panic.
	missing, ok := mnemonica.As[*Widget](admin)
	if ok || missing != nil {
		t.Errorf("As[*Widget](admin) = %v, %v; want nil, false", missing, ok)
	}
	_, ok = mnemonica.As[*User](nil)
	if ok {
		t.Error("As[*User](nil) matched, want false")
	}
}

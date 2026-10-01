package mnemonica_test

import (
	"errors"
	"reflect"
	"testing"

	"mnemonica/mnemonica"
)

func TestNewConstructsRoot(t *testing.T) {
	fx := newFixture()
	user, err := fx.userT.New("ada")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if user.Name != "ada" {
		t.Errorf("Name = %q, want %q", user.Name, "ada")
	}
}

func TestNewOnSubtypeRejected(t *testing.T) {
	fx := newFixture()
	_, err := fx.adminT.New("root")
	if !errors.Is(err, mnemonica.ErrWrongModificationPattern) {
		t.Fatalf("New on a subtype error = %v, want ErrWrongModificationPattern", err)
	}
}

func TestNewHandlerErrorBecomesErroredInstance(t *testing.T) {
	collection := mnemonica.NewCollection()
	boom := errors.New("boom")
	failing := mnemonica.Must(mnemonica.Define[User](collection, "User", func(u *User, name string) error {
		return boom
	}))
	_, err := failing.New("ada")
	var errored *mnemonica.ErroredInstance
	if !errors.As(err, &errored) {
		t.Fatalf("New error = %v, want an *ErroredInstance", err)
	}
	if !errors.Is(err, boom) {
		t.Errorf("New error does not unwrap to the handler error: %v", err)
	}
	if errored.Type.Path() != "User" {
		t.Errorf("ErroredInstance.Type = %q, want %q", errored.Type.Path(), "User")
	}
	if errored.Instance == nil {
		t.Error("ErroredInstance carries no half-built instance")
	}
}

func TestFromConstructsFromParent(t *testing.T) {
	fx := newFixture()
	user, err := fx.userT.New("ada")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	admin, err := fx.adminT.From(user, "root")
	if err != nil {
		t.Fatalf("From: %v", err)
	}
	if admin.Role != "root" {
		t.Errorf("Role = %q, want %q", admin.Role, "root")
	}
	// The embedded pointer must be the very parent instance, aliased.
	if admin.User != user {
		t.Errorf("embedded *User = %p, want the parent %p", admin.User, user)
	}
}

func TestReadThroughPromotesFromParent(t *testing.T) {
	fx := newFixture()
	user, err := fx.userT.New("ada")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	admin, err := fx.adminT.From(user, "root")
	if err != nil {
		t.Fatalf("From: %v", err)
	}
	if admin.Name != "ada" {
		t.Errorf("promoted Name = %q, want %q (C2.3 read-through)", admin.Name, "ada")
	}
	// C2.6: the parent is shared, not copied — later mutation is visible.
	user.Name = "grace"
	if admin.Name != "grace" {
		t.Errorf("after parent mutation, promoted Name = %q, want %q (shared parent)", admin.Name, "grace")
	}
}

func TestWriteThroughIsShared(t *testing.T) {
	// C2.4 adapted: Go cannot shadow promoted writes. This test pins the
	// adapted semantics — the write lands on the shared parent — so the
	// mnemonicatest helper and the P5 mnemonicavet analyzer have a fixture.
	fx := newFixture()
	user, err := fx.userT.New("ada")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	admin, err := fx.adminT.From(user, "root")
	if err != nil {
		t.Fatalf("From: %v", err)
	}
	// The write goes through reflect so the port's own sources stay
	// mnemonicavet-clean: the test pins the SEMANTICS (the promoted write
	// reaches the shared parent), not the source pattern the analyzer
	// guards.
	nameField := reflect.ValueOf(admin).Elem().FieldByName("Name")
	nameField.SetString("mallory")
	if user.Name != "mallory" {
		t.Errorf("user.Name = %q, want %q (promoted write hits the shared parent)", user.Name, "mallory")
	}
}

func TestFromWrongParentType(t *testing.T) {
	fx := newFixture()
	widgetT := newWidget()
	widget, err := widgetT.New("sku-1")
	if err != nil {
		t.Fatalf("New widget: %v", err)
	}
	// The untyped bridge is how a wrong-typed parent reaches From at all.
	adminType, ok := mnemonica.Lookup(fx.col, "User.Admin")
	if !ok {
		t.Fatal("Lookup(User.Admin) not found")
	}
	if _, err := adminType.From(widget, "root"); !errors.Is(err, mnemonica.ErrWrongModificationPattern) {
		t.Fatalf("From(widget) error = %v, want ErrWrongModificationPattern", err)
	}
}

func TestFromRequiresDirectParent(t *testing.T) {
	fx := newFixture()
	user, err := fx.userT.New("ada")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	// SuperAdmin's declared parent is Admin; a User instance is not an
	// Admin (typed From cannot even express this; the untyped bridge can).
	superType, ok := mnemonica.Lookup(fx.col, "User.Admin.SuperAdmin")
	if !ok {
		t.Fatal("Lookup(User.Admin.SuperAdmin) not found")
	}
	if _, err := superType.From(user, 9); !errors.Is(err, mnemonica.ErrWrongModificationPattern) {
		t.Fatalf("From(user) into SuperAdmin error = %v, want ErrWrongModificationPattern", err)
	}
}

func TestFromNilParent(t *testing.T) {
	fx := newFixture()
	_, err := fx.adminT.From(nil, "root")
	if !errors.Is(err, mnemonica.ErrWrongModificationPattern) {
		t.Fatalf("From(nil) error = %v, want ErrWrongModificationPattern", err)
	}
}

func TestRootFromRejected(t *testing.T) {
	fx := newFixture()
	user, err := fx.userT.New("ada")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	// A root handle's From takes the Root sentinel; no *Root can be an
	// Instance, so the call type-checks but always fails at runtime.
	_, err = fx.userT.From(&mnemonica.Root{}, "ada")
	if !errors.Is(err, mnemonica.ErrWrongModificationPattern) {
		t.Fatalf("root typed From error = %v, want ErrWrongModificationPattern", err)
	}
	// Through the untyped bridge the parent IS an instance, and the
	// root-type guard is what rejects it.
	rootType, ok := mnemonica.Lookup(fx.col, "User")
	if !ok {
		t.Fatal("Lookup(User) not found")
	}
	if _, err := rootType.From(user, "ada"); !errors.Is(err, mnemonica.ErrWrongModificationPattern) {
		t.Fatalf("root untyped From error = %v, want ErrWrongModificationPattern", err)
	}
}

func TestFromUnconstructedParentRejected(t *testing.T) {
	fx := newFixture()
	// A hand-made *User embeds Node (so it is an Instance) but never went
	// through construction: it has no type identity to parent with.
	if _, err := fx.adminT.From(&User{}, "root"); !errors.Is(err, mnemonica.ErrWrongModificationPattern) {
		t.Fatalf("From(&User{}) error = %v, want ErrWrongModificationPattern", err)
	}
}

func TestFromHandlerErrorBecomesErroredInstance(t *testing.T) {
	collection := mnemonica.NewCollection()
	userT := mnemonica.Must(mnemonica.Define[User](collection, "User", func(u *User, name string) error {
		u.Name = name
		return nil
	}))
	boom := errors.New("sub boom")
	failing := mnemonica.Must(mnemonica.Sub[Admin](userT, "Admin", func(a *Admin, role string) error {
		return boom
	}))
	user, err := userT.New("ada")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	_, err = failing.From(user, "root")
	var errored *mnemonica.ErroredInstance
	if !errors.As(err, &errored) {
		t.Fatalf("From error = %v, want an *ErroredInstance", err)
	}
	if !errors.Is(err, boom) {
		t.Errorf("From error does not unwrap to the handler error: %v", err)
	}
	if errored.Instance == nil {
		t.Error("ErroredInstance carries no half-built instance")
	}
}

func TestSiblingsShareParent(t *testing.T) {
	fx := newFixture()
	user, err := fx.userT.New("ada")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	first, err := fx.adminT.From(user, "root")
	if err != nil {
		t.Fatalf("first From: %v", err)
	}
	second, err := fx.adminT.From(user, "operator")
	if err != nil {
		t.Fatalf("second From: %v", err)
	}
	// One parent, many children (C2.6): each child sees the same parent.
	if first.User != second.User {
		t.Error("siblings do not share the same parent instance")
	}
	if first.Role == second.Role {
		t.Errorf("sibling Roles collide: %q", first.Role)
	}
	user.Name = "grace"
	if first.Name != "grace" || second.Name != "grace" {
		t.Error("siblings do not both observe parent mutation")
	}
}

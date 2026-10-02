package mnemonica_test

import (
	"errors"
	"testing"

	"github.com/wentout/mnemonica-go/mnemonica"
)

func TestLookupFunctionAndMethod(t *testing.T) {
	fx := newFixture()
	for name, lookup := range map[string]func(string) (mnemonica.Type, bool){
		"function": func(path string) (mnemonica.Type, bool) {
			return mnemonica.Lookup(fx.col, path)
		},
		"method": fx.col.Lookup,
	} {
		root, ok := lookup("User")
		if !ok {
			t.Fatalf("%s: Lookup(User) not found", name)
		}
		if root.Name() != "User" || root.Path() != "User" {
			t.Errorf("%s: Lookup(User) = (%q, %q), want (User, User)", name, root.Name(), root.Path())
		}
		nested, ok := lookup("User.Admin")
		if !ok {
			t.Fatalf("%s: Lookup(User.Admin) not found", name)
		}
		if nested.Name() != "Admin" || nested.Path() != "User.Admin" {
			t.Errorf("%s: Lookup(User.Admin) = (%q, %q), want (Admin, User.Admin)", name, nested.Name(), nested.Path())
		}
		deep, ok := lookup("User.Admin.SuperAdmin")
		if !ok {
			t.Fatalf("%s: Lookup(User.Admin.SuperAdmin) not found", name)
		}
		if deep.Path() != "User.Admin.SuperAdmin" {
			t.Errorf("%s: deep Path() = %q, want %q", name, deep.Path(), "User.Admin.SuperAdmin")
		}
	}
}

func TestLookupUnknownPaths(t *testing.T) {
	fx := newFixture()
	for _, path := range []string{
		"Nope",
		"User.Nope",
		"User.Admin.Nope",
		"",
		"User.",
		"User..Admin",
	} {
		if typ, ok := fx.col.Lookup(path); ok {
			t.Errorf("Lookup(%q) = %v, true; want ok=false (no panic)", path, typ.Path())
		}
	}
}

func TestLookupEmptyCollection(t *testing.T) {
	collection := mnemonica.NewCollection()
	if _, ok := collection.Lookup("User"); ok {
		t.Error("Lookup in an empty collection found a type")
	}
}

func TestLookupUntypedConstruction(t *testing.T) {
	// The untyped Type returned by Lookup still constructs (C1.4): New for
	// roots, From — with the same parent check — for subtypes.
	fx := newFixture()
	user, err := fx.userT.New("ada")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	rootType, ok := mnemonica.Lookup(fx.col, "User")
	if !ok {
		t.Fatal("Lookup(User) not found")
	}
	constructed, err := rootType.New("grace")
	if err != nil {
		t.Fatalf("untyped New: %v", err)
	}
	record, err := mnemonica.Props(constructed)
	if err != nil {
		t.Fatalf("Props of untyped construction: %v", err)
	}
	if record.Type.Path() != "User" {
		t.Errorf("untyped New produced type %q, want %q", record.Type.Path(), "User")
	}

	subType, ok := mnemonica.Lookup(fx.col, "User.Admin")
	if !ok {
		t.Fatal("Lookup(User.Admin) not found")
	}
	fromTyped, err := subType.From(user, "operator")
	if err != nil {
		t.Fatalf("untyped From: %v", err)
	}
	if fromTyped.(*Admin).Role != "operator" {
		t.Errorf("untyped From Role = %q, want %q", fromTyped.(*Admin).Role, "operator")
	}
	// Construction requires the DIRECT declared parent type: admin is a
	// User by lineage (Is[*User](admin) is true), but it is not an
	// instance OF User, so From rejects it — strictChain's default shape.
	admin, err := fx.adminT.From(user, "root")
	if err != nil {
		t.Fatalf("From: %v", err)
	}
	if _, err := subType.From(admin, "x"); !errors.Is(err, mnemonica.ErrWrongModificationPattern) {
		t.Errorf("untyped From(admin) error = %v, want ErrWrongModificationPattern", err)
	}
	if _, err := subType.From(constructed, "y"); err != nil {
		t.Errorf("untyped From with valid parent: %v", err)
	}
}

func TestLookupUntypedWrongArgs(t *testing.T) {
	// The untyped bridge is the one place a wrong args type can reach a
	// typed handler: it must fail with the WRONG_ARGUMENTS_USED sentinel,
	// not panic.
	fx := newFixture()
	rootType, ok := mnemonica.Lookup(fx.col, "User")
	if !ok {
		t.Fatal("Lookup(User) not found")
	}
	if _, err := rootType.New(42); !errors.Is(err, mnemonica.ErrWrongArgumentsUsed) {
		t.Fatalf("untyped New with wrong args error = %v, want ErrWrongArgumentsUsed", err)
	}
	subType, ok := mnemonica.Lookup(fx.col, "User.Admin")
	if !ok {
		t.Fatal("Lookup(User.Admin) not found")
	}
	user, err := fx.userT.New("ada")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if _, err := subType.From(user, 42); !errors.Is(err, mnemonica.ErrWrongArgumentsUsed) {
		t.Fatalf("untyped From with wrong args error = %v, want ErrWrongArgumentsUsed", err)
	}
}

func TestLookupTypeSubtypes(t *testing.T) {
	fx := newFixture()
	// A second subtype exercises the sort comparator: Subtypes come back
	// sorted by name regardless of declaration order.
	_, err := mnemonica.Sub[Tagged](fx.userT, "Tagged", func(x *Tagged, label string) error { return nil })
	if err != nil {
		t.Fatalf("Sub[Tagged]: %v", err)
	}
	rootType, ok := fx.col.Lookup("User")
	if !ok {
		t.Fatal("Lookup(User) not found")
	}
	subs := rootType.Subtypes()
	if len(subs) != 2 || subs[0].Name() != "Admin" || subs[1].Name() != "Tagged" {
		t.Fatalf("Subtypes() = %v, want [Admin Tagged] sorted", subs)
	}
	nested, ok := fx.col.Lookup("User.Admin")
	if !ok {
		t.Fatal("Lookup(User.Admin) not found")
	}
	if len(nested.Subtypes()) != 1 || nested.Subtypes()[0].Name() != "SuperAdmin" {
		t.Fatalf("Admin Subtypes() = %v, want [SuperAdmin]", nested.Subtypes())
	}
}

func TestLookupUntypedConstructionErrors(t *testing.T) {
	collection := mnemonica.NewCollection()
	boom := errors.New("untyped boom")
	okRoot := mnemonica.Must(mnemonica.Define[User](collection, "User", func(u *User, name string) error {
		u.Name = name
		return nil
	}))
	_, err := mnemonica.Define[User](collection, "Failing", func(u *User, name string) error { return boom })
	if err != nil {
		t.Fatalf("Define Failing: %v", err)
	}
	failingSub := mnemonica.Must(mnemonica.Sub[Admin](okRoot, "FailingSub", func(a *Admin, role string) error {
		return boom
	}))
	_ = failingSub

	failRootType, ok := mnemonica.Lookup(collection, "Failing")
	if !ok {
		t.Fatal("Lookup(Failing) not found")
	}
	_, err = failRootType.New("x")
	var errored *mnemonica.ErroredInstance
	if !errors.As(err, &errored) {
		t.Fatalf("untyped New error = %v, want an *ErroredInstance", err)
	}

	subType, ok := mnemonica.Lookup(collection, "User.FailingSub")
	if !ok {
		t.Fatal("Lookup(User.FailingSub) not found")
	}
	user, err := okRoot.New("ada")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	_, err = subType.From(user, "root")
	if !errors.As(err, &errored) {
		t.Fatalf("untyped From error = %v, want an *ErroredInstance", err)
	}
}

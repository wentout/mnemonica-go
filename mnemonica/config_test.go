package mnemonica_test

import (
	"errors"
	"strings"
	"testing"

	"mnemonica/mnemonica"
)

// strictChain is exercised through the untyped bridge: the typed From takes
// the parent by its pointer type, so the chain walk-up it gates is only
// reachable with a dynamically-typed parent (C4.1, both JS checks).

// relaxedFixture: target relaxed, entity relaxed — walk-up must succeed.
type relaxedFixture struct {
	col    *mnemonica.Collection
	userT  *mnemonica.TypeDef[User, mnemonica.Root, string]
	adminT *mnemonica.TypeDef[Admin, User, string]
	superT *mnemonica.TypeDef[SuperAdmin, Admin, int]
}

func newRelaxedFixture() relaxedFixture {
	collection := mnemonica.NewCollection()
	userT := mnemonica.Must(mnemonica.Define[User](collection, "User", func(u *User, name string) error {
		u.Name = name
		return nil
	}))
	adminT := mnemonica.Must(mnemonica.Sub[Admin](userT, "Admin", func(a *Admin, role string) error {
		a.Role = role
		return nil
	}, mnemonica.WithStrictChain(false)))
	superT := mnemonica.Must(mnemonica.Sub[SuperAdmin](adminT, "SuperAdmin", func(s *SuperAdmin, level int) error {
		s.Level = level
		return nil
	}))
	return relaxedFixture{col: collection, userT: userT, adminT: adminT, superT: superT}
}

func TestStrictChainDefaultRejectsWalkUp(t *testing.T) {
	fx := newFixture()
	super, err := fx.superT.From(fxAdmin(t, fx), 9)
	if err != nil {
		t.Fatalf("From: %v", err)
	}
	adminType, ok := fx.col.Lookup("User.Admin")
	if !ok {
		t.Fatal("Lookup(User.Admin) not found")
	}
	// super's chain contains a User, but the target type's strictChain
	// (default true) forbids the walk-up.
	if _, err := adminType.From(super, "root"); !errors.Is(err, mnemonica.ErrWrongModificationPattern) {
		t.Fatalf("walk-up with strictChain error = %v, want ErrWrongModificationPattern", err)
	}
}

func TestStrictChainFalseAllowsWalkUp(t *testing.T) {
	fx := newRelaxedFixture()
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
	adminType, ok := fx.col.Lookup("User.Admin")
	if !ok {
		t.Fatal("Lookup(User.Admin) not found")
	}
	constructed, err := adminType.From(super, "operator")
	if err != nil {
		t.Fatalf("walk-up From: %v", err)
	}
	// The WIRED parent is the nearest ancestor matching the declared parent
	// type — the User found up super's chain — not super itself.
	record, err := mnemonica.Props(constructed)
	if err != nil {
		t.Fatalf("Props: %v", err)
	}
	if record.Parent != user {
		t.Errorf("wired parent = %v, want the User ancestor %p", record.Parent, user)
	}
}

// fxAdmin builds an Admin for tests that need one without naming the handle.
func fxAdmin(t *testing.T, fx fixture) *Admin {
	t.Helper()
	user, err := fx.userT.New("ada")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	admin, err := fx.adminT.From(user, "root")
	if err != nil {
		t.Fatalf("From: %v", err)
	}
	return admin
}

func TestStrictChainParentTypeGate(t *testing.T) {
	// Target relaxed, ENTITY strict: the parent's own type config forbids
	// the walk-up — the second independent JS check.
	collection := mnemonica.NewCollection()
	userT := mnemonica.Must(mnemonica.Define[User](collection, "User", func(u *User, name string) error {
		u.Name = name
		return nil
	}))
	adminT := mnemonica.Must(mnemonica.Sub[Admin](userT, "Admin", func(a *Admin, role string) error {
		a.Role = role
		return nil
	}, mnemonica.WithStrictChain(false)))
	superT := mnemonica.Must(mnemonica.Sub[SuperAdmin](adminT, "SuperAdmin", func(s *SuperAdmin, level int) error {
		s.Level = level
		return nil
	}, mnemonica.WithStrictChain(true)))
	user, err := userT.New("ada")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	admin, err := adminT.From(user, "root")
	if err != nil {
		t.Fatalf("From: %v", err)
	}
	super, err := superT.From(admin, 9)
	if err != nil {
		t.Fatalf("From super: %v", err)
	}
	adminType, ok := collection.Lookup("User.Admin")
	if !ok {
		t.Fatal("Lookup(User.Admin) not found")
	}
	if _, err := adminType.From(super, "operator"); !errors.Is(err, mnemonica.ErrWrongModificationPattern) {
		t.Fatalf("walk-up with strict entity error = %v, want ErrWrongModificationPattern", err)
	}
}

func TestStrictChainTargetGate(t *testing.T) {
	// Entity relaxed, TARGET strict: the target type's own config forbids
	// the walk-up even though the parent's type allows it.
	collection := mnemonica.NewCollection()
	userT := mnemonica.Must(mnemonica.Define[User](collection, "User", func(u *User, name string) error {
		u.Name = name
		return nil
	}, mnemonica.WithStrictChain(false)))
	adminT := mnemonica.Must(mnemonica.Sub[Admin](userT, "Admin", func(a *Admin, role string) error {
		a.Role = role
		return nil
	}, mnemonica.WithStrictChain(true)))
	superT := mnemonica.Must(mnemonica.Sub[SuperAdmin](adminT, "SuperAdmin", func(s *SuperAdmin, level int) error {
		s.Level = level
		return nil
	}, mnemonica.WithStrictChain(false)))
	user, err := userT.New("ada")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	admin, err := adminT.From(user, "root")
	if err != nil {
		t.Fatalf("From: %v", err)
	}
	super, err := superT.From(admin, 9)
	if err != nil {
		t.Fatalf("From super: %v", err)
	}
	adminType, ok := collection.Lookup("User.Admin")
	if !ok {
		t.Fatal("Lookup(User.Admin) not found")
	}
	if _, err := adminType.From(super, "operator"); !errors.Is(err, mnemonica.ErrWrongModificationPattern) {
		t.Fatalf("walk-up with strict target error = %v, want ErrWrongModificationPattern", err)
	}
}

func TestStrictChainWalkUpNotFound(t *testing.T) {
	// Fully relaxed collection, but the parent's chain holds no instance
	// of the declared parent type: a foreign kind cannot be wired into the
	// static *P field, so the walk must fail.
	collection := mnemonica.NewCollection(mnemonica.WithStrictChain(false))
	userT := mnemonica.Must(mnemonica.Define[User](collection, "User", func(u *User, name string) error {
		u.Name = name
		return nil
	}))
	_, err := mnemonica.Sub[Admin](userT, "Admin", func(a *Admin, role string) error {
		a.Role = role
		return nil
	})
	if err != nil {
		t.Fatalf("Sub: %v", err)
	}
	widgetT := mnemonica.Must(mnemonica.Define[Widget](collection, "Widget", func(w *Widget, sku string) error {
		w.SKU = sku
		return nil
	}))
	widget, err := widgetT.New("sku-1")
	if err != nil {
		t.Fatalf("New widget: %v", err)
	}
	adminType, ok := collection.Lookup("User.Admin")
	if !ok {
		t.Fatal("Lookup(User.Admin) not found")
	}
	if _, err := adminType.From(widget, "root"); !errors.Is(err, mnemonica.ErrWrongModificationPattern) {
		t.Fatalf("walk-up of a foreign kind error = %v, want ErrWrongModificationPattern", err)
	}
}

func TestBlockErrorsDefaultBlocks(t *testing.T) {
	collection := mnemonica.NewCollection()
	boom := errors.New("boom")
	userT := mnemonica.Must(mnemonica.Define[User](collection, "User", func(u *User, name string) error {
		return boom
	}))
	adminT := mnemonica.Must(mnemonica.Sub[Admin](userT, "Admin", func(a *Admin, role string) error {
		a.Role = role
		return nil
	}))
	_, err := userT.New("ada")
	var errored *mnemonica.ErroredInstance
	if !errors.As(err, &errored) {
		t.Fatalf("New error = %v, want an *ErroredInstance", err)
	}
	// Constructing FROM the errored instance is refused on both the typed
	// and the untyped paths.
	if _, err := adminT.From(errored.Instance.(*User), "root"); !errors.Is(err, mnemonica.ErrBlocked) {
		t.Fatalf("typed From(errored) error = %v, want ErrBlocked", err)
	}
	adminType, ok := collection.Lookup("User.Admin")
	if !ok {
		t.Fatal("Lookup(User.Admin) not found")
	}
	if _, err := adminType.From(errored.Instance, "root"); !errors.Is(err, mnemonica.ErrBlocked) {
		t.Fatalf("untyped From(errored) error = %v, want ErrBlocked", err)
	}
}

func TestBlockErrorsFalsePropagatesAndAllows(t *testing.T) {
	collection := mnemonica.NewCollection()
	boom := errors.New("boom")
	failing := mnemonica.Must(mnemonica.Define[User](collection, "User", func(u *User, name string) error {
		return boom
	}, mnemonica.WithBlockErrors(false)))
	_, err := failing.New("ada")
	var errored *mnemonica.ErroredInstance
	if errors.As(err, &errored) {
		t.Fatalf("blockErrors=false must not wrap: got %v", err)
	}
	if !errors.Is(err, boom) {
		t.Errorf("error = %v, want the raw handler error", err)
	}

	// And with the check off, an errored instance can parent new
	// constructions (the JS "propagates as-is" relaxation). Source the
	// errored instance from a blockErrors=true type, then construct the
	// relaxed Admin FROM it.
	source := mnemonica.Must(mnemonica.Define[User](collection, "Source", func(u *User, name string) error {
		return boom
	}))
	relaxedAdmin := mnemonica.Must(mnemonica.Sub[Admin](source, "Admin", func(a *Admin, role string) error {
		a.Role = role
		return nil
	}, mnemonica.WithBlockErrors(false)))
	_, err = source.New("ada")
	var erroredInstance *mnemonica.ErroredInstance
	if !errors.As(err, &erroredInstance) {
		t.Fatalf("setup: New error = %v, want an *ErroredInstance", err)
	}
	admin, err := relaxedAdmin.From(erroredInstance.Instance.(*User), "root")
	if err != nil {
		t.Fatalf("From(errored) with blockErrors=false: %v", err)
	}
	if admin.Role != "root" {
		t.Errorf("Role = %q, want %q", admin.Role, "root")
	}
}

func TestBlockErrorsChecksWholeLineage(t *testing.T) {
	// Mixed configs: the parent was built over an errored instance with
	// blockErrors off; a stricter grandchild type must still see the error
	// up the chain ("errors exist in the prototype chain").
	collection := mnemonica.NewCollection()
	boom := errors.New("boom")
	userT := mnemonica.Must(mnemonica.Define[User](collection, "User", func(u *User, name string) error {
		return boom
	}))
	adminT := mnemonica.Must(mnemonica.Sub[Admin](userT, "Admin", func(a *Admin, role string) error {
		a.Role = role
		return nil
	}, mnemonica.WithBlockErrors(false)))
	superT := mnemonica.Must(mnemonica.Sub[SuperAdmin](adminT, "SuperAdmin", func(s *SuperAdmin, level int) error {
		s.Level = level
		return nil
	}, mnemonica.WithBlockErrors(true)))
	_, err := userT.New("ada")
	var errored *mnemonica.ErroredInstance
	if !errors.As(err, &errored) {
		t.Fatalf("setup: New error = %v, want an *ErroredInstance", err)
	}
	admin, err := adminT.From(errored.Instance.(*User), "root")
	if err != nil {
		t.Fatalf("setup: relaxed From: %v", err)
	}
	if _, err := superT.From(admin, 9); !errors.Is(err, mnemonica.ErrBlocked) {
		t.Fatalf("strict grandchild From error = %v, want ErrBlocked", err)
	}
}

func TestSubmitStack(t *testing.T) {
	collection := mnemonica.NewCollection()
	userT := mnemonica.Must(mnemonica.Define[User](collection, "User", func(u *User, name string) error {
		u.Name = name
		return nil
	}, mnemonica.WithSubmitStack(true)))
	user, err := userT.New("ada")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	record, err := mnemonica.Props(user)
	if err != nil {
		t.Fatalf("Props: %v", err)
	}
	if !strings.Contains(record.Stack, "goroutine") {
		t.Errorf("Stack = %q, want a captured Go stack (submitStack)", record.Stack)
	}
}

func TestSubtypeConfigInheritance(t *testing.T) {
	// A subtype inherits the parent's RESOLVED config; an option overrides
	// per key. submitStack is the observable key here.
	collection := mnemonica.NewCollection()
	userT := mnemonica.Must(mnemonica.Define[User](collection, "User", func(u *User, name string) error {
		u.Name = name
		return nil
	}, mnemonica.WithSubmitStack(true)))
	adminT := mnemonica.Must(mnemonica.Sub[Admin](userT, "Admin", func(a *Admin, role string) error {
		a.Role = role
		return nil
	}))
	superT := mnemonica.Must(mnemonica.Sub[SuperAdmin](adminT, "SuperAdmin", func(s *SuperAdmin, level int) error {
		s.Level = level
		return nil
	}, mnemonica.WithSubmitStack(false)))
	user, err := userT.New("ada")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	admin, err := adminT.From(user, "root")
	if err != nil {
		t.Fatalf("From: %v", err)
	}
	super, err := superT.From(admin, 9)
	if err != nil {
		t.Fatalf("From super: %v", err)
	}
	adminRecord, err := mnemonica.Props(admin)
	if err != nil {
		t.Fatalf("Props(admin): %v", err)
	}
	if adminRecord.Stack == "" {
		t.Error("Admin should inherit submitStack=true from User")
	}
	superRecord, err := mnemonica.Props(super)
	if err != nil {
		t.Fatalf("Props(super): %v", err)
	}
	if superRecord.Stack != "" {
		t.Error("SuperAdmin overrides submitStack back to false")
	}
}

func TestCollectionConfigDefaults(t *testing.T) {
	// Collection-level defaults apply to roots defined without options.
	collection := mnemonica.NewCollection(mnemonica.WithSubmitStack(true), mnemonica.WithBlockErrors(false))
	healthy := mnemonica.Must(mnemonica.Define[User](collection, "User", func(u *User, name string) error {
		u.Name = name
		return nil
	}))
	boom := errors.New("boom")
	failing := mnemonica.Must(mnemonica.Define[Admin](collection, "Admin", func(a *Admin, role string) error {
		return boom
	}))
	user, err := healthy.New("ada")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	record, err := mnemonica.Props(user)
	if err != nil {
		t.Fatalf("Props: %v", err)
	}
	if record.Stack == "" {
		t.Error("collection default submitStack=true not applied")
	}
	_, err = failing.New("root")
	var errored *mnemonica.ErroredInstance
	if errors.As(err, &errored) {
		t.Fatalf("collection default blockErrors=false not applied: got %v", err)
	}
	if !errors.Is(err, boom) {
		t.Errorf("error = %v, want the raw handler error", err)
	}
}

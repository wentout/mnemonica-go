package mnemonica_test

import (
	"errors"
	"strings"
	"testing"

	"mnemonica/mnemonica"
)

func TestDefineDeclaresRootType(t *testing.T) {
	fx := newFixture()
	if fx.userT.Name() != "User" {
		t.Errorf("Name() = %q, want %q", fx.userT.Name(), "User")
	}
	if fx.userT.Path() != "User" {
		t.Errorf("Path() = %q, want %q", fx.userT.Path(), "User")
	}
}

func TestDefineDuplicateRoot(t *testing.T) {
	collection := mnemonica.NewCollection()
	_, err := mnemonica.Define[User](collection, "User", func(u *User, name string) error { return nil })
	if err != nil {
		t.Fatalf("first Define: %v", err)
	}
	_, err = mnemonica.Define[Admin](collection, "User", func(a *Admin, role string) error { return nil })
	if !errors.Is(err, mnemonica.ErrAlreadyDeclared) {
		t.Fatalf("second Define error = %v, want ErrAlreadyDeclared", err)
	}
}

func TestDefineEmptyName(t *testing.T) {
	collection := mnemonica.NewCollection()
	_, err := mnemonica.Define[User](collection, "", func(u *User, name string) error { return nil })
	if !errors.Is(err, mnemonica.ErrWrongTypeDefinition) {
		t.Fatalf("empty name error = %v, want ErrWrongTypeDefinition", err)
	}
}

func TestDefineDottedName(t *testing.T) {
	collection := mnemonica.NewCollection()
	_, err := mnemonica.Define[User](collection, "User.Admin", func(u *User, name string) error { return nil })
	if !errors.Is(err, mnemonica.ErrWrongTypeDefinition) {
		t.Fatalf("dotted name error = %v, want ErrWrongTypeDefinition", err)
	}
}

func TestDefineWithoutNode(t *testing.T) {
	collection := mnemonica.NewCollection()
	_, err := mnemonica.Define[noNode](collection, "NoNode", func(n *noNode, name string) error { return nil })
	if !errors.Is(err, mnemonica.ErrWrongTypeDefinition) {
		t.Fatalf("Define[noNode] error = %v, want ErrWrongTypeDefinition", err)
	}
	_, err = mnemonica.Define[int](collection, "Int", func(n *int, zero int) error { return nil })
	if !errors.Is(err, mnemonica.ErrWrongTypeDefinition) {
		t.Fatalf("Define[int] error = %v, want ErrWrongTypeDefinition", err)
	}
}

func TestSubDeclaresSubtypePath(t *testing.T) {
	fx := newFixture()
	if fx.adminT.Name() != "Admin" {
		t.Errorf("Name() = %q, want %q", fx.adminT.Name(), "Admin")
	}
	if fx.adminT.Path() != "User.Admin" {
		t.Errorf("Path() = %q, want %q", fx.adminT.Path(), "User.Admin")
	}
}

func TestSubDuplicateSibling(t *testing.T) {
	fx := newFixture()
	// Tagged embeds *User, so the field walk passes and the duplicate
	// declaration is what fails.
	_, err := mnemonica.Sub[Tagged](fx.userT, "Admin", func(x *Tagged, label string) error { return nil })
	if !errors.Is(err, mnemonica.ErrAlreadyDeclared) {
		t.Fatalf("duplicate Sub error = %v, want ErrAlreadyDeclared", err)
	}
}

func TestSubEmptyName(t *testing.T) {
	fx := newFixture()
	_, err := mnemonica.Sub[Admin](fx.userT, "", func(a *Admin, role string) error { return nil })
	if !errors.Is(err, mnemonica.ErrWrongTypeDefinition) {
		t.Fatalf("empty Sub name error = %v, want ErrWrongTypeDefinition", err)
	}
}

func TestSubDottedName(t *testing.T) {
	fx := newFixture()
	_, err := mnemonica.Sub[Admin](fx.userT, "A.B", func(a *Admin, role string) error { return nil })
	if !errors.Is(err, mnemonica.ErrWrongTypeDefinition) {
		t.Fatalf("dotted Sub name error = %v, want ErrWrongTypeDefinition", err)
	}
}

func TestSubWithoutNode(t *testing.T) {
	fx := newFixture()
	_, err := mnemonica.Sub[noNode](fx.userT, "NoNode", func(n *noNode, name string) error { return nil })
	if !errors.Is(err, mnemonica.ErrWrongTypeDefinition) {
		t.Fatalf("Sub[noNode] error = %v, want ErrWrongTypeDefinition", err)
	}
}

func TestSubMissingParentEmbed(t *testing.T) {
	fx := newFixture()
	_, err := mnemonica.Sub[Orphan](fx.userT, "Orphan", func(o *Orphan, role string) error { return nil })
	if !errors.Is(err, mnemonica.ErrWrongTypeDefinition) {
		t.Fatalf("Sub[Orphan] error = %v, want ErrWrongTypeDefinition", err)
	}
}

func TestSubValueEmbedRejected(t *testing.T) {
	fx := newFixture()
	_, err := mnemonica.Sub[Copied](fx.userT, "Copied", func(c *Copied, role string) error { return nil })
	if !errors.Is(err, mnemonica.ErrWrongTypeDefinition) {
		t.Fatalf("Sub[Copied] error = %v, want ErrWrongTypeDefinition", err)
	}
	if !strings.Contains(err.Error(), "value embed") {
		t.Fatalf("Sub[Copied] error = %v, want mention of value embed", err)
	}
}

func TestSubWrongParentType(t *testing.T) {
	widgetT := newWidget()
	// Admin embeds *User, not *Widget: defining it under Widget must fail.
	_, err := mnemonica.Sub[Admin](widgetT, "Admin", func(a *Admin, role string) error { return nil })
	if !errors.Is(err, mnemonica.ErrWrongTypeDefinition) {
		t.Fatalf("Sub[Admin] under Widget error = %v, want ErrWrongTypeDefinition", err)
	}
}

func TestSubSkipsPlainFields(t *testing.T) {
	fx := newFixture()
	taggedT, err := mnemonica.Sub[Tagged](fx.userT, "Tagged", func(x *Tagged, label string) error {
		x.Label = label
		return nil
	})
	if err != nil {
		t.Fatalf("Sub[Tagged]: %v", err)
	}
	user, err := fx.userT.New("ada")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	tagged, err := taggedT.From(user, "hello")
	if err != nil {
		t.Fatalf("From: %v", err)
	}
	if tagged.Label != "hello" {
		t.Errorf("Label = %q, want %q", tagged.Label, "hello")
	}
	if tagged.Name != "ada" {
		t.Errorf("promoted Name = %q, want %q (read-through)", tagged.Name, "ada")
	}
}

func TestSubUnexportedParentEmbedRejected(t *testing.T) {
	// An unexported parent TYPE names its embedded field with a lowercase
	// identifier, which reflect cannot wire — Sub must reject the
	// definition instead of panicking per construction.
	collection := mnemonica.NewCollection()
	userT := mnemonica.Must(mnemonica.Define[hiddenUser](collection, "User", func(u *hiddenUser, name string) error {
		u.Name = name
		return nil
	}))
	_, err := mnemonica.Sub[hiddenChild](userT, "Child", func(c *hiddenChild, role string) error { return nil })
	if !errors.Is(err, mnemonica.ErrWrongTypeDefinition) {
		t.Fatalf("Sub with unexported parent embed error = %v, want ErrWrongTypeDefinition", err)
	}
}

type hiddenUser struct {
	mnemonica.Node
	Name string
}

type hiddenChild struct {
	mnemonica.Node
	*hiddenUser
}

func TestMust(t *testing.T) {
	collection := mnemonica.NewCollection()
	td, err := mnemonica.Define[User](collection, "User", func(u *User, name string) error { return nil })
	got := mnemonica.Must(td, err)
	if got.Name() != "User" {
		t.Errorf("Must returned Name() = %q, want %q", got.Name(), "User")
	}
}

func TestMustPanics(t *testing.T) {
	collection := mnemonica.NewCollection()
	_, err := mnemonica.Define[User](collection, "User", func(u *User, name string) error { return nil })
	if err != nil {
		t.Fatalf("Define: %v", err)
	}
	_, dupErr := mnemonica.Define[User](collection, "User", func(u *User, name string) error { return nil })
	defer func() {
		if recover() == nil {
			t.Error("Must did not panic on a declaration error")
		}
	}()
	var zero *mnemonica.TypeDef[User, mnemonica.Root, string]
	_ = mnemonica.Must(zero, dupErr)
}

func TestSetHandlerAffectsNewConstructionsOnly(t *testing.T) {
	fx := newFixture()
	first, err := fx.userT.New("ada")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	fx.userT.SetHandler(func(u *User, name string) error {
		u.Name = "swapped:" + name
		return nil
	})
	second, err := fx.userT.New("grace")
	if err != nil {
		t.Fatalf("New after swap: %v", err)
	}
	if second.Name != "swapped:grace" {
		t.Errorf("second Name = %q, want %q (handler read at every construction)", second.Name, "swapped:grace")
	}
	if first.Name != "ada" {
		t.Errorf("first Name = %q, want %q (existing instance unaffected)", first.Name, "ada")
	}
}

func TestDefaultCollection(t *testing.T) {
	if mnemonica.Default == nil {
		t.Fatal("Default collection is nil")
	}
	if _, ok := mnemonica.Lookup(mnemonica.Default, "NoSuchType"); ok {
		t.Fatal("Lookup of unknown type in Default found something")
	}
	td, err := mnemonica.Define[Widget](mnemonica.Default, "P2DefaultProbe", func(w *Widget, sku string) error {
		w.SKU = sku
		return nil
	})
	if err != nil {
		t.Fatalf("Define in Default: %v", err)
	}
	widget, err := td.New("sku-1")
	if err != nil {
		t.Fatalf("New from Default-defined type: %v", err)
	}
	if widget.SKU != "sku-1" {
		t.Errorf("SKU = %q, want %q", widget.SKU, "sku-1")
	}
}

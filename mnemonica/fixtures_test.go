package mnemonica_test

import (
	"github.com/wentout/mnemonica-go/mnemonica"
)

// Fixture types for the port tests: the canonical User/Admin graph plus
// the structural edge cases (wrong parent, missing embed, value embed,
// plain field before the embed, struct without Node).

type User struct {
	mnemonica.Node
	Name string
	// cache is unexported on purpose: the parent-snapshot walk must skip it.
	cache string
}

type Admin struct {
	mnemonica.Node
	*User
	Role string
}

type SuperAdmin struct {
	mnemonica.Node
	*Admin
	Level int
}

type Widget struct {
	mnemonica.Node
	SKU string
}

// Tagged has a non-embedded field before the parent pointer: the Sub-time
// field walk must skip it.
type Tagged struct {
	mnemonica.Node
	Label string
	*User
}

// Copied embeds User BY VALUE: Sub must reject it, because a copied parent
// would break the shared-history aliasing.
type Copied struct {
	mnemonica.Node
	User
}

// Orphan embeds Node but no parent pointer at all.
type Orphan struct {
	mnemonica.Node
}

// noNode does not embed mnemonica.Node.
type noNode struct {
	Name string
}

// Task carries a struct args value: construction must store a copy, not
// alias the caller's value.
type Task struct {
	mnemonica.Node
	Note string
}

type taskArgs struct {
	text string
}

// Instances are pointers: the seal method has a pointer receiver.
var _ mnemonica.Instance = (*User)(nil)

type fixture struct {
	col    *mnemonica.Collection
	userT  *mnemonica.TypeDef[User, mnemonica.Root, string]
	adminT *mnemonica.TypeDef[Admin, User, string]
	superT *mnemonica.TypeDef[SuperAdmin, Admin, int]
}

// newFixture builds an isolated collection with the User → Admin →
// SuperAdmin graph, so tests never share registry state.
func newFixture() fixture {
	collection := mnemonica.NewCollection()
	userT := mnemonica.Must(mnemonica.Define[User](collection, "User", func(u *User, name string) error {
		u.Name = name
		return nil
	}))
	adminT := mnemonica.Must(mnemonica.Sub[Admin](userT, "Admin", func(a *Admin, role string) error {
		a.Role = role
		return nil
	}))
	superT := mnemonica.Must(mnemonica.Sub[SuperAdmin](adminT, "SuperAdmin", func(s *SuperAdmin, level int) error {
		s.Level = level
		return nil
	}))
	return fixture{col: collection, userT: userT, adminT: adminT, superT: superT}
}

// newWidget defines an unrelated root type in its own collection.
func newWidget() *mnemonica.TypeDef[Widget, mnemonica.Root, string] {
	collection := mnemonica.NewCollection()
	return mnemonica.Must(mnemonica.Define[Widget](collection, "Widget", func(w *Widget, sku string) error {
		w.SKU = sku
		return nil
	}))
}

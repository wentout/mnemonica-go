// Package lineage exercises mnemonicavet: positives are promoted writes
// into shared lineage parents; everything else must stay silent.
package lineage

import (
	"github.com/wentout/mnemonica-go/mnemonica"
	"time"
)

var collection = mnemonica.NewCollection()

type User struct {
	mnemonica.Node
	Name string
}

var UserT = mnemonica.Must(mnemonica.Define[User](collection, "User", func(u *User, name string) error {
	u.Name = name // own field on the very instance: never flagged
	return nil
}))

type Admin struct {
	mnemonica.Node
	*User
	Role string
}

var AdminT = mnemonica.Must(mnemonica.Sub[Admin](UserT, "Admin", func(a *Admin, role string) error {
	a.Role = role // own field: no diagnostic
	return nil
}))

type Plain struct {
	Name string
}

type Base struct {
	Name string
}

// Mixed embeds the lineage header but promotes Name from a NON-lineage
// embed: the write stays on Mixed's own copy.
type Mixed struct {
	mnemonica.Node
	Base
}

// Copied embeds a lineage struct BY VALUE: the promoted write lands on
// Mixed's own copy, not a shared parent.
type Copied struct {
	mnemonica.Node
	User
}

// MixedPtr promotes through a POINTER to a non-lineage struct: no shared
// parent exists, so the write is safe.
type MixedPtr struct {
	mnemonica.Node
	*Base
}

// Odd puts a named non-struct embed before the lineage parent: the walk
// must skip the non-struct and find the header behind it.
type Num int

type Odd struct {
	*Num
	*User
}

// Fake's first embed is a Node lookalike from this package; the lineage
// header sits behind it.
type Node struct {
	Lookalike int
}

type Fake struct {
	Node
	*User
}

// Loop embeds itself through a pointer before the lineage parent: the
// cycle guard must fire and the walk continue to the real parent.
type Loop struct {
	*Loop
	*User
}

// Inner/Outer: a promoted field with no lineage anywhere — the receiver
// check rejects it before any parent logic runs.
type Inner struct {
	Name string
}

type Outer struct {
	Inner
}

func (u *User) Rename(name string) {
	u.Name = name // own field on the parent itself: no diagnostic
}

func modify(admin *Admin, user *User, replacement *Admin) {
	admin.Name = "mallory"            // want "assignment to promoted field Name"
	admin.Role = "root"               // own field: no diagnostic
	user.Name = "ada"                 // the parent itself: no diagnostic
	admin.Name, admin.Role = "x", "y" // want "assignment to promoted field Name"
	admin.User = user                 // the sanctioned wiring shape (depth 1): no diagnostic
	replacement = admin               // plain identifier target: no diagnostic

	var plain Plain
	plain.Name = "p" // non-mnemonica struct: no diagnostic
	plain = Plain{}

	var mixed Mixed
	mixed.Name = "m" // promoted from a NON-lineage embed: no diagnostic

	var copied Copied
	copied.Name = "c" // value embed of a lineage struct: stays on the copy

	var mixedPtr MixedPtr
	mixedPtr.Name = "mp" // pointer to a NON-lineage embed: no shared parent to guard

	// The non-struct embed is skipped; the write reaches *User behind it.
	var odd Odd
	odd.Name = "o" // want "assignment to promoted field Name"

	// The lineage parent sits behind the lookalike.
	var fake Fake
	fake.Name = "f" // want "assignment to promoted field Name"

	// The cycle guard fires and the walk continues to the real parent.
	var loop Loop
	loop.Name = "l" // want "assignment to promoted field Name"

	inner := Inner{}
	outer := Outer{Inner: inner}
	outer.Name = "o" // promoted field, but no lineage anywhere: no diagnostic

	admin.Rename("ada") // method call, not an assignment: no diagnostic
	time.UTC = nil      // package selector: no types.Selection, no diagnostic
}

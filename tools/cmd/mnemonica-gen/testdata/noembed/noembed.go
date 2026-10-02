// Package noembed: the subtype has no embedded *Parent field — nothing to
// wire (the WithWireFunc case) — the generator must error clearly.
package noembed

import (
	"github.com/wentout/mnemonica-go/mnemonica"
)

var collection = mnemonica.NewCollection()

type User struct {
	mnemonica.Node
	Name string
}

var UserT = mnemonica.Must(mnemonica.Define[User](collection, "User", func(u *User, name string) error {
	u.Name = name
	return nil
}))

type Detached struct {
	mnemonica.Node
	Note string
}

var DetachedT = mnemonica.Must(mnemonica.Sub[Detached](UserT, "Detached", func(d *Detached, note string) error {
	d.Note = note
	return nil
}))

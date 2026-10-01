// Package nonstruct: the subtype's type argument is a named non-struct —
// the generator cannot wire it and must error.
package nonstruct

import (
	"mnemonica/mnemonica"
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

type Alias int

var AliasT = mnemonica.Must(mnemonica.Sub[Alias](UserT, "Alias", func(a *Alias, note string) error {
	return nil
}))

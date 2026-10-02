// Package notident: the Sub's parent is not an identifier — the generator
// only wires package-level handle variables.
package notident

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

func parentOf() *mnemonica.TypeDef[User, mnemonica.Root, string] {
	return UserT
}

type Admin struct {
	mnemonica.Node
	*User
	Role string
}

var AdminT = mnemonica.Must(mnemonica.Sub[Admin](parentOf(), "Admin", func(a *Admin, role string) error {
	a.Role = role
	return nil
}))

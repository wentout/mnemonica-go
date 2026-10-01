// Package collide: the parent struct already declares a method with the
// subtype's name — the generator must refuse, not overwrite.
package collide

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

type Admin struct {
	mnemonica.Node
	*User
	Role string
}

// Admin pre-exists: mnemonica-gen must error on the Sub below.
func (u *User) Admin(role string) (*Admin, error) {
	return AdminT.From(u, role)
}

var AdminT = mnemonica.Must(mnemonica.Sub[Admin](UserT, "Admin", func(a *Admin, role string) error {
	a.Role = role
	return nil
}))

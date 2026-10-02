// Package dynamicname: the Sub name is not a constant string — the
// generator must refuse to guess.
package dynamicname

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

type Admin struct {
	mnemonica.Node
	*User
	Role string
}

var dynamic = "Admin"

var AdminT = mnemonica.Must(mnemonica.Sub[Admin](UserT, dynamic, func(a *Admin, role string) error {
	a.Role = role
	return nil
}))

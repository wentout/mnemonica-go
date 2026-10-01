// Package anon: an anonymous-struct subtype argument has no name to
// generate against; the anonymous-parent Define is skipped by the var
// table (no name either), and the Sub with an anonymous child errors.
package anon

import (
	"mnemonica/mnemonica"
)

var collection = mnemonica.NewCollection()

var AnonT, anonErr = mnemonica.Define[struct {
	mnemonica.Node
	Name string
}](collection, "Anon", func(a *struct {
	mnemonica.Node
	Name string
}, name string) error {
	a.Name = name
	return nil
})

type User struct {
	mnemonica.Node
	Name string
}

var UserT = mnemonica.Must(mnemonica.Define[User](collection, "User", func(u *User, name string) error {
	u.Name = name
	return nil
}))

var AnonChildT, anonChildErr = mnemonica.Sub[struct {
	mnemonica.Node
	*User
}](UserT, "AnonChild", func(c *struct {
	mnemonica.Node
	*User
}, note string) error {
	return nil
})

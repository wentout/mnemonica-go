package mnemonica_test

import (
	"fmt"

	"mnemonica/mnemonica"
)

// The package doc example: define a root, define a subtype FROM it,
// construct along the lineage.
func Example() {
	users := mnemonica.NewCollection()
	UserT := mnemonica.Must(mnemonica.Define[User](users, "User", func(u *User, name string) error {
		u.Name = name
		return nil
	}))
	AdminT := mnemonica.Must(mnemonica.Sub[Admin](UserT, "Admin", func(a *Admin, role string) error {
		a.Role = role
		return nil
	}))

	user, _ := UserT.New("ada")
	admin, _ := AdminT.From(user, "root")

	fmt.Println(admin.Name)                 // promoted from the embedded *User
	fmt.Println(mnemonica.Is[*User](admin)) // an Admin built from a User IS a User

	// Output:
	// ada
	// true
}

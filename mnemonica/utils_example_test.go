package mnemonica_test

import (
	"errors"
	"fmt"

	"mnemonica/mnemonica"
)

// The JS docs' utils examples, ported: each Example matches its doc snippet.

func ExampleExtract() {
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

	fmt.Println(mnemonica.Extract(admin))

	// Output:
	// map[Name:ada Role:root]
}

func ExamplePick() {
	_, gadgetT, jobT := newGadgetFixture()
	account, _ := gadgetT.New("ada")
	member, _ := jobT.From(account, "admin")

	fmt.Println(mnemonica.Pick(member, "Name", "Email"))

	// Output:
	// map[Email:ada@example.com Name:member-admin]
}

func ExampleParent() {
	users := mnemonica.NewCollection()
	UserT := mnemonica.Must(mnemonica.Define[User](users, "User", func(u *User, name string) error {
		u.Name = name
		return nil
	}))
	AdminT := mnemonica.Must(mnemonica.Sub[Admin](UserT, "Admin", func(a *Admin, role string) error {
		a.Role = role
		return nil
	}))
	SuperT := mnemonica.Must(mnemonica.Sub[SuperAdmin](AdminT, "SuperAdmin", func(s *SuperAdmin, level int) error {
		s.Level = level
		return nil
	}))
	user, _ := UserT.New("ada")
	admin, _ := AdminT.From(user, "root")
	super, _ := SuperT.From(admin, 9)

	direct, _ := mnemonica.Parent(super)
	named, _ := mnemonica.Parent(super, "User.Admin") // contiguous dotted path
	fmt.Println(direct == admin, named == admin)

	// Output:
	// true true
}

func ExampleClone() {
	users := mnemonica.NewCollection()
	UserT := mnemonica.Must(mnemonica.Define[User](users, "User", func(u *User, name string) error {
		u.Name = name
		return nil
	}))
	user, _ := UserT.New("ada")

	clone, _ := mnemonica.Clone(user)
	fmt.Println(clone.Name, clone != user)

	// Output:
	// ada true
}

func ExampleFork() {
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

	forked, _ := mnemonica.Fork(admin, "operator")
	fmt.Println(forked.Role, forked.Name)

	// Output:
	// operator ada
}

func ExampleMerge() {
	// The UTILS.md merge example: a provides the fields, b fills the
	// non-overlapping keys.
	types := mnemonica.NewCollection()
	UserT := mnemonica.Must(mnemonica.Define[MergeUser](types, "User", func(u *MergeUser, name string) error {
		u.Name = name
		u.Age = 30
		return nil
	}))
	RoleT := mnemonica.Must(mnemonica.Define[MergeRole](types, "Role", func(r *MergeRole, role string) error {
		r.Role = role
		return nil
	}))
	user, _ := UserT.New("alice")
	role, _ := RoleT.New("admin")

	merged, _ := mnemonica.Merge(user, role, "alice")
	fmt.Println(mnemonica.Extract(merged))

	// Output:
	// map[Age:30 Name:alice Role:admin]
}

func ExampleException() {
	users := mnemonica.NewCollection()
	UserT := mnemonica.Must(mnemonica.Define[User](users, "User", func(u *User, name string) error {
		u.Name = name
		return nil
	}))
	user, _ := UserT.New("ada")

	boom := errors.New("boom")
	err := mnemonica.NewException(user, boom, 1, 2, 3)
	fmt.Println(errors.Is(err, boom))
	fmt.Println(err.Instance() == user)

	// Output:
	// true
	// true
}

func ExampleParse() {
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

	parsed := mnemonica.Parse(admin)
	fmt.Println(parsed.Name, parsed.Parent == user)

	// Output:
	// Admin true
}

func ExampleToJSON() {
	_, gadgetT, jobT := newGadgetFixture()
	account, _ := gadgetT.New("ada")
	member, _ := jobT.From(account, "admin")

	encoded, _ := mnemonica.ToJSON(member)
	fmt.Println(encoded)

	// Output:
	// {"Email":"ada@example.com","Name":"member-admin","Role":"admin"}
}

func ExampleConstructorSequence() {
	users := mnemonica.NewCollection()
	UserT := mnemonica.Must(mnemonica.Define[User](users, "User", func(u *User, name string) error {
		u.Name = name
		return nil
	}))
	AdminT := mnemonica.Must(mnemonica.Sub[Admin](UserT, "Admin", func(a *Admin, role string) error {
		a.Role = role
		return nil
	}))
	SuperT := mnemonica.Must(mnemonica.Sub[SuperAdmin](AdminT, "SuperAdmin", func(s *SuperAdmin, level int) error {
		s.Level = level
		return nil
	}))
	user, _ := UserT.New("ada")
	admin, _ := AdminT.From(user, "root")
	super, _ := SuperT.From(admin, 9)

	fmt.Println(mnemonica.ConstructorSequence(super))

	// Output:
	// [SuperAdmin Admin User]
}

package mnemonica_test

import (
	"context"
	"fmt"

	"mnemonica/mnemonica"
)

// The P4 story: context bound to calls (Go) meets context bound to data
// (mnemonica). The request dies; the values live on with the data.

func ExampleTypeDef_NewCtx() {
	users := mnemonica.NewCollection()
	UserT := mnemonica.Must(mnemonica.Define[User](users, "User", func(u *User, name string) error {
		u.Name = name
		return nil
	}))
	type requestKey struct{}
	requestCtx, cancel := context.WithCancel(context.WithValue(context.Background(), requestKey{}, "req-42"))

	user, _ := UserT.NewCtx(requestCtx, "ada")
	cancel() // the request is over

	work := mnemonica.ContextOf(user)
	fmt.Println(work.Value(requestKey{}), work.Err())

	// Output:
	// req-42 <nil>
}

func ExampleTypeDef_FromCtx() {
	users := mnemonica.NewCollection()
	UserT := mnemonica.Must(mnemonica.Define[User](users, "User", func(u *User, name string) error {
		u.Name = name
		return nil
	}))
	AdminT := mnemonica.Must(mnemonica.Sub[Admin](UserT, "Admin", func(a *Admin, role string) error {
		a.Role = role
		return nil
	}))
	type requestKey struct{}
	parent, _ := UserT.NewCtx(context.WithValue(context.Background(), requestKey{}, "parent-req"), "ada")
	admin, _ := AdminT.FromCtx(context.WithValue(context.Background(), requestKey{}, "child-req"), parent, "root")

	fmt.Println(mnemonica.ContextOf(admin).Value(requestKey{}))

	// Output:
	// child-req
}

func ExampleContextOf() {
	// A worker goroutine outlives its request: it keeps the lineage's
	// values (trace ids, request tags) through ContextOf, wrapped in its
	// own cancellation — the request's cancel cannot touch it.
	users := mnemonica.NewCollection()
	UserT := mnemonica.Must(mnemonica.Define[User](users, "User", func(u *User, name string) error {
		u.Name = name
		return nil
	}))
	type traceKey struct{}
	requestCtx, requestDone := context.WithCancel(context.WithValue(context.Background(), traceKey{}, "span-1"))
	user, _ := UserT.NewCtx(requestCtx, "ada")
	requestDone() // request returned

	workerCtx, workerDone := context.WithCancel(mnemonica.ContextOf(user))
	defer workerDone()

	fmt.Println(workerCtx.Value(traceKey{}), workerCtx.Err(), mnemonica.ContextOf(user).Err())

	// Output:
	// span-1 <nil> <nil>
}

func ExampleForkCtx() {
	// Fork keeps the source's request lineage by default; ForkCtx moves
	// the fork into a different one.
	users := mnemonica.NewCollection()
	UserT := mnemonica.Must(mnemonica.Define[User](users, "User", func(u *User, name string) error {
		u.Name = name
		return nil
	}))
	AdminT := mnemonica.Must(mnemonica.Sub[Admin](UserT, "Admin", func(a *Admin, role string) error {
		a.Role = role
		return nil
	}))
	type requestKey struct{}
	user, _ := UserT.New("ada")
	source, _ := AdminT.FromCtx(context.WithValue(context.Background(), requestKey{}, "old-request"), user, "root")

	forked, _ := mnemonica.ForkCtx(source, context.WithValue(context.Background(), requestKey{}, "new-request"), "operator")
	fmt.Println(mnemonica.ContextOf(forked).Value(requestKey{}))

	// Output:
	// new-request
}

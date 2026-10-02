package mnemonica_test

import (
	"fmt"
	"sync"
	"testing"

	"github.com/wentout/mnemonica-go/mnemonica"
)

// TestConcurrentUse hammers the registry from many goroutines:
// definitions, lookups, constructions, handler swaps, hook registrations
// and Props reads interleave. scripts/check runs the suite with -race; a
// racy registry, handler slot or hook list would fail there.
func TestConcurrentUse(t *testing.T) {
	collection := mnemonica.NewCollection()
	const workers = 8
	const iterations = 100

	errs := make(chan error, workers*iterations)
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			rootName := fmt.Sprintf("T%d", worker)
			root, err := mnemonica.Define[User](collection, rootName, func(u *User, name string) error {
				u.Name = name
				return nil
			})
			if err != nil {
				errs <- err
				return
			}
			sub, err := mnemonica.Sub[Admin](root, "Sub", func(a *Admin, role string) error {
				a.Role = role
				return nil
			})
			if err != nil {
				errs <- err
				return
			}
			err = collection.RegisterHook(mnemonica.HookPreCreation, func(*mnemonica.HookData) error {
				return nil
			})
			if err != nil {
				errs <- err
				return
			}
			for i := 0; i < iterations; i++ {
				if _, ok := collection.Lookup(fmt.Sprintf("%s.Sub", rootName)); !ok {
					errs <- fmt.Errorf("lookup %s.Sub failed", rootName)
					return
				}
				user, err := root.New("ada")
				if err != nil {
					errs <- err
					return
				}
				if _, err := sub.From(user, "root"); err != nil {
					errs <- err
					return
				}
				record, err := mnemonica.Props(user)
				if err != nil {
					errs <- err
					return
				}
				if record.Type.Path() != rootName {
					errs <- fmt.Errorf("Props Type.Path() = %q, want %q", record.Type.Path(), rootName)
					return
				}
				root.SetHandler(func(u *User, name string) error {
					u.Name = "swapped"
					return nil
				})
				swapped, err := root.New("ignored")
				if err != nil {
					errs <- err
					return
				}
				if swapped.Name != "swapped" {
					errs <- fmt.Errorf("after swap Name = %q, want %q", swapped.Name, "swapped")
					return
				}
			}
		}(w)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}
}

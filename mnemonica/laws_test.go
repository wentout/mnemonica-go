package mnemonica_test

import (
	"errors"
	"testing"

	"mnemonica/mnemonica"
)

// Property tests for the HoTT correspondence's EXACT claims — the Go
// counterparts of the JS core's test/hott-laws.js ("Executable witnesses
// for docs/hott-correspondence.md"). Each fuzz target pins one law; the
// seed corpus makes it run as a unit test under plain `go test` (so the
// property bodies count toward coverage and -race), and every property is
// deterministic given its seed.
//
// All six laws are predicates over arbitrary construction arguments on a
// fixed type graph, so all six are native fuzz tests (testing/quick would
// add nothing a seed corpus does not). Go cannot declare types at runtime,
// so the randomized chains run over the declared User → Admin →
// SuperAdmin graph with fuzz-derived args and depths; genuinely deep
// chains (depth 1/10/100) are a benchmark concern — see
// chain_bench_test.go — because each level needs a declared type.

// lawMust unwraps a construction result in property tests. It panics
// rather than taking *testing.T because Go cannot mix a leading parameter
// with a multi-value call — lawMust(x.New()) does not compile — and a
// construction failure in a law witness is a test failure either way.
func lawMust[T any](value T, err error) T {
	if err != nil {
		panic(err)
	}
	return value
}

// lawChain walks the lineage exactly as hott-laws.js walks the prototype
// chain: self first, then ancestors to the root.
func lawChain(x mnemonica.Instance) []mnemonica.Instance {
	var chain []mnemonica.Instance
	for cursor := x; ; {
		chain = append(chain, cursor)
		parent, ok := mnemonica.Parent(cursor)
		if !ok {
			break
		}
		cursor = parent
	}
	return chain
}

// newLawGraph builds the fixed three-level graph in an isolated collection.
func newLawGraph(t *testing.T) (*mnemonica.TypeDef[User, mnemonica.Root, string], *mnemonica.TypeDef[Admin, User, string], *mnemonica.TypeDef[SuperAdmin, Admin, int]) {
	t.Helper()
	collection := mnemonica.NewCollection()
	userT, err := mnemonica.Define[User](collection, "LawUser", func(u *User, name string) error {
		u.Name = name
		return nil
	})
	if err != nil {
		t.Fatalf("Define: %v", err)
	}
	adminT, err := mnemonica.Sub[Admin](userT, "LawAdmin", func(a *Admin, role string) error {
		a.Role = role
		return nil
	})
	if err != nil {
		t.Fatalf("Sub: %v", err)
	}
	superT, err := mnemonica.Sub[SuperAdmin](adminT, "LawSuperAdmin", func(s *SuperAdmin, level int) error {
		s.Level = level
		return nil
	})
	if err != nil {
		t.Fatalf("Sub: %v", err)
	}
	return userT, adminT, superT
}

// FuzzPathTypes: the prototype chain IS the identity path to root
// (hott-laws.js: "path types"). The lineage holds the exact parent
// instances, nearest first.
func FuzzPathTypes(f *testing.F) {
	f.Add("ada", "root", uint(0))
	f.Add("grace", "operator", uint(1))
	f.Add("mallory", "intruder", uint(2))
	f.Fuzz(func(t *testing.T, name, role string, depth uint) {
		userT, adminT, superT := newLawGraph(t)
		level := 1 + int(depth%3)
		user := lawMust(userT.New(name))
		leaf := mnemonica.Instance(user)
		var admin *Admin
		if level >= 2 {
			admin = lawMust(adminT.From(user, role))
			leaf = admin
		}
		if level >= 3 {
			leaf = lawMust(superT.From(admin, int(depth)))
		}

		chain := lawChain(leaf)
		if len(chain) != level {
			t.Fatalf("chain length = %d, want %d", len(chain), level)
		}
		if chain[0] != leaf {
			t.Error("chain does not start at the instance itself")
		}
		if level == 2 && chain[1] != mnemonica.Instance(user) {
			t.Error("the root is not reachable as the last ancestor")
		}
		if level == 3 && (chain[1] != mnemonica.Instance(admin) || chain[2] != mnemonica.Instance(user)) {
			t.Error("the chain does not materialize mid before root, nearest first")
		}
	})
}

// FuzzMonadRightIdentity: binding a subtype preserves the parent context
// (hott-laws.js: "monad right identity"). Read-through shows every
// ancestor's fields, and the type IS every ancestor nominally.
func FuzzMonadRightIdentity(f *testing.F) {
	f.Add("ada", "root", 7)
	f.Add("grace", "operator", -3)
	f.Fuzz(func(t *testing.T, name, role string, level int) {
		userT, adminT, superT := newLawGraph(t)
		user := lawMust(userT.New(name))
		admin := lawMust(adminT.From(user, role))
		leaf := lawMust(superT.From(admin, level))

		if !mnemonica.Is[*User](leaf) || !mnemonica.Is[*Admin](leaf) {
			t.Error("leaf is not its ancestors nominally")
		}
		if leaf.Name != name {
			t.Errorf("root field through the chain = %q, want %q", leaf.Name, name)
		}
		if leaf.Role != role {
			t.Errorf("mid field through the chain = %q, want %q", leaf.Role, role)
		}
		if leaf.Level != level {
			t.Errorf("own field = %d, want %d", leaf.Level, level)
		}
		if !mnemonica.Is[*User](admin) {
			t.Error("admin is not a User along its chain")
		}
	})
}

// FuzzMonadAssociativity: chain extension order is irrelevant to grouping
// (hott-laws.js: "monad associativity"). Go constructs strictly stepwise —
// there is no second grouping to compare — so the witness pins what the
// law claims in either grouping: the chain holds the FULL path, every
// hop materialized, and every ancestor nominal.
func FuzzMonadAssociativity(f *testing.F) {
	f.Add("a", "b", 1)
	f.Add("root", "mid", 99)
	f.Fuzz(func(t *testing.T, name, role string, level int) {
		userT, adminT, superT := newLawGraph(t)
		// ((root >>= Admin) >>= SuperAdmin), built only stepwise.
		leaf := lawMust(superT.From(lawMust(adminT.From(lawMust(userT.New(name)), role)), level))

		chain := lawChain(leaf)
		if len(chain) != 3 {
			t.Fatalf("chain length = %d, want the full 3-hop path", len(chain))
		}
		if !mnemonica.Is[*User](leaf) || !mnemonica.Is[*Admin](leaf) || !mnemonica.Is[*SuperAdmin](leaf) {
			t.Error("the stepwise chain lost an ancestor")
		}
		if chain[0] != mnemonica.Instance(leaf) {
			t.Error("the chain does not start at the leaf")
		}
	})
}

// lawNominalA and lawNominalB are byte-identical in shape and handler
// behavior — the law's whole point is that they are still different types.
type lawNominalA struct {
	mnemonica.Node
	Value string
}

type lawNominalB struct {
	mnemonica.Node
	Value string
}

// FuzzNominalIdentity: identity is nominal — shape does not identify
// (hott-laws.js: "identity is nominal (univalence intuition)"), and the
// name is frozen: re-declaring refuses the lift.
func FuzzNominalIdentity(f *testing.F) {
	f.Add("same-value")
	f.Fuzz(func(t *testing.T, value string) {
		collection := mnemonica.NewCollection()
		nominalT, err := mnemonica.Define[lawNominalA](collection, "LawNominal", func(a *lawNominalA, v string) error {
			a.Value = v
			return nil
		})
		if err != nil {
			t.Fatalf("Define A: %v", err)
		}
		otherT, err := mnemonica.Define[lawNominalB](collection, "LawNominalOther", func(b *lawNominalB, v string) error {
			b.Value = v
			return nil
		})
		if err != nil {
			t.Fatalf("Define B: %v", err)
		}
		a := lawMust(nominalT.New(value))
		b := lawMust(otherT.New(value))

		if a.Value != b.Value {
			t.Error("same shape expected: identical handlers over identical structs")
		}
		if mnemonica.Is[*lawNominalB](a) || mnemonica.Is[*lawNominalA](b) {
			t.Error("shape leaked into identity: the two types are interchangeable")
		}
		if _, ok := mnemonica.As[*lawNominalB](a); ok {
			t.Error("As crossed into the same-shaped foreign type")
		}
		_, err = mnemonica.Define[lawNominalA](collection, "LawNominal", func(a *lawNominalA, v string) error {
			return nil
		})
		if !errors.Is(err, mnemonica.ErrAlreadyDeclared) {
			t.Errorf("re-declaring the name error = %v, want ErrAlreadyDeclared", err)
		}
	})
}

// FuzzPathUniqueness: different construction order, different identity
// (hott-laws.js: "path uniqueness"). Same type, same endpoint name, two
// different paths — the carried context differs and the parents differ.
func FuzzPathUniqueness(f *testing.F) {
	f.Add("first", "second", "alpha", "beta")
	f.Fuzz(func(t *testing.T, nameA, nameB, roleA, roleB string) {
		userT, adminT, _ := newLawGraph(t)
		rootA := lawMust(userT.New(nameA))
		rootB := lawMust(userT.New(nameB))
		direct := lawMust(adminT.From(rootA, roleA))
		viaFresh := lawMust(adminT.From(rootB, roleB))

		if !mnemonica.Is[*Admin](direct) || !mnemonica.Is[*Admin](viaFresh) {
			t.Error("both paths must produce the same nominal type")
		}
		if direct.Role == viaFresh.Role && roleA != roleB {
			t.Error("the carried context does not differ across paths")
		}
		parentA, okA := mnemonica.Parent(direct)
		parentB, okB := mnemonica.Parent(viaFresh)
		if !okA || !okB || parentA == parentB {
			t.Error("different paths must have different parents")
		}
		if parentA != mnemonica.Instance(rootA) || parentB != mnemonica.Instance(rootB) {
			t.Error("each path must carry its own root")
		}
	})
}

// FuzzComparatorsComparable: naming-path extraction is deterministic
// (hott-laws.js: "comparators are comparable"). The comparator IS the
// naming path: same naming path → same comparator, different paths stay
// distinguishable, and identical comparators never make instances
// identical.
func FuzzComparatorsComparable(f *testing.F) {
	f.Add("ada", "root", 5)
	f.Fuzz(func(t *testing.T, name, role string, level int) {
		userT, adminT, superT := newLawGraph(t)
		user := lawMust(userT.New(name))
		admin := lawMust(adminT.From(user, role))
		leaf := lawMust(superT.From(admin, level))
		sibling := lawMust(superT.From(admin, level+1))

		spine := mnemonica.ConstructorSequence(leaf)
		if second := mnemonica.ConstructorSequence(leaf); len(second) != len(spine) {
			t.Fatal("the same instance produced two different spines")
		} else {
			for index := range spine {
				if second[index] != spine[index] {
					t.Fatal("the same instance produced two different spines")
				}
			}
		}
		if siblingSpine := mnemonica.ConstructorSequence(sibling); len(siblingSpine) != len(spine) {
			t.Error("same naming path produced different comparators")
		} else {
			for index := range spine {
				if siblingSpine[index] != spine[index] {
					t.Error("same naming path produced different comparators")
				}
			}
		}
		if mnemonica.Instance(leaf) == mnemonica.Instance(sibling) {
			t.Error("identical comparators made the instances identical")
		}
		if adminSpine := mnemonica.ConstructorSequence(admin); len(adminSpine) == len(spine) {
			t.Error("different paths stayed indistinguishable")
		}
	})
}

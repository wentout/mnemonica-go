package mnemonicatest

import (
	"fmt"
	"strings"
	"testing"

	"github.com/wentout/mnemonica-go/mnemonica"
)

// Fixture: the canonical User → Admin graph, defined with parent snapshots
// so AssertParentsUnchanged has a baseline.

type User struct {
	mnemonica.Node
	Name string
}

type Admin struct {
	mnemonica.Node
	*User
	Role string
}

type fixture struct {
	userT  *mnemonica.TypeDef[User, mnemonica.Root, string]
	adminT *mnemonica.TypeDef[Admin, User, string]
}

func newFixture() fixture {
	collection := mnemonica.NewCollection()
	userT := mnemonica.Must(mnemonica.Define[User](collection, "User", func(u *User, name string) error {
		u.Name = name
		return nil
	}, mnemonica.WithParentSnapshots(true)))
	adminT := mnemonica.Must(mnemonica.Sub[Admin](userT, "Admin", func(a *Admin, role string) error {
		a.Role = role
		return nil
	}))
	return fixture{userT: userT, adminT: adminT}
}

// errorRecorder captures t.Errorf output instead of failing the test, so
// the assertion's failure paths can be exercised in a green suite. It
// embeds the real testing.TB: the unexported seal method promotes from it,
// which testing.TB requires of any implementation.
type errorRecorder struct {
	testing.TB
	messages []string
}

func (r *errorRecorder) Errorf(format string, args ...any) {
	r.messages = append(r.messages, fmt.Sprintf(format, args...))
}

// TestAssertParentsUnchangedPassesWhenLineageIsStable: no failures are
// reported for a lineage whose ancestors are untouched. This one runs
// against the REAL testing.TB: zero Errorf calls is what "pass" means.
func TestAssertParentsUnchangedPassesWhenLineageIsStable(t *testing.T) {
	fx := newFixture()
	user, err := fx.userT.New("ada")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	admin, err := fx.adminT.From(user, "root")
	if err != nil {
		t.Fatalf("From: %v", err)
	}
	// Mutating the CHILD's own field is fine: the guard is about ancestors.
	admin.Role = "operator"
	AssertParentsUnchanged(t, admin)
}

// TestAssertParentsUnchangedReportsMutation: the adapted C2.4 violation —
// a promoted write into the shared parent — is caught and reported.
func TestAssertParentsUnchangedReportsMutation(t *testing.T) {
	fx := newFixture()
	user, err := fx.userT.New("ada")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	admin, err := fx.adminT.From(user, "root")
	if err != nil {
		t.Fatalf("From: %v", err)
	}
	user.Name = "mallory" // the mutation the guard exists for
	recorder := &errorRecorder{TB: t}
	AssertParentsUnchanged(recorder, admin)
	if len(recorder.messages) != 1 {
		t.Fatalf("reported failures = %v, want exactly one", recorder.messages)
	}
	if !strings.Contains(recorder.messages[0], "User.Name") || !strings.Contains(recorder.messages[0], "mallory") {
		t.Errorf("failure = %q, want it to name User.Name and the new value", recorder.messages[0])
	}
}

// TestAssertParentsUnchangedReportsMissingSnapshot: a type defined without
// snapshots cannot be asserted against.
func TestAssertParentsUnchangedReportsMissingSnapshot(t *testing.T) {
	collection := mnemonica.NewCollection()
	userT := mnemonica.Must(mnemonica.Define[User](collection, "User", func(u *User, name string) error {
		u.Name = name
		return nil
	}))
	user, err := userT.New("ada")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	recorder := &errorRecorder{TB: t}
	AssertParentsUnchanged(recorder, user)
	if len(recorder.messages) != 1 {
		t.Fatalf("reported failures = %v, want exactly one", recorder.messages)
	}
	if !strings.Contains(recorder.messages[0], "WithParentSnapshots") {
		t.Errorf("failure = %q, want it to name the option", recorder.messages[0])
	}
}

// TestAssertParentsUnchangedReportsNonInstance: a value without a
// construction record is a failure, not a panic.
func TestAssertParentsUnchangedReportsNonInstance(t *testing.T) {
	recorder := &errorRecorder{TB: t}
	AssertParentsUnchanged(recorder, 42)
	if len(recorder.messages) != 1 {
		t.Fatalf("reported failures = %v, want exactly one", recorder.messages)
	}
	if !strings.Contains(recorder.messages[0], "not a mnemonica instance") {
		t.Errorf("failure = %q, want the not-an-instance message", recorder.messages[0])
	}
}

// TestAssertParentsUnchangedDeepLineage: every ancestor is checked, not
// just the direct parent — the mutation of a grandparent is caught.
func TestAssertParentsUnchangedDeepLineage(t *testing.T) {
	collection := mnemonica.NewCollection()
	userT := mnemonica.Must(mnemonica.Define[User3](collection, "User", func(u *User3, name string) error {
		u.Name = name
		return nil
	}, mnemonica.WithParentSnapshots(true)))
	adminT := mnemonica.Must(mnemonica.Sub[Admin3](userT, "Admin", func(a *Admin3, role string) error {
		a.Role = role
		return nil
	}))
	managerT := mnemonica.Must(mnemonica.Sub[Manager3](adminT, "Manager", func(m *Manager3, scope int) error {
		m.Scope = scope
		return nil
	}))
	user, err := userT.New("ada")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	admin, err := adminT.From(user, "root")
	if err != nil {
		t.Fatalf("From: %v", err)
	}
	manager, err := managerT.From(admin, 7)
	if err != nil {
		t.Fatalf("From manager: %v", err)
	}
	recorder := &errorRecorder{TB: t}
	AssertParentsUnchanged(recorder, manager)
	if len(recorder.messages) != 0 {
		t.Errorf("reported failures = %v, want none", recorder.messages)
	}
	// Mutating the GRANDPARENT is caught too: both ancestors are walked.
	user.Name = "mallory"
	AssertParentsUnchanged(recorder, manager)
	if len(recorder.messages) != 1 {
		t.Fatalf("reported failures = %v, want exactly one", recorder.messages)
	}
	if !strings.Contains(recorder.messages[0], "User3.Name") {
		t.Errorf("failure = %q, want it to name User3.Name", recorder.messages[0])
	}
}

// A three-level fixture local to the deep-lineage test, to keep the shared
// fixture small. Exported names matter: an unexported embedded parent type
// would name its field with a lowercase identifier, which reflect cannot
// wire (Sub rejects such definitions at define time).

type User3 struct {
	mnemonica.Node
	Name string
}

type Admin3 struct {
	mnemonica.Node
	*User3
	Role string
}

type Manager3 struct {
	mnemonica.Node
	*Admin3
	Scope int
}

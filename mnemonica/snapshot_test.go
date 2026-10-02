package mnemonica_test

import (
	"testing"

	"github.com/wentout/mnemonica-go/mnemonica"
)

func TestParentSnapshotCaptured(t *testing.T) {
	// WithParentSnapshots makes construction copy every ancestor's
	// user-visible fields; the subtype inherits the option from its
	// parent (JS config inheritance).
	collection := mnemonica.NewCollection()
	userT := mnemonica.Must(mnemonica.Define[User](collection, "User", func(u *User, name string) error {
		u.Name = name
		return nil
	}, mnemonica.WithParentSnapshots(true)))
	adminT := mnemonica.Must(mnemonica.Sub[Admin](userT, "Admin", func(a *Admin, role string) error {
		a.Role = role
		return nil
	}))
	user, err := userT.New("ada")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	user.cache = "warm" // unexported fixture state: the snapshot must not see it
	admin, err := adminT.From(user, "root")
	if err != nil {
		t.Fatalf("From: %v", err)
	}
	record, err := mnemonica.Props(admin)
	if err != nil {
		t.Fatalf("Props: %v", err)
	}
	// User contributes exactly one entry: Name. The unexported cache field
	// and the embedded Node/*User fields are skipped.
	if len(record.Snapshot) != 1 {
		t.Fatalf("Snapshot = %+v, want exactly one entry (User.Name)", record.Snapshot)
	}
	entry := record.Snapshot[0]
	if entry.Type != "User" || entry.Field != "Name" || entry.Value != "ada" {
		t.Errorf("Snapshot[0] = %+v, want {User Name ada}", entry)
	}
	// Mutating the parent afterwards does not rewrite the baseline, and
	// mutating the unexported field is invisible to it either way.
	user.Name = "changed"
	user.cache = "cold"
	if user.cache != "cold" {
		t.Fatal("fixture: unexported field not writable")
	}
	record, err = mnemonica.Props(admin)
	if err != nil {
		t.Fatalf("Props: %v", err)
	}
	if record.Snapshot[0].Value != "ada" {
		t.Errorf("Snapshot[0].Value = %v, want %q (baseline is immutable)", record.Snapshot[0].Value, "ada")
	}
}

func TestRootSnapshotIsNil(t *testing.T) {
	// Roots have no ancestors: the snapshot stays nil even when enabled.
	collection := mnemonica.NewCollection()
	userT := mnemonica.Must(mnemonica.Define[User](collection, "User", func(u *User, name string) error {
		u.Name = name
		return nil
	}, mnemonica.WithParentSnapshots(true)))
	user, err := userT.New("ada")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	record, err := mnemonica.Props(user)
	if err != nil {
		t.Fatalf("Props: %v", err)
	}
	if record.Snapshot != nil {
		t.Errorf("root Snapshot = %v, want nil", record.Snapshot)
	}
}

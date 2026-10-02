package mnemonica_test

import (
	"errors"
	"testing"
	"time"

	"github.com/wentout/mnemonica-go/mnemonica"
)

func TestPropsOfSubtype(t *testing.T) {
	fx := newFixture()
	user, err := fx.userT.New("ada")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	admin, err := fx.adminT.From(user, "root")
	if err != nil {
		t.Fatalf("From: %v", err)
	}
	record, err := mnemonica.Props(admin)
	if err != nil {
		t.Fatalf("Props: %v", err)
	}
	if record.Type.Name() != "Admin" || record.Type.Path() != "User.Admin" {
		t.Errorf("Type = (%q, %q), want (Admin, User.Admin)", record.Type.Name(), record.Type.Path())
	}
	if record.Parent != user {
		t.Errorf("Parent = %v, want the user instance %p", record.Parent, user)
	}
	if record.Args != "root" {
		t.Errorf("Args = %v, want %q", record.Args, "root")
	}
	if record.Timestamp.IsZero() {
		t.Error("Timestamp is zero")
	}
	if record.Creator.Path() != "User.Admin" {
		t.Errorf("Creator.Path() = %q, want %q", record.Creator.Path(), "User.Admin")
	}
	if record.Collection != fx.col {
		t.Errorf("Collection = %p, want %p", record.Collection, fx.col)
	}
	if record.Type.Collection() != fx.col {
		t.Errorf("Type.Collection() = %p, want %p", record.Type.Collection(), fx.col)
	}
	subtypeNames := make([]string, 0, len(record.Subtypes))
	for _, sub := range record.Subtypes {
		subtypeNames = append(subtypeNames, sub.Name())
	}
	if len(subtypeNames) != 1 || subtypeNames[0] != "SuperAdmin" {
		t.Errorf("Subtypes = %v, want [SuperAdmin] (declared subtypes of Admin)", subtypeNames)
	}
	if record.Self != admin {
		t.Errorf("Self = %v, want the admin itself %p", record.Self, admin)
	}
	if record.Stack != "" {
		t.Errorf("Stack = %q, want empty without submitStack", record.Stack)
	}
	if record.Snapshot != nil {
		t.Errorf("Snapshot = %v, want nil without WithParentSnapshots", record.Snapshot)
	}
}

func TestPropsOfRoot(t *testing.T) {
	fx := newFixture()
	user, err := fx.userT.New("ada")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	record, err := mnemonica.Props(user)
	if err != nil {
		t.Fatalf("Props: %v", err)
	}
	if record.Parent != nil {
		t.Errorf("Parent = %v, want nil for a root", record.Parent)
	}
	if !record.Timestamp.Before(time.Now().Add(time.Second)) {
		t.Errorf("Timestamp %v is in the future", record.Timestamp)
	}
	subtypeNames := make([]string, 0, len(record.Subtypes))
	for _, sub := range record.Subtypes {
		subtypeNames = append(subtypeNames, sub.Name())
	}
	if len(subtypeNames) != 1 || subtypeNames[0] != "Admin" {
		t.Errorf("Subtypes = %v, want [Admin]", subtypeNames)
	}
}

func TestPropsArgsHoldAValueCopy(t *testing.T) {
	// The record stores the args value as passed: value types are copied
	// by the call itself, so later tampering with the caller's value
	// cannot rewrite history.
	collection := mnemonica.NewCollection()
	taskT := mnemonica.Must(mnemonica.Define[Task](collection, "Task", func(t *Task, args taskArgs) error {
		t.Note = args.text
		return nil
	}))
	argValue := taskArgs{text: "hello"}
	task, err := taskT.New(argValue)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	argValue.text = "tampered"
	record, err := mnemonica.Props(task)
	if err != nil {
		t.Fatalf("Props: %v", err)
	}
	if record.Args != (taskArgs{text: "hello"}) {
		t.Errorf("Args = %v, want %v (record must hold the construction-time value)", record.Args, taskArgs{text: "hello"})
	}
}

func TestPropsNotAnInstance(t *testing.T) {
	cases := map[string]any{
		"nil":         nil,
		"int":         42,
		"string":      "not an instance",
		"plain value": User{},
		"plain ptr":   &struct{ X int }{},
	}
	for name, value := range cases {
		_, err := mnemonica.Props(value)
		if !errors.Is(err, mnemonica.ErrNotAnInstance) {
			t.Errorf("Props(%s) error = %v, want ErrNotAnInstance", name, err)
		}
	}
}

func TestPropsOfUnconstructedNode(t *testing.T) {
	// A hand-made User embeds Node but never went through construction:
	// no record, so it is not an instance for Props purposes.
	_, err := mnemonica.Props(&User{})
	if !errors.Is(err, mnemonica.ErrNotAnInstance) {
		t.Fatalf("Props(&User{}) error = %v, want ErrNotAnInstance", err)
	}
}

func TestPropsSelfRoundTrip(t *testing.T) {
	// Props(record.Self) must return the equivalent record (JS getProps
	// idiom for finding your place back in the chain).
	fx := newFixture()
	user, err := fx.userT.New("ada")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	admin, err := fx.adminT.From(user, "root")
	if err != nil {
		t.Fatalf("From: %v", err)
	}
	record, err := mnemonica.Props(admin)
	if err != nil {
		t.Fatalf("Props: %v", err)
	}
	again, err := mnemonica.Props(record.Self)
	if err != nil {
		t.Fatalf("Props(Self): %v", err)
	}
	if again.Type.Path() != record.Type.Path() || again.Parent != record.Parent {
		t.Errorf("Props(Self) = (%q, %v), want (%q, %v)",
			again.Type.Path(), again.Parent, record.Type.Path(), record.Parent)
	}
}

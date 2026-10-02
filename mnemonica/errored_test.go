package mnemonica_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/wentout/mnemonica-go/mnemonica"
)

func TestErroredInstanceShape(t *testing.T) {
	collection := mnemonica.NewCollection()
	boom := errors.New("the original failure")
	failing := mnemonica.Must(mnemonica.Define[User](collection, "User", func(u *User, name string) error {
		u.Name = name // partially applied before the failure
		return boom
	}))
	_, err := failing.New("ada")
	var errored *mnemonica.ErroredInstance
	if !errors.As(err, &errored) {
		t.Fatalf("New error = %v, want an *ErroredInstance", err)
	}
	message := errored.Error()
	if !strings.Contains(message, "User") || !strings.Contains(message, "the original failure") {
		t.Errorf("Error() = %q, want the type path and the original error", message)
	}
	if !errors.Is(errored, boom) {
		t.Error("errors.Is(errored, boom) = false, want true (Unwrap)")
	}
	half, ok := errored.Instance.(*User)
	if !ok {
		t.Fatalf("carried instance = %T, want *User", errored.Instance)
	}
	// The half-built instance is inspectable: the constructor ran
	// partially before it failed.
	if half.Name != "ada" {
		t.Errorf("half-built Name = %q, want %q", half.Name, "ada")
	}
}

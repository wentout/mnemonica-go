package mnemonica_test

import (
	"errors"
	"testing"

	"mnemonica/mnemonica"
)

func TestHookExecutionOrder(t *testing.T) {
	// C5.2: preCreation runs collection hooks then type hooks;
	// postCreation runs type hooks then collection hooks.
	collection := mnemonica.NewCollection()
	var order []string
	mustRegister(t, collection.RegisterHook(mnemonica.HookPreCreation, recordHook(&order, "collection-pre")))
	mustRegister(t, collection.RegisterHook(mnemonica.HookPostCreation, recordHook(&order, "collection-post")))
	userT := mnemonica.Must(mnemonica.Define[User](collection, "User", func(u *User, name string) error {
		u.Name = name
		return nil
	}))
	mustRegister(t, userT.RegisterHook(mnemonica.HookPreCreation, recordHook(&order, "type-pre")))
	mustRegister(t, userT.RegisterHook(mnemonica.HookPostCreation, recordHook(&order, "type-post")))
	if _, err := userT.New("ada"); err != nil {
		t.Fatalf("New: %v", err)
	}
	want := []string{"collection-pre", "type-pre", "type-post", "collection-post"}
	if !equalStrings(order, want) {
		t.Errorf("hook order = %v, want %v", order, want)
	}
}

func TestCreationErrorHookOrder(t *testing.T) {
	// creationError runs type hooks then collection hooks (C5.2).
	collection := mnemonica.NewCollection()
	var order []string
	mustRegister(t, collection.RegisterHook(mnemonica.HookCreationError, recordHook(&order, "collection-error")))
	userT := mnemonica.Must(mnemonica.Define[User](collection, "User", func(u *User, name string) error {
		return errors.New("boom")
	}))
	mustRegister(t, userT.RegisterHook(mnemonica.HookCreationError, recordHook(&order, "type-error")))
	_, err := userT.New("ada")
	var errored *mnemonica.ErroredInstance
	if !errors.As(err, &errored) {
		t.Fatalf("New error = %v, want an *ErroredInstance", err)
	}
	want := []string{"type-error", "collection-error"}
	if !equalStrings(order, want) {
		t.Errorf("creationError hook order = %v, want %v", order, want)
	}
}

func recordHook(order *[]string, name string) mnemonica.HookFunc {
	hook := func(*mnemonica.HookData) error {
		*order = append(*order, name)
		return nil
	}
	return hook
}

func equalStrings(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for index := range got {
		if got[index] != want[index] {
			return false
		}
	}
	return true
}

func mustRegister(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("RegisterHook: %v", err)
	}
}

func TestPreCreationAbortsWithoutCreationError(t *testing.T) {
	collection := mnemonica.NewCollection()
	var order []string
	mustRegister(t, collection.RegisterHook(mnemonica.HookPreCreation, func(*mnemonica.HookData) error {
		order = append(order, "collection-pre")
		return nil
	}))
	abort := errors.New("abort")
	userT := mnemonica.Must(mnemonica.Define[User](collection, "User", func(u *User, name string) error {
		u.Name = name
		return nil
	}))
	mustRegister(t, userT.RegisterHook(mnemonica.HookPreCreation, func(*mnemonica.HookData) error {
		order = append(order, "type-pre")
		return abort
	}))
	mustRegister(t, userT.RegisterHook(mnemonica.HookCreationError, func(*mnemonica.HookData) error {
		order = append(order, "type-error")
		return nil
	}))
	_, err := userT.New("ada")
	if !errors.Is(err, abort) {
		t.Fatalf("New error = %v, want the preCreation abort error", err)
	}
	// Collection pre ran, type pre aborted, the handler never ran, and NO
	// creationError hooks fired for the abort.
	if !equalStrings(order, []string{"collection-pre", "type-pre"}) {
		t.Errorf("hook order = %v, want [collection-pre type-pre]", order)
	}
}

func TestPostCreationErrorPropagates(t *testing.T) {
	collection := mnemonica.NewCollection()
	userT := mnemonica.Must(mnemonica.Define[User](collection, "User", func(u *User, name string) error {
		u.Name = name
		return nil
	}))
	postErr := errors.New("post boom")
	mustRegister(t, userT.RegisterHook(mnemonica.HookPostCreation, func(*mnemonica.HookData) error {
		return postErr
	}))
	_, err := userT.New("ada")
	if !errors.Is(err, postErr) {
		t.Fatalf("New error = %v, want the postCreation hook error", err)
	}
}

func TestCreationErrorHookErrorPropagates(t *testing.T) {
	collection := mnemonica.NewCollection()
	handlerErr := errors.New("handler boom")
	hookErr := errors.New("hook boom")
	userT := mnemonica.Must(mnemonica.Define[User](collection, "User", func(u *User, name string) error {
		return handlerErr
	}))
	mustRegister(t, userT.RegisterHook(mnemonica.HookCreationError, func(*mnemonica.HookData) error {
		return hookErr
	}))
	_, err := userT.New("ada")
	if !errors.Is(err, hookErr) {
		t.Fatalf("New error = %v, want the creationError hook error", err)
	}
}

func TestHookDataPayload(t *testing.T) {
	// The hook payload mirrors the JS HookData: Type, Args, Parent
	// (existentInstance), Instance (inheritedInstance, nil in preCreation),
	// Err (creationError only).
	collection := mnemonica.NewCollection()
	var preData, postData, errorData *mnemonica.HookData
	userT := mnemonica.Must(mnemonica.Define[User](collection, "User", func(u *User, name string) error {
		u.Name = name
		return nil
	}))
	mustRegister(t, userT.RegisterHook(mnemonica.HookPreCreation, func(data *mnemonica.HookData) error {
		preData = data
		return nil
	}))
	mustRegister(t, userT.RegisterHook(mnemonica.HookPostCreation, func(data *mnemonica.HookData) error {
		postData = data
		return nil
	}))
	user, err := userT.New("ada")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if preData == nil || postData == nil {
		t.Fatal("hooks did not fire")
	}
	if preData.Type.Path() != "User" || preData.Args != "ada" {
		t.Errorf("preCreation data = (%q, %v), want (User, ada)", preData.Type.Path(), preData.Args)
	}
	if preData.Instance != nil {
		t.Errorf("preCreation Instance = %v, want nil (no instance exists yet)", preData.Instance)
	}
	if preData.Parent != nil {
		t.Errorf("preCreation Parent = %v, want nil for a root", preData.Parent)
	}
	if postData.Instance == nil {
		t.Error("postCreation Instance is nil, want the constructed instance")
	}
	if postData.Instance != user {
		t.Errorf("postCreation Instance = %v, want %p", postData.Instance, user)
	}
	if postData.Err != nil {
		t.Errorf("postCreation Err = %v, want nil", postData.Err)
	}

	failing := mnemonica.Must(mnemonica.Define[Widget](collection, "Widget", func(w *Widget, sku string) error {
		return errors.New("widget boom")
	}))
	mustRegister(t, failing.RegisterHook(mnemonica.HookCreationError, func(data *mnemonica.HookData) error {
		errorData = data
		return nil
	}))
	_, err = failing.New("sku")
	var errored *mnemonica.ErroredInstance
	if !errors.As(err, &errored) {
		t.Fatalf("setup: New error = %v, want an *ErroredInstance", err)
	}
	if errorData == nil {
		t.Fatal("creationError hook did not fire")
	}
	if errorData.Err == nil || errorData.Instance == nil {
		t.Errorf("creationError data = (Instance %v, Err %v), want both set", errorData.Instance, errorData.Err)
	}
}

func TestWrongHookKind(t *testing.T) {
	collection := mnemonica.NewCollection()
	err := collection.RegisterHook("nope", func(*mnemonica.HookData) error { return nil })
	if !errors.Is(err, mnemonica.ErrWrongHookType) {
		t.Errorf("collection.RegisterHook wrong kind error = %v, want ErrWrongHookType", err)
	}
	userT := mnemonica.Must(mnemonica.Define[User](collection, "User", func(u *User, name string) error {
		u.Name = name
		return nil
	}))
	err = userT.RegisterHook("nope", func(*mnemonica.HookData) error { return nil })
	if !errors.Is(err, mnemonica.ErrWrongHookType) {
		t.Errorf("type RegisterHook wrong kind error = %v, want ErrWrongHookType", err)
	}
}

func TestMissingHookCallback(t *testing.T) {
	collection := mnemonica.NewCollection()
	err := collection.RegisterHook(mnemonica.HookPreCreation, nil)
	if !errors.Is(err, mnemonica.ErrMissingHookCallback) {
		t.Errorf("collection.RegisterHook nil callback error = %v, want ErrMissingHookCallback", err)
	}
	userT := mnemonica.Must(mnemonica.Define[User](collection, "User", func(u *User, name string) error {
		u.Name = name
		return nil
	}))
	err = userT.RegisterHook(mnemonica.HookPostCreation, nil)
	if !errors.Is(err, mnemonica.ErrMissingHookCallback) {
		t.Errorf("type RegisterHook nil callback error = %v, want ErrMissingHookCallback", err)
	}
}

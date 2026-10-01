package mnemonica_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"mnemonica/mnemonica"
)

// ctxKey is the test key type for context values (the recommended
// unexported-type key pattern, so tests cannot collide with user keys).
type ctxKey struct{ name string }

// nilCtx is nil under a name: the P4 API deliberately accepts a nil ctx
// (it means "no context"), which staticcheck's blanket SA1012 cannot know.
var nilCtx context.Context

func TestNewCtxRecordsDetachedValues(t *testing.T) {
	fx := newFixture()
	source := context.WithValue(context.Background(), ctxKey{"request-id"}, "req-42")
	user, err := fx.userT.NewCtx(source, "ada")
	if err != nil {
		t.Fatalf("NewCtx: %v", err)
	}
	ctx := mnemonica.ContextOf(user)
	if value := ctx.Value(ctxKey{"request-id"}); value != "req-42" {
		t.Errorf("ContextOf Value = %v, want %q", value, "req-42")
	}
	if ctx.Err() != nil {
		t.Errorf("ContextOf Err() = %v, want nil", ctx.Err())
	}
	if ctx.Done() != nil {
		t.Error("ContextOf Done() != nil, want nil channel")
	}
	if _, ok := ctx.Deadline(); ok {
		t.Error("ContextOf Deadline() ok, want none")
	}
}

func TestContextOfSurvivesSourceCancellation(t *testing.T) {
	// The probe fact pinned as a test: canceling the request ctx must not
	// cancel, deadline, or clear work on the constructed data.
	fx := newFixture()
	source, cancel := context.WithCancel(context.WithValue(context.Background(), ctxKey{"request-id"}, "req-42"))
	user, err := fx.userT.NewCtx(source, "ada")
	if err != nil {
		t.Fatalf("NewCtx: %v", err)
	}
	cancel()

	ctx := mnemonica.ContextOf(user)
	if ctx.Err() != nil {
		t.Errorf("after source cancel, Err() = %v, want nil", ctx.Err())
	}
	if ctx.Done() != nil {
		t.Error("after source cancel, Done() != nil, want nil channel")
	}
	if _, ok := ctx.Deadline(); ok {
		t.Error("after source cancel, Deadline() ok, want none")
	}
	if value := ctx.Value(ctxKey{"request-id"}); value != "req-42" {
		t.Errorf("after source cancel, Value = %v, want %q (values survive)", value, "req-42")
	}

	// A worker may attach its own lifetime; the lineage ctx stays detached.
	worker, stop := context.WithCancel(ctx)
	stop()
	if worker.Err() == nil {
		t.Error("worker cancel did not fire, want it to")
	}
	if ctx.Err() != nil {
		t.Errorf("lineage ctx cancelled by the worker's stop, want detached: %v", ctx.Err())
	}
}

func TestContextOfNearestFirst(t *testing.T) {
	fx := newFixture()
	parent, err := fx.userT.NewCtx(context.WithValue(context.Background(), ctxKey{"id"}, "parent"), "ada")
	if err != nil {
		t.Fatalf("NewCtx: %v", err)
	}
	child, err := fx.adminT.FromCtx(
		context.WithValue(context.Background(), ctxKey{"id"}, "child"),
		parent, "root",
	)
	if err != nil {
		t.Fatalf("FromCtx: %v", err)
	}
	ctx := mnemonica.ContextOf(child)
	if value := ctx.Value(ctxKey{"id"}); value != "child" {
		t.Errorf("nearest-first Value = %v, want %q (child shadows parent)", value, "child")
	}

	// An intermediate node without a ctx is skipped on the walk.
	grand, err := fx.superT.From(child, 9)
	if err != nil {
		t.Fatalf("From: %v", err)
	}
	if value := mnemonica.ContextOf(grand).Value(ctxKey{"id"}); value != "child" {
		t.Errorf("Value across a ctx-less node = %v, want %q", value, "child")
	}

	// An unknown key walks the whole chain and yields nil.
	if value := ctx.Value(ctxKey{"nope"}); value != nil {
		t.Errorf("unknown key Value = %v, want nil", value)
	}
}

func TestSiblingIsolation(t *testing.T) {
	// Two children of one parent, each built under a different request:
	// neither sees the other's values.
	fx := newFixture()
	parent, err := fx.userT.New("ada")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	first, err := fx.adminT.FromCtx(context.WithValue(context.Background(), ctxKey{"branch"}, "first"), parent, "root")
	if err != nil {
		t.Fatalf("FromCtx first: %v", err)
	}
	second, err := fx.adminT.FromCtx(context.WithValue(context.Background(), ctxKey{"branch"}, "second"), parent, "operator")
	if err != nil {
		t.Fatalf("FromCtx second: %v", err)
	}
	if value := mnemonica.ContextOf(first).Value(ctxKey{"branch"}); value != "first" {
		t.Errorf("first branch Value = %v, want %q", value, "first")
	}
	if value := mnemonica.ContextOf(second).Value(ctxKey{"branch"}); value != "second" {
		t.Errorf("second branch Value = %v, want %q", value, "second")
	}
}

func TestContextOfWithoutCtxIsBackground(t *testing.T) {
	fx := newFixture()
	user, err := fx.userT.New("ada")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	admin, err := fx.adminT.From(user, "root")
	if err != nil {
		t.Fatalf("From: %v", err)
	}
	if ctx := mnemonica.ContextOf(admin); ctx != context.Background() {
		t.Errorf("ContextOf(ctx-less instance) = %v, want context.Background()", ctx)
	}
	if ctx := mnemonica.ContextOf(nil); ctx != context.Background() {
		t.Errorf("ContextOf(nil) = %v, want context.Background()", ctx)
	}
	var nilUser *User
	if ctx := mnemonica.ContextOf(nilUser); ctx != context.Background() {
		t.Errorf("ContextOf((*User)(nil)) = %v, want context.Background()", ctx)
	}
	if ctx := mnemonica.ContextOf(&User{}); ctx != context.Background() {
		t.Errorf("ContextOf(&User{}) = %v, want context.Background()", ctx)
	}
}

func TestNilCtxBehavesLikeNew(t *testing.T) {
	// Approved design: nil ctx records NOTHING — same as New/From, no
	// panic (context.WithoutCancel(nil) panics; the runtime guards first).
	fx := newFixture()
	user, err := fx.userT.NewCtx(nilCtx, "ada")
	if err != nil {
		t.Fatalf("NewCtx(nil): %v", err)
	}
	if ctx := mnemonica.ContextOf(user); ctx != context.Background() {
		t.Errorf("ContextOf after NewCtx(nil) = %v, want context.Background()", ctx)
	}
	admin, err := fx.adminT.FromCtx(nilCtx, user, "root")
	if err != nil {
		t.Fatalf("FromCtx(nil): %v", err)
	}
	if ctx := mnemonica.ContextOf(admin); ctx != context.Background() {
		t.Errorf("ContextOf after FromCtx(nil) = %v, want context.Background()", ctx)
	}
}

func TestHookDataCtx(t *testing.T) {
	// Every hook kind sees Ctx: the detached ctx when one was passed,
	// context.Background() when none was.
	collection := mnemonica.NewCollection()
	var preCtx, postCtx, errorCtx context.Context
	userT := mnemonica.Must(mnemonica.Define[User](collection, "User", func(u *User, name string) error {
		u.Name = name
		return nil
	}))
	if err := userT.RegisterHook(mnemonica.HookPreCreation, func(data *mnemonica.HookData) error {
		preCtx = data.Ctx
		return nil
	}); err != nil {
		t.Fatalf("RegisterHook pre: %v", err)
	}
	if err := userT.RegisterHook(mnemonica.HookPostCreation, func(data *mnemonica.HookData) error {
		postCtx = data.Ctx
		return nil
	}); err != nil {
		t.Fatalf("RegisterHook post: %v", err)
	}

	// Without a ctx: Background, non-nil.
	if _, err := userT.New("ada"); err != nil {
		t.Fatalf("New: %v", err)
	}
	if preCtx == nil || preCtx != context.Background() {
		t.Errorf("preCreation Ctx without ctx = %v, want context.Background()", preCtx)
	}
	if postCtx == nil || postCtx != context.Background() {
		t.Errorf("postCreation Ctx without ctx = %v, want context.Background()", postCtx)
	}

	// With a ctx: the detached ctx — values survive, cancellation does not.
	source, cancel := context.WithCancel(context.WithValue(context.Background(), ctxKey{"id"}, "hooked"))
	failing := mnemonica.Must(mnemonica.Define[Widget](collection, "Widget", func(w *Widget, sku string) error {
		return context.Canceled
	}))
	if err := failing.RegisterHook(mnemonica.HookCreationError, func(data *mnemonica.HookData) error {
		errorCtx = data.Ctx
		return nil
	}); err != nil {
		t.Fatalf("RegisterHook error: %v", err)
	}
	if _, err := userT.NewCtx(source, "grace"); err != nil {
		t.Fatalf("NewCtx: %v", err)
	}
	cancel()
	if preCtx == nil || preCtx.Err() != nil {
		t.Errorf("preCreation Ctx after source cancel: Err = %v, want nil", preCtx.Err())
	}
	if value := preCtx.Value(ctxKey{"id"}); value != "hooked" {
		t.Errorf("preCreation Ctx Value = %v, want %q", value, "hooked")
	}
	if _, err := failing.NewCtx(source, "sku"); err == nil {
		t.Fatal("failing NewCtx succeeded, want an error")
	}
	if errorCtx == nil || errorCtx.Err() != nil {
		t.Errorf("creationError Ctx after source cancel: Err = %v, want nil", errorCtx.Err())
	}
	if value := errorCtx.Value(ctxKey{"id"}); value != "hooked" {
		t.Errorf("creationError Ctx Value = %v, want %q", value, "hooked")
	}
}

func TestCloneAndForkInheritCtx(t *testing.T) {
	// The parent carries NO ctx, the subtype does: the only way Clone and
	// Fork results can see the value is inheriting the SOURCE's recorded
	// ctx (a lineage walk would find nothing).
	fx := newFixture()
	source := context.WithValue(context.Background(), ctxKey{"id"}, "inherited")
	admin := adminWithCtx(t, fx, source)

	clone, err := mnemonica.Clone(admin)
	if err != nil {
		t.Fatalf("Clone: %v", err)
	}
	if value := mnemonica.ContextOf(clone).Value(ctxKey{"id"}); value != "inherited" {
		t.Errorf("clone Value = %v, want %q (Clone inherits)", value, "inherited")
	}
	fork, err := mnemonica.Fork(admin, "operator")
	if err != nil {
		t.Fatalf("Fork: %v", err)
	}
	if value := mnemonica.ContextOf(fork).Value(ctxKey{"id"}); value != "inherited" {
		t.Errorf("fork Value = %v, want %q (Fork inherits)", value, "inherited")
	}
}

// adminWithCtx builds an Admin under a ctx-LESS parent and attaches ctx to
// the Admin itself, so ctx assertions on derived instances distinguish
// inheritance from a lineage walk.
func adminWithCtx(t *testing.T, fx fixture, ctx context.Context) *Admin {
	t.Helper()
	user, err := fx.userT.New("ada")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	admin, err := fx.adminT.FromCtx(ctx, user, "root")
	if err != nil {
		t.Fatalf("FromCtx: %v", err)
	}
	return admin
}

func TestForkCtxOverrides(t *testing.T) {
	fx := newFixture()
	source := context.WithValue(context.Background(), ctxKey{"id"}, "old")
	admin := adminWithCtx(t, fx, source)

	override := context.WithValue(context.Background(), ctxKey{"id"}, "new")
	forked, err := mnemonica.ForkCtx(admin, override, "operator")
	if err != nil {
		t.Fatalf("ForkCtx: %v", err)
	}
	if value := mnemonica.ContextOf(forked).Value(ctxKey{"id"}); value != "new" {
		t.Errorf("ForkCtx Value = %v, want %q (explicit ctx wins)", value, "new")
	}
	// A nil ctx on the explicit variant inherits, same as Fork.
	inherited, err := mnemonica.ForkCtx(admin, nilCtx, "operator")
	if err != nil {
		t.Fatalf("ForkCtx(nil): %v", err)
	}
	if value := mnemonica.ContextOf(inherited).Value(ctxKey{"id"}); value != "old" {
		t.Errorf("ForkCtx(nil) Value = %v, want %q (nil inherits)", value, "old")
	}
}

func TestForkOntoCtxOverrides(t *testing.T) {
	fx := newFixture()
	source := context.WithValue(context.Background(), ctxKey{"id"}, "old")
	admin := adminWithCtx(t, fx, source)
	other, err := fx.userT.New("grace")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	dag, err := mnemonica.ForkOntoCtx(admin, other, context.WithValue(context.Background(), ctxKey{"id"}, "new"), "operator")
	if err != nil {
		t.Fatalf("ForkOntoCtx: %v", err)
	}
	if value := mnemonica.ContextOf(dag).Value(ctxKey{"id"}); value != "new" {
		t.Errorf("ForkOntoCtx Value = %v, want %q (explicit ctx wins)", value, "new")
	}
	inherited, err := mnemonica.ForkOntoCtx(admin, other, nilCtx, "operator")
	if err != nil {
		t.Fatalf("ForkOntoCtx(nil): %v", err)
	}
	if value := mnemonica.ContextOf(inherited).Value(ctxKey{"id"}); value != "old" {
		t.Errorf("ForkOntoCtx(nil) Value = %v, want %q (nil inherits)", value, "old")
	}
}

func TestSiblingAndMergeInheritCtx(t *testing.T) {
	fx := newFixture()
	source := context.WithValue(context.Background(), ctxKey{"id"}, "inherited")
	admin := adminWithCtx(t, fx, source)

	sibling, err := mnemonica.Sibling(admin, "operator")
	if err != nil {
		t.Fatalf("Sibling: %v", err)
	}
	if value := mnemonica.ContextOf(sibling).Value(ctxKey{"id"}); value != "inherited" {
		t.Errorf("sibling Value = %v, want %q (inherits)", value, "inherited")
	}
	// SiblingCtx with an explicit ctx replaces; nil inherits.
	overridden, err := mnemonica.SiblingCtx(admin, context.WithValue(context.Background(), ctxKey{"id"}, "new"), "operator")
	if err != nil {
		t.Fatalf("SiblingCtx: %v", err)
	}
	if value := mnemonica.ContextOf(overridden).Value(ctxKey{"id"}); value != "new" {
		t.Errorf("SiblingCtx Value = %v, want %q", value, "new")
	}

	// Merge inherits A's ctx: the merged instance's own recorded ctx wins
	// over its new parent's (nearest-first) — proof it inherited rather
	// than merely sees the parent.
	user, err := fx.userT.NewCtx(context.WithValue(context.Background(), ctxKey{"id"}, "root-ctx"), "ada")
	if err != nil {
		t.Fatalf("NewCtx: %v", err)
	}
	merged, err := mnemonica.Merge(user, admin, "ada")
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}
	if value := mnemonica.ContextOf(merged).Value(ctxKey{"id"}); value != "root-ctx" {
		t.Errorf("merge Value = %v, want %q (A's ctx inherited, nearest-first)", value, "root-ctx")
	}
	mergedOverride, err := mnemonica.MergeCtx(user, admin, context.WithValue(context.Background(), ctxKey{"id"}, "new"), "ada")
	if err != nil {
		t.Fatalf("MergeCtx: %v", err)
	}
	if value := mnemonica.ContextOf(mergedOverride).Value(ctxKey{"id"}); value != "new" {
		t.Errorf("MergeCtx Value = %v, want %q", value, "new")
	}
}

func TestFromCtxTypedConstruction(t *testing.T) {
	// FromCtx constructs exactly like From, ctx aside.
	fx := newFixture()
	user, err := fx.userT.NewCtx(context.WithValue(context.Background(), ctxKey{"id"}, "req"), "ada")
	if err != nil {
		t.Fatalf("NewCtx: %v", err)
	}
	admin, err := fx.adminT.FromCtx(context.WithValue(context.Background(), ctxKey{"id"}, "child-req"), user, "root")
	if err != nil {
		t.Fatalf("FromCtx: %v", err)
	}
	if admin.Role != "root" || admin.Name != "ada" {
		t.Errorf("admin = (%q, %q), want (root, ada)", admin.Role, admin.Name)
	}
	if value := mnemonica.ContextOf(admin).Value(ctxKey{"id"}); value != "child-req" {
		t.Errorf("admin ctx Value = %v, want %q", value, "child-req")
	}
	// A cancelled source does not fail the construction either.
	source, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := fx.userT.NewCtx(source, "grace"); err != nil {
		t.Fatalf("NewCtx with pre-cancelled source: %v", err)
	}
	// FromCtx validates the parent exactly like From.
	if _, err := fx.adminT.FromCtx(context.Background(), nil, "root"); !errors.Is(err, mnemonica.ErrWrongModificationPattern) {
		t.Errorf("FromCtx(nil parent) error = %v, want ErrWrongModificationPattern", err)
	}
	if _, err := fx.userT.FromCtx(context.Background(), &mnemonica.Root{}, "ada"); !errors.Is(err, mnemonica.ErrWrongModificationPattern) {
		t.Errorf("FromCtx(root parent) error = %v, want ErrWrongModificationPattern", err)
	}
}

func TestContextOfValueDeadlineDetachedFromTimeout(t *testing.T) {
	// A deadline on the source is request-scoped: it must not become the
	// data's deadline.
	fx := newFixture()
	source, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	user, err := fx.userT.NewCtx(source, "ada")
	if err != nil {
		t.Fatalf("NewCtx: %v", err)
	}
	if _, ok := mnemonica.ContextOf(user).Deadline(); ok {
		t.Error("ContextOf reports a deadline, want none (the data outlives the request)")
	}
}

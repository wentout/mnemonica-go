package mnemonica_test

import (
	"context"
	"testing"

	"github.com/wentout/mnemonica-go/mnemonica"
)

var benchSink any

type benchCtxKey struct{}

var benchCtx = context.WithValue(context.Background(), benchCtxKey{}, "v")

func BenchmarkNew(b *testing.B) {
	fx := newFixture()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		user, err := fx.userT.New("ada")
		if err != nil {
			b.Fatal(err)
		}
		benchSink = user
	}
}

func BenchmarkNewCtx(b *testing.B) {
	// P4: attaching a ctx costs one extra allocation (the WithoutCancel
	// wrapper); everything else is identical to New.
	fx := newFixture()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		user, err := fx.userT.NewCtx(benchCtx, "ada")
		if err != nil {
			b.Fatal(err)
		}
		benchSink = user
	}
}

func BenchmarkFrom(b *testing.B) {
	fx := newFixture()
	user, err := fx.userT.New("ada")
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		admin, err := fx.adminT.From(user, "root")
		if err != nil {
			b.Fatal(err)
		}
		benchSink = admin
	}
}

func BenchmarkFromCtx(b *testing.B) {
	fx := newFixture()
	user, err := fx.userT.NewCtx(benchCtx, "ada")
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		admin, err := fx.adminT.FromCtx(benchCtx, user, "root")
		if err != nil {
			b.Fatal(err)
		}
		benchSink = admin
	}
}

func BenchmarkFromWire(b *testing.B) {
	// P5: the explicit wire func path — the wiring is a direct typed store
	// instead of the cached reflect.Set.
	fx := newFixture()
	wiredT, err := mnemonica.Sub[Admin](fx.userT, "BenchWired",
		func(a *Admin, role string) error {
			a.Role = role
			return nil
		},
		mnemonica.WithWireFunc(func(a *Admin, u *User) {
			a.User = u
		}),
	)
	if err != nil {
		b.Fatal(err)
	}
	user, err := fx.userT.New("ada")
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		admin, err := wiredT.From(user, "root")
		if err != nil {
			b.Fatal(err)
		}
		benchSink = admin
	}
}

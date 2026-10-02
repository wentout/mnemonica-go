// Package sample is the generator's happy-path fixture: direct and
// Must-wrapped calls, an aliased mnemonica import, an inferred type
// argument, fully explicit type arguments, a three-level chain, and decoy
// calls that must never be picked up.
package sample

import (
	"time"

	"github.com/wentout/mnemonica-go/mnemonica"
)

var collection = mnemonica.NewCollection()

type widgetArgs struct {
	name string
}

type Widget struct {
	mnemonica.Node
	Name string
}

var WidgetT, widgetErr = mnemonica.Define[Widget](collection, "Widget", func(a *Widget, args widgetArgs) error {
	a.Name = args.name
	return nil
})

type Gadget struct {
	mnemonica.Node
	*Widget
	Role string
}

var GadgetT = mnemonica.Must(mnemonica.Sub[Gadget](WidgetT, "Gadget", func(m *Gadget, role string) error {
	m.Role = role
	return nil
}))

type Job struct {
	mnemonica.Node
	*Gadget
	Scope int
}

var JobT = mnemonica.Must(mnemonica.Sub[Job](GadgetT, "Job", func(m *Job, scope int) error {
	m.Scope = scope
	return nil
}))

type Inferred struct {
	mnemonica.Node
	*Widget
	Note string
}

var InferredT, inferredErr = mnemonica.Sub(WidgetT, "Inferred", func(i *Inferred, note string) error {
	i.Note = note
	return nil
})

type Verbose struct {
	mnemonica.Node
	*Widget
	SKU string
}

var VerboseT = mnemonica.Must(mnemonica.Sub[Verbose, Widget, mnemonica.Root, widgetArgs, string](WidgetT, "Verbose", func(v *Verbose, sku string) error {
	v.SKU = sku
	return nil
}))

// Decoys: plain calls and same-named helpers must not confuse the scanner.

func Sub[T any](x T) T { return x }

var DecoyPlain = Sub(42)

type builder struct{}

func (builder) Must(x int) int { return x }

var tool = builder{}

var DecoyMust = tool.Must(1)

var plainT, plainErr = mnemonica.Define[Widget](collection, "Widget2", func(a *Widget, args widgetArgs) error {
	return nil
})

var OddT = mnemonica.Must(plainT, plainErr)

func Helper() int { return 1 }

func (Widget) Tag() string { return "" }

// Timed has an args type from another package: the generated signature
// must carry the package qualifier.
type Timed struct {
	mnemonica.Node
	*Widget
	Timeout time.Duration
}

var TimedT = mnemonica.Must(mnemonica.Sub[Timed](WidgetT, "Timed", func(t *Timed, timeout time.Duration) error {
	t.Timeout = timeout
	return nil
}))

var Two, Three = 1, 2

var NotCall = 42

// Shapes the generator's foreign-import walk must cover: pointer, slice,
// map, and struct args from another package, plus a same-package args
// type that needs no import.
type Pinned struct {
	mnemonica.Node
	*Widget
	At *time.Location
}

var PinnedT = mnemonica.Must(mnemonica.Sub[Pinned](WidgetT, "Pinned", func(p *Pinned, at *time.Location) error {
	p.At = at
	return nil
}))

type Batched struct {
	mnemonica.Node
	*Widget
	Durations []time.Duration
}

var BatchedT = mnemonica.Must(mnemonica.Sub[Batched](WidgetT, "Batched", func(b *Batched, durations []time.Duration) error {
	b.Durations = durations
	return nil
}))

type Indexed struct {
	mnemonica.Node
	*Widget
	Timeouts map[string]time.Duration
}

var IndexedT = mnemonica.Must(mnemonica.Sub[Indexed](WidgetT, "Indexed", func(i *Indexed, timeouts map[string]time.Duration) error {
	i.Timeouts = timeouts
	return nil
}))

type Bundled struct {
	mnemonica.Node
	*Widget
	Inner struct {
		Timeout time.Duration
	}
}

var BundledT = mnemonica.Must(mnemonica.Sub[Bundled](WidgetT, "Bundled", func(b *Bundled, inner struct {
	Timeout time.Duration
}) error {
	b.Inner = inner
	return nil
}))

type LocalArgs struct {
	mnemonica.Node
	*Widget
	X int
}

var LocalArgsT = mnemonica.Must(mnemonica.Sub[LocalArgs](WidgetT, "LocalArgs", func(l *LocalArgs, args widgetArgs) error {
	l.X = len(args.name)
	return nil
}))

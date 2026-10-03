# mnemonica for Go

**What `%w` did for errors, for your data.** Every instance remembers
what it is, where it came from, and what was done to it: a subtype is
constructed FROM an existing parent instance, keeps a live link to it,
and the chain back to the root is the data's own history. Identity is
nominal — checked by walking the lineage, like `errors.Is` walks a
wrapped error chain.

```go
type User struct {
	mnemonica.Node
	Name string
}
type Admin struct {
	mnemonica.Node
	*User        // embedded pointer to the parent — read-through by promotion
	Role string
}

var Users = mnemonica.NewCollection()
var UserT  = mnemonica.Must(mnemonica.Define[User](Users, "User",
	func(u *User, name string) error { u.Name = name; return nil }))
var AdminT = mnemonica.Must(mnemonica.Sub[Admin](UserT, "Admin",
	func(a *Admin, role string) error { a.Role = role; return nil }))

user, err  := UserT.New("ada")          // a root instance
admin, err := AdminT.From(user, "root") // constructed FROM the user

admin.Name              // "ada" — promoted from the embedded *User
mnemonica.Is[*User](admin)  // true — an Admin built from a User IS a User
```

## Install

```bash
go get github.com/wentout/mnemonica-go
```

```go
import "github.com/wentout/mnemonica-go/mnemonica"
```

The runtime package depends only on the standard library. The
generator and the analyzer live in a separate tools module and are only
needed at development time:

```bash
go install github.com/wentout/mnemonica-go/tools/cmd/mnemonica-gen@latest
go install github.com/wentout/mnemonica-go/tools/analysis/mnemonicavet@latest
```

## Quickstart

- `mnemonica.NewCollection()` creates an isolated type collection;
  `mnemonica.Default` is the package-level one. Options passed here are
  collection-level config defaults.
- `mnemonica.Define[T](collection, name, handler, options...)` declares a
  root type; `handler` is `func(*T, A) error` and runs on every
  construction with the single typed args value.
- `mnemonica.Sub[Child](parentT, name, handler, options...)` declares a
  subtype under a parent handle. `T` must embed `mnemonica.Node` and a
  `*Parent`; a value embed is rejected at define time.
- `mnemonica.Must(handle, err)` wraps Define/Sub for package-level vars,
  like `template.Must`.
- `UserT.New(args)` constructs a root; `AdminT.From(parent, args)`
  constructs a subtype FROM a parent instance. Both return
  `(*T, error)` — construction never panics; failures are sentinel
  errors checkable with `errors.Is`.

## Identity: Is and As

```go
mnemonica.Is[*User](admin)          // true — the lineage contains a *User
typed, ok := mnemonica.As[*User](admin) // the exact parent instance
```

Both walk the lineage exactly like `errors.Is`/`errors.As` walk a
wrapped error chain.

## The construction record: Props

```go
record, err := mnemonica.Props(admin)
record.Type.Path()   // "User.Admin"
record.Parent        // the user instance
record.Args          // "root"
record.Timestamp     // construction time
record.Collection    // the collection
record.Subtypes      // declared subtypes of Admin (sorted snapshot)
record.Stack         // set when submitStack config is on
record.Self          // admin itself
```

## Configuration

Per-type options; subtypes inherit the parent's resolved config, and a
root resolves against the collection defaults. All default to the JS
core's defaults.

```go
mnemonica.WithStrictChain(false)  // default true: only direct subtypes constructible
mnemonica.WithBlockErrors(false)  // default true: handler errors propagate raw instead of *ErroredInstance
mnemonica.WithSubmitStack(true)   // default false: record.Stack gets the construction stack
mnemonica.WithParentSnapshots(true) // default false: ancestor snapshot for the C2.4 test helper
```

With `blockErrors` on (default), a failing handler returns an
`*mnemonica.ErroredInstance` carrying the half-built instance, and
constructing from an errored lineage returns `mnemonica.ErrBlocked`.

## Hooks

```go
collection.RegisterHook(mnemonica.HookPreCreation, func(data *mnemonica.HookData) error {
	// data.Type, data.Args, data.Parent, data.Instance (nil pre-creation), data.Err, data.Ctx
	return nil // return an error to abort construction
})
typeT.RegisterHook(mnemonica.HookPostCreation, fn) // same on a type
```

Order (same as the JS core): **preCreation runs collection hooks first,
then type hooks; postCreation and creationError run type hooks first,
then collection hooks.** A preCreation error aborts construction with
no creationError hooks; hook errors propagate unwrapped.

## Context: bound to data, not to calls

Go binds context to calls; mnemonica keeps what must outlive the call.

```go
user, _ := UserT.NewCtx(requestCtx, "ada") // records the ctx's VALUES,
                                           // detached from cancellation
// ... the request returns and its ctx is cancelled ...

work := mnemonica.ContextOf(user) // never nil; Background() if nothing attached
work.Value(requestKey{})          // still readable — the request cannot
                                  // kill later work on the data
workerCtx, stop := context.WithCancel(work) // the worker owns its lifetime
```

`NewCtx`/`FromCtx` record `context.WithoutCancel(ctx)` — values without
the deadline or the Done channel, because the data outlives the request.
Reconstructing utils inherit the source's recorded context;
`ForkCtx`/`ForkOntoCtx`/`SiblingCtx`/`MergeCtx` attach an explicit one.

## Utils

`Extract` (flat map of fields along the lineage, nearest wins), `Pick`,
`Parent` (with a name or dotted path, contiguous-upwards), `Clone`
(re-runs the constructor from the same parent with the same args),
`Fork`, `ForkOnto` (a different parent — a DAG), `Sibling`, `Merge`
(b's fields fill what a doesn't define), `Parse` (one-level snapshot —
`Parent` is the instance), `ToJSON` (always valid, `{}` when empty),
`ConstructorSequence` / `CollectConstructors` (the naming-path spine),
`NewException` (an error carrying the instance).

## Lineage export

`mnemonica.ID(x)` returns a stable, process-unique instance id
(`"<hex8>:<counter>"`), assigned lazily — instances that are never
exported pay nothing. `mnemonica.DeepParse(x)` walks the lineage,
instance first, root last. `mnemonica.Lineage([]Instance{a, b}, opts...)`
exports a deduplicated graph — the FIRST DRAFT of the cross-language
shared format:

```json
{ "version": "1",
  "heads": ["62bd7d68:4", "62bd7d68:9"],
  "nodes": { "62bd7d68:4": { "type": {"collection": "fixture", "path": "User.Admin"},
                             "own": {"Role": "root"},
                             "parent": "62bd7d68:1" } } }
```

- `own` = the fields that level's own struct declares (embedded parents
  and the Node header never appear) — shadowed values stay with their
  writer; ancestors shared by several exports appear once, at any depth.
- A field holding another instance exports as `{ "$ref": "<id>" }` and
  the target joins `nodes` with its own chain.
- Non-JSON values (funcs, channels, NaN/±Inf, complex numbers, cycles)
  export as tagged placeholders — `{"$mnemonica": "unsupported",
  "kind": "func"}` — never an error or panic.
- Options: `mnemonica.WithArgs()` (sanitised construction args),
  `mnemonica.WithProps("timestamp")` (opt-in metadata).

The format is pinned by [lethe](https://github.com/mythographica/lethe) —
`lineage.schema.json` (JSON Schema 2020-12) plus one shared fixture and
its recipe README — the construction script every port (JS, Python)
reproduces byte-for-byte, with ids mapped 1:1 in first-encounter order
(ids are per-process and implementation-specific). Go consumers read the
contract through the embedded exports of `github.com/mythographica/lethe`
(`lethe.LineageSchema`, `lethe.LineageFixture`); the schema-validation
tests live in the tools module (`tools/lineageschema`) since the validator
dependency must stay out of the stdlib-only runtime.

## Generator: `mnemonica-gen`

The Go analogue of the JS tactica. Add `//go:generate mnemonica-gen` to
a package with `Define`/`Sub` calls; it emits `mnemonica_gen.go`:

- a typed wire func per subtype (`func wireGadgetToWidget(...)`) attached
  to the subtype's TypeDef in a generated `init()` — replacing the
  default reflect setter, and
- a construction method per subtype on the PARENT struct:
  `user.Admin(args)` and `user.AdminCtx(ctx, args)`, fully typed.

Output is deterministic, gofmt'd, refuses to overwrite existing methods,
and the Define/Sub call sites stay hand-written.

## OpenTelemetry

The `otel` module (`go get github.com/wentout/mnemonica-go/otel`) links
constructions to traces, in both directions:

```go
otelx.StampConstructions(types)   // once: every construction stamps the span
                                  // that was current at construction

// in the request:
user, _ := UserT.NewCtx(requestCtx, "ada")   // stamps the request span

// after the request, in a worker goroutine:
_, worker := otelx.StartLinkedSpan(context.Background(), tracer, "job", admin)
defer worker.End()        // LINKED to the construction span — the request's
                          // cancellation cannot kill work on the data

// on failure: the lineage graph rides the error span
otelx.RecordLineage(ctx, mnemonica.NewException(admin, err))
```

Four attribute names form the cross-language contract:
`mnemonica.instance.id`, `mnemonica.parent.id`,
`mnemonica.type.collection`, `mnemonica.type.path` — a Jaeger trace and a
lineage graph join on them. Stamping uses span attributes (queryable);
the error path attaches the lineage graph JSON as a span event named
`mnemonica.error`. The OTEL dependency never enters the runtime module.

## The write-through hazard

Go cannot shadow promoted writes: **`admin.Name = x` writes into the
shared parent** — the lineage makes it visible everywhere. This is the
one adapted rule of the port (the JS core shadows writes natively), and
it is guarded twice:

1. Static: the `mnemonicavet` analyzer (in the tools module) reports
   assignments through promoted fields of embedded lineage parents —
   `admin.Name = ...` — while leaving own fields, non-lineage embeds, and
   the sanctioned wiring shape alone. Run it with
   `go vet -vettool=$(which mnemonicavet) ./...`.
2. In tests: define types with `mnemonica.WithParentSnapshots(true)` and
   assert `mnemonicatest.AssertParentsUnchanged(t, instance)` — it
   compares the live lineage against the construction-time snapshot and
   names the mutated ancestor.

## Errors

Sentinels, checkable with `errors.Is`: `ErrAlreadyDeclared`,
`ErrWrongTypeDefinition`, `ErrWrongModificationPattern` (wrong parent,
root From, subtype New), `ErrNotAnInstance`, `ErrBlocked` (errored
lineage), `ErrWrongArgumentsUsed` (untyped bridge), `ErrWrongHookType`,
`ErrMissingHookCallback`.

## Status

P1–P6 complete: runtime (types, construction, lineage, nominal identity,
config, hooks, context), utils, the generator, the analyzer, the laws as
property tests, and benchmarks. See `docs/conformance.md` for the
item-by-item conformance matrix and `docs/performance.md` for measured
numbers.

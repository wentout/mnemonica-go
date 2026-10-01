# mnemonica for Go — the port plan

WHAT to build is in the shared contract:
`/code/mnemonica/plan/core-ports-contract.md` (C1–C9, the gates, the phases).
This file is HOW, in Go. Author: claude (viktor asked for claude's own design;
he does not write Go). Questions go to the room, not guesses.

## Design — what Go gives, what it refuses

Go gives read-through natively: a struct that embeds a POINTER to its parent
promotes the parent's fields and methods. Go refuses three things mnemonica
relies on: runtime type definitions, per-object write shadowing, and
inheritance-style nominal checks. The design keeps the semantics anyway:

| contract | Go |
|---|---|
| C2.3 read-through | embedded parent pointer — `admin.Name` promotes from `*User` |
| C2.4 write-local | NOT native: `admin.Name = "x"` writes into the shared parent. Guarded by the analyzer `mnemonicavet` (P5), a test helper that detects parent mutation (P2), and the docs. Recorded as `adapted` |
| C2.5 nominal identity | `mnemonica.Is[*User](admin)` / `mnemonica.As[*User](admin)` walk the lineage — exactly like `errors.Is` / `errors.As` walk a wrapped error chain |
| C1.2 define | generics at runtime (P1), code generation for the nice syntax (P5) |
| C1.5 handler swap | a `TypeDef` holds its handler as a func value read at every construction — swappable at runtime with a pre-compiled func (no eval in Go) |
| C6 async | N/A — Go has no async constructors; `context.Context` is P4 |

The pitch bridge to Go users: **"what `%w` did for errors, for your data."**

### The runtime (P1) — generics, no codegen

```go
type User struct {
	mnemonica.Node          // lineage header: parent, type, args, timestamp …
	Name string
}

type Admin struct {
	mnemonica.Node
	*User                   // the parent instance: read-through by promotion
	Role string
}

var Users = mnemonica.NewCollection()

var UserT = mnemonica.Define(Users, "User",
	func(u *User, name string) error { u.Name = name; return nil })

var AdminT = mnemonica.Sub(UserT, "Admin",
	func(a *Admin, role string) error { a.Role = role; return nil })

user, err := UserT.New("ada")
admin, err := AdminT.From(user, "root")   // constructed FROM user (C2.2)
admin.Name                                 // "ada" (C2.3)
mnemonica.Is[*User](admin)                 // true (C2.5)
```

- `mnemonica.Node` is the lineage header every type embeds; it implements an
  unexported-method interface so only real mnemonica instances satisfy
  `mnemonica.Instance`. It carries the C3 record (parent, type, args,
  timestamp, creator, collection, stack) — `mnemonica.Props(x)`.
- `Sub` must connect the child's embedded `*User` field to the parent. PROBE
  the cleanest way in P0: a setter the generator would emit, a generic
  constraint (`interface{ *Admin; setParent(*User) }`), or reflection once at
  `Sub` time (never per construction). Report the choice before building.
- Construction returns `(T, error)`; errors are sentinel values checkable by
  `errors.Is` (`mnemonica.ErrAlreadyDeclared`, …). Contract errors that a
  static type system makes impossible (a non-string type name) are `N/A`.
- `blockErrors` (C4.2): a failing handler returns an `*ErroredInstance`
  error that carries the half-built instance; `From` on an errored parent
  returns `ErrBlocked`. `creationError` hooks fire as in C5.
- `lookup(path)` (C1.4) returns `(mnemonica.TypeDef, bool)` — untyped by
  necessity; the typed handles are the `UserT`/`AdminT` values.
- Concurrency: a collection is read concurrently by many goroutines; define
  happens at init. Registry guarded by `sync.RWMutex`; every test runs with
  `-race`.

### Write-local guard (C2.4)

1. **P2 — test helper** `mnemonicatest.AssertParentsUnchanged(t, x)`:
   snapshots the lineage at construction (opt-in config) and reports a
   mutated ancestor.
2. **P5 — analyzer** `mnemonicavet` (`golang.org/x/tools/go/analysis`):
   reports assignments through a promoted field of an embedded LINEAGE
   parent (`admin.Name = …`). Idiomatic Go: a vet check, not a runtime cost.
   Runs inside `scripts/check` on the port's own examples.

### Code generation (P5)

`go generate` with `mnemonica-gen` (stdlib `go/ast` + `go/types` — the Go
analogue of tactica): from the `Define`/`Sub` calls it emits methods so a
subtype reads like mnemonica — `user.Admin("root")` — with full static types.

### Context (P4)

Go made context explicit (`context.Context`, bound to CALLS). mnemonica binds
context to DATA: P4 lets a constructor take a `ctx`, and `Node` keeps what
must outlive the call (trace/span context for later OTEL). The "goroutine
outlives its request" problem (`context.WithoutCancel`) is the use case. Plan
P4 in detail only after P3 — report a design first, STOP.

## Toolchain

- **Go is NOT installed on this machine yet** — viktor installs it (P0 is
  blocked until `go version` works). Target Go ≥ 1.23.
- Module path: placeholder `mnemonica` until viktor decides the real one
  (e.g. a GitHub path) — do not invent an organization.
- Tests: `go test -race -coverprofile`; `scripts/check` fails unless
  `go tool cover -func` reports **100.0%** total. Go has no branch coverage:
  every `if` gets a test for each side, checked by review.
- Static: `go vet`, **staticcheck** (zero findings), `gofmt -l` empty
  (gofmt uses tabs — same as JS core).
- No `any` in the public API except `Lookup`'s untyped result and the
  args record; generics everywhere else.
- Benchmarks: `go test -bench` with `-benchmem`; raw output to
  `/code/experiments/<date>-go-bench/`, summary in `docs/performance.md`.
- Dependencies: stdlib only for the runtime package; `golang.org/x/tools`
  only for the analyzer and generator modules.

## Layout

```
core-go/
  AGENTS.md
  go.mod
  scripts/check
  mnemonica/            node.go, collection.go, define.go, lookup.go,
                        props.go, hooks.go, errors.go, utils.go (+ _test.go)
  mnemonicatest/        test helpers (P2)
  cmd/mnemonica-gen/    generator (P5)
  analysis/mnemonicavet/ analyzer (P5)
  docs/conformance.md
  docs/performance.md
  README.md             for humans (P6)
  plans/PLAN.md         this file
```

## Phase checklist

- **P0** (after Go is installed) module, `scripts/check` green with one smoke
  test, `docs/conformance.md` all `open`; probe the `Sub` parent-wiring
  options → report → STOP.
- **P1** C1–C3, C8. STOP.
- **P2** C4, C5, the parent-mutation test helper. STOP.
- **P3** C7 (each util with its JS doc example ported as a test or an
  `Example` function). STOP.
- **P4** context design → STOP → build. STOP.
- **P5** generator + analyzer, each at 100%. STOP.
- **P6** laws as property tests (`testing/quick` or fuzz tests), benchmarks,
  README. STOP.

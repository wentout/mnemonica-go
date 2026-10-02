# Performance — mnemonica for Go

All numbers measured, not claimed (contract §4.5). Raw benchmark output
lives in the project's experiments area; this file holds the summary.

Machine: AMD Ryzen 5 7530U (uname: Linux 6.x x86_64), Linux amd64,
go1.27.1 (goenv), benchmarks without `-race`.

## Construction cost

| benchmark | ns/op | B/op | allocs/op |
|---|---:|---:|---:|
| `TypeDef.New` (root construction) | ~2970 | 208 | 2 |
| `TypeDef.NewCtx` (root, with a ctx) | ~3220 | 224 | 3 |
| `TypeDef.From` (subtype, incl. parent wiring) | ~3080 | 192 | 2 |
| `TypeDef.FromCtx` (subtype, with a ctx) | ~3160 | 208 | 3 |
| `TypeDef.From` with an explicit wire func | ~4870 | 192 | 2 |

Caveat, measured and verified by a dedicated probe: on this host
`time.Now()` alone costs ~3.0 µs/call (slow clocksource; ~92% of the
construction profile is the timestamp call). The mnemonica work itself —
allocation, lineage header, config resolution, parent validation, handler
invocation, and the cached `reflect.Value.Set` (or the explicit wire
func's direct store) that wires the embedded parent — is the remainder of
the numbers above. On hardware with a normal fast clocksource (typical
`time.Now` ~20-30 ns), construction is expected to land in the tens of
nanoseconds, as measured by the P0 wiring probe (~56 ns/op for the
reflect-setter variant, which is the variant `From` uses by default).

Typed args (P2) removed the variadic-args slice: construction allocates
exactly twice — the instance and nothing else (hook payloads are built
only when hooks are registered). P4's optional ctx field rides in the
existing allocation: no ctx → still 2 allocs; a ctx → +1 alloc, +16 B
(the `context.WithoutCancel` wrapper). L1's id counter added 8 bytes to
Node and pushed the root struct into the next size class (208 B/op where
it was 192; subtypes were already in a larger class and stay at 192). P5's
explicit wire func changes allocations not at all.

## Lineage ids and export (L1)

Instance ids (`mnemonica.ID`) carry their counter in Node itself (8
bytes, always present), assigned lazily from a global atomic on the first
ask — no assignment, no prefix generation, no allocation until then
(asserted by `TestIDLazy`: construction stays at 2 allocs/op). The id
string is formatted on demand, so nothing caches or pins the instance: an
id'd instance stays garbage-collectable (`TestIDCollectable` watches one
die after `runtime.GC`). The on-demand format measures ~123 ns/op, 48 B/op
(2 allocs: the formatted string plus the benchmark sink's boxing;
`BenchmarkIDAssign`). Exporting the small shared-fixture graph
(`BenchmarkLineage`, 4 deduplicated nodes with one `$ref`) costs ~1.1
µs/op, 1888 B/op, 18 allocs/op — reflection-heavy, pay-only-when-exported.

OTEL stamping (otel module, `BenchmarkStamping`): +8 allocs/op and
+360 B/op per construction over an unstamped one (the hook payload, two
id formats, and the attribute slice); the no-span path skips the id
formatting entirely. ns/op is within the slow-clocksource noise on this
host — allocs are the honest signal.

## Deep-chain read cost (depth 1 / 10 / 100) vs plain embedding

Reading the root's `Label` through a depth-N chain, against a plain
Go embedding of identical shape (no Node headers):

| depth | mnemonica chain | plain embedding |
|---:|---:|---:|
| 1 | ~44 ns | ~44 ns |
| 10 | ~51 ns | ~47 ns |
| 100 | ~285 ns | ~276 ns |

(16 B/op, 1 alloc/op in every read row — that is the benchmark sink
boxing the string, constant across rows.)

A promoted field read compiles each hop's offset into the access, but
the embedded pointer chain is still chased at runtime: depth-100 reads
cost ~6x depth-1, ~2.4 ns per level with a warm cache. The mnemonica
chain tracks the plain embedding within noise — the lineage headers add
nothing measurable to reads. Reads never call `time.Now`, so these
numbers carry no clocksource caveat.

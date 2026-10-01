# Performance — mnemonica for Go

All numbers measured, not claimed (contract §4.5). Raw benchmark output
lives in the project's experiments area; this file holds the summary.

Machine: AMD Ryzen 5 7530U (uname: Linux 6.x x86_64), Linux amd64,
go1.27.1 (goenv), benchmarks without `-race`.

## Construction cost

| benchmark | ns/op | B/op | allocs/op |
|---|---:|---:|---:|
| `TypeDef.New` (root construction) | ~4810 | 192 | 2 |
| `TypeDef.NewCtx` (root, with a ctx) | ~4840 | 208 | 3 |
| `TypeDef.From` (subtype, incl. parent wiring) | ~5020 | 192 | 2 |
| `TypeDef.FromCtx` (subtype, with a ctx) | ~4980 | 208 | 3 |
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
(the `context.WithoutCancel` wrapper). P5's explicit wire func changes
allocations not at all (192 B, 2 allocs, same as the reflect setter).

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

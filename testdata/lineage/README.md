# Shared lineage fixture — the cross-language graph

This directory holds the FIRST DRAFT of the shared lineage export format
(`github.com/wentout/mnemonica-go`, runtime package `mnemonica`; the
schema is `lineage.schema.json` at the repo root). The JS and Python ports
reproduce `fixture.json` byte-for-byte from the construction script below;
`docs/` in each port points back here.

## The construction script (language-neutral)

Types (a `User` root; `Admin` is a subtype of `User`; `SuperAdmin` is a
subtype of `Admin`):

- `User` declares one own field: `Name string`.
- `Admin` embeds the parent (`*User`) and declares own fields
  `Role string` and `Attached *User` (an own field holding another
  mnemonica instance — it exports as `{ "$ref": <id> }`).
- `SuperAdmin` embeds `*Admin` and declares one own field: `Level int`.

Collection: one collection named `fixture` (the lineage export records
`type.collection` and `type.path`; `type.path` is the dotted path,
`type.collection` is the collection's name).

Constructions:

1. `u  = User.New("ada")`
2. `a1 = Admin.From(u, "root")`
3. `a2 = Admin.From(u, "operator")`  — a sibling of `a1` (same parent)
4. `s  = SuperAdmin.From(a1, 7)`
5. `a1.Attached = u`                 — the `$ref` field
6. export `Lineage([s, a2])`

## The exported graph

- `heads` = the ids of `s` and `a2`, in argument order.
- `nodes` contains exactly four entries: `s`, `a1`, `a2`, and `u`. The
  user `u` is shared by three nodes and appears ONCE (dedup at any depth).
- `own` holds each level's OWN declared fields: `s` has `{"Level": 7}`;
  `a1` has `Role` and `Attached` (the `$ref`); `a2` has `Role` and
  `Attached: null` (nil pointer fields export as null); `u` has `Name`.
  Embedded parents and the Node header never appear.
- `parent` is the parent's id, `null` at the root.

## The id convention (READ THIS — cross-language contract)

Instance ids are **implementation-specific**: a per-process random prefix
plus a process-local counter (in Go, `"<hex8>:<counter>"`, e.g.
`62bd7d68:7`). They are stable within one process and never reused; they
are NOT comparable across processes or languages.

Therefore `fixture.json`'s ids are **semantic placeholders**, and the
byte-for-byte comparison maps ids **1:1 in first-encounter order**:

1. the heads, in argument order;
2. then the nodes, in the order each node is first encountered during
   export — a node itself first, then its own fields in field-declaration
   order (a `$ref` target is exported at first encounter), then its
   parent, recursively, depth first.

For this fixture that order is `s`, `a1`, `u`, `a2`: `s` first; `s` has
no instance fields of its own, so its parent `a1` is next; `a1`'s
`Attached` field exports `u`; `a2` is the second head. Every port maps its
i-th encountered id to the fixture's i-th id in that order (`s`, `a1`,
`u`, `a2`) before comparing bytes. The JSON is the canonical form: no
whitespace, object keys sorted, exactly as `json.Marshal` of the decoded
graph produces.

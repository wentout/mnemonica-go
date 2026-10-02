# mnemonica otel — the OTEL link

The OpenTelemetry bridge for
[mnemonica-go](https://github.com/wentout/mnemonica-go). The OTEL
dependency lives in THIS module — the runtime module stays stdlib-only
and the tools module stays tooling-only.

```bash
go get github.com/wentout/mnemonica-go/otel
```

## The attribute contract

Four attribute names, shared cross-language with the JS and Python ports;
a Jaeger trace and a lineage graph join on them:

| constant | value |
|---|---|
| `otelx.AttrInstanceID` | `mnemonica.instance.id` |
| `otelx.AttrParentID` | `mnemonica.parent.id` |
| `otelx.AttrTypeCollection` | `mnemonica.type.collection` |
| `otelx.AttrTypePath` | `mnemonica.type.path` |

The type identity is two fields — collection name plus dotted path —
matching the lineage graph's `type` shape.

## What it does

- `otelx.StampConstructions(collection)` registers hooks so every
  construction — successful or errored — stamps the four attributes on
  the span that was current at construction (the ctx of `NewCtx`/`FromCtx`,
  or one inherited through Clone/Fork). Without a recorded span it is a
  cheap no-op: ids are formatted only when a real span will receive them.
  Stamping uses span **attributes** (not events): they are the queryable
  surface — "which request spans constructed a `User.Admin`" is a tag
  search — and `mnemonica.instance.id` is the join key downstream.
- `otelx.StartLinkedSpan(ctx, tracer, name, x, opts...)` starts a span
  for work on `x` that outlives the request, **linked** via `trace.Link`
  to the span current at `x`'s construction — the recorded (detached) ctx
  carries the SpanContext even after the request is cancelled (probe:
  `SpanContextFromContext(mnemonica.ContextOf(x))` matches after the
  request span ends). Without a recorded span context: a plain span.
- `otelx.RecordLineage(ctx, err)` attaches the carried instance's lineage
  graph JSON to the span current in `ctx` (the error-handling span) as a
  span event named `mnemonica.error`, with the graph in one
  `mnemonica.lineage.graph` attribute (plus the instance id). Carriers:
  `*mnemonica.ErroredInstance` and `*mnemonica.Exception`. A span event
  renders beside the error in every trace viewer; a log record would need
  a separate logs pipeline — probe-evidenced decision.

```go
otelx.StampConstructions(types)              // once, at setup

// in the request, under the request span:
user, _ := UserT.NewCtx(requestCtx, "ada")
admin, _ := AdminT.FromCtx(requestCtx, user, "root")

// after the request, in a worker goroutine:
_, worker := otelx.StartLinkedSpan(context.Background(), tracer, "job", admin)
defer worker.End()

// on failure:
otelx.RecordLineage(trace.ContextWithSpan(context.Background(), worker),
    mnemonica.NewException(admin, err))
```

Measured cost of stamping: +8 allocs / +360 B per construction on this
host (see docs/performance.md in the repo root); the no-span path is
allocation-free beyond the hook machinery that already exists.

// Package otelx is the OTEL link for the mnemonica Go port (L2): it
// stamps constructions onto the request's trace, links post-request work
// back to the construction span, and attaches lineage graphs on the error
// path. The OTEL dependency lives HERE, in the otel module — never in
// the stdlib-only runtime. Design decisions (attributes over events for
// stamping, an event with one JSON attribute for the error path, the
// SpanContext-through-ContextOf link) are probe-evidenced; see the probe
// in the project's experiments area and this module's README.
package otelx

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/wentout/mnemonica-go/mnemonica"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// The cross-language attribute contract (L2): the names are shared with
// the JS and Python ports, so a Jaeger trace and a lineage graph join on
// them. The type identity is two fields, matching the lineage graph's
// `type` shape.
const (
	AttrInstanceID     = "mnemonica.instance.id"
	AttrParentID       = "mnemonica.parent.id"
	AttrTypeCollection = "mnemonica.type.collection"
	AttrTypePath       = "mnemonica.type.path"
	// AttrLineageGraph carries the exported lineage graph JSON on the
	// error path. One attribute, no count pressure (SDK default limit is
	// 128; value length is unlimited).
	AttrLineageGraph = "mnemonica.lineage.graph"
	// EventError is the span event carrying the lineage graph.
	EventError = "mnemonica.error"
)

// StampConstructions registers hooks on the collection so every
// construction — successful or errored — stamps the four contract
// attributes on the span that was current at construction (the ctx passed
// to NewCtx/FromCtx, or one inherited through Clone/Fork and friends).
// Without a recorded span context the hooks are cheap no-ops: the id
// strings are formatted only when a real span will receive them.
func StampConstructions(collection *mnemonica.Collection) {
	// RegisterHook errors only on an unknown kind or a nil hook; both are
	// impossible with these constants, so the results are not checked.
	_ = collection.RegisterHook(mnemonica.HookPostCreation, stampConstruction)
	_ = collection.RegisterHook(mnemonica.HookCreationError, stampConstruction)
}

// stampConstruction stamps data.Instance's construction onto its recorded
// span. The recorded ctx carries the construction's span context — valid
// even after the request span ends; recording on an ended or noop span
// is a silent no-op either way.
func stampConstruction(data *mnemonica.HookData) error {
	span := trace.SpanFromContext(data.Ctx)
	if !span.SpanContext().IsValid() {
		return nil // no span to stamp: skip before formatting any ids
	}
	attributes := []attribute.KeyValue{
		attribute.String(AttrInstanceID, mnemonica.ID(data.Instance)),
		attribute.String(AttrTypeCollection, data.Type.Collection().Name()),
		attribute.String(AttrTypePath, data.Type.Path()),
	}
	if data.Parent != nil {
		attributes = append(attributes, attribute.String(AttrParentID, mnemonica.ID(data.Parent)))
	}
	span.SetAttributes(attributes...)
	return nil
}

// StartLinkedSpan starts a span for work on x that outlives the request:
// when x carries a recorded span context (see StampConstructions and
// NewCtx/FromCtx), the new span is LINKED to the construction's span via
// trace.Link — the worker shows up attached to the request in the trace
// even though the request is long gone. Without a recorded span context
// it starts a plain span. The caller owns the span's lifetime (End it).
func StartLinkedSpan(ctx context.Context, tracer trace.Tracer, name string, x mnemonica.Instance, options ...trace.SpanStartOption) (context.Context, trace.Span) {
	recorded := trace.SpanContextFromContext(mnemonica.ContextOf(x))
	if recorded.IsValid() {
		options = append(options, trace.WithLinks(trace.Link{SpanContext: recorded}))
	}
	result, span := tracer.Start(ctx, name, options...)
	return result, span
}

// RecordLineage attaches the carried instance's lineage graph to the span
// current in ctx (the error-handling span) as a span event with the graph
// JSON in one attribute — rendering beside the error in every trace
// viewer, with no separate logs pipeline required. Supported carriers are
// *mnemonica.ErroredInstance and *mnemonica.Exception; anything else
// fails with ErrNotAnInstance.
func RecordLineage(ctx context.Context, err error) error {
	instance, ok := instanceOf(err)
	if !ok {
		return fmt.Errorf("%w: %T carries no instance", mnemonica.ErrNotAnInstance, err)
	}
	graph, graphErr := mnemonica.Lineage([]mnemonica.Instance{instance})
	if graphErr != nil {
		return graphErr
	}
	// The lineage encoder's output is JSON-safe by construction (every
	// value passes the placeholder machinery), so Marshal cannot fail.
	raw, _ := json.Marshal(graph)
	span := trace.SpanFromContext(ctx)
	span.AddEvent(EventError, trace.WithAttributes(
		attribute.String(AttrInstanceID, mnemonica.ID(instance)),
		attribute.String(AttrLineageGraph, string(raw)),
	))
	return nil
}

// instanceOf extracts the carried instance from the supported error types.
// The runtime's two carriers expose it differently (a field vs a method),
// so a type switch, not an interface.
func instanceOf(err error) (mnemonica.Instance, bool) {
	switch typed := err.(type) {
	case *mnemonica.ErroredInstance:
		return typed.Instance, true
	case *mnemonica.Exception:
		return typed.Instance(), true
	default:
		return nil, false
	}
}

package mnemonica

import (
	"context"
	"time"
)

// This file is the P4 context binding: Go binds context to CALLS
// (context.Context), mnemonica binds it to DATA. A constructor may take a
// ctx through NewCtx/FromCtx; the Node keeps what must outlive the call —
// the VALUES, never the cancellation or the deadline, because the data
// outlives the request that carried them. The design, with probes, is in
// the P4 design report; the short form: context.WithoutCancel keeps values
// and drops Done/Deadline/Err, exactly the detachment this needs.

// detachedContext detaches a construction ctx for recording. A nil ctx
// records NOTHING — no panic (context.WithoutCancel(nil) panics, which is
// why the guard lives here), no allocation, no marker.
func detachedContext(ctx context.Context) context.Context {
	if ctx == nil {
		return nil
	}
	result := context.WithoutCancel(ctx)
	return result
}

// noCtx is a nil context under a name. The P4 API contract makes nil
// meaningful — "no context attached" for NewCtx/FromCtx, "inherit the
// source's" for the Ctx variants of the reconstructing utils — and
// staticcheck's blanket SA1012 ("do not pass a nil Context") cannot see a
// deliberate contract, so the nil gets a name and this comment instead of
// repeated lint suppressions.
var noCtx context.Context

// hookContext is what HookData carries: the detached construction ctx, or
// Background when none was attached — hooks must be able to rely on a
// non-nil ctx without checking.
func hookContext(recorded context.Context) context.Context {
	if recorded == nil {
		return context.Background()
	}
	result := recorded
	return result
}

// ContextOf returns a context derived from x's lineage, for work that
// outlives the request that constructed the data. It is NEVER nil: when no
// ctx was ever attached to x or any ancestor, it is context.Background().
//
// The result is detached by construction: Value sees every ancestor's
// attached ctx, nearest first (a nearer value shadows an ancestor's, the
// same rule as field read-through); Done returns nil, Err returns nil, and
// Deadline reports none — canceling the original request ctx cannot kill
// later work on the data. A worker wraps it in its own WithCancel when it
// needs a lifetime.
func ContextOf(x Instance) context.Context {
	node := nodeOf(x)
	if node == nil {
		return context.Background()
	}
	for cursor := node; cursor != nil; cursor = nodeOf(cursor.parent) {
		if cursor.ctx != nil {
			result := lineageContext{start: node}
			return result
		}
	}
	return context.Background()
}

// lineageContext implements context.Context over a node chain: values only,
// nearest first, nothing cancellable.
type lineageContext struct {
	start *Node
}

func (lineageContext) Deadline() (time.Time, bool) {
	return time.Time{}, false
}

func (lineageContext) Done() <-chan struct{} {
	return nil
}

func (lineageContext) Err() error {
	return nil
}

func (c lineageContext) Value(key any) any {
	for cursor := c.start; cursor != nil; cursor = nodeOf(cursor.parent) {
		if cursor.ctx == nil {
			continue
		}
		if value := cursor.ctx.Value(key); value != nil {
			return value
		}
	}
	return nil
}

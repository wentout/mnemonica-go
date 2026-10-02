package otelx_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/wentout/mnemonica-go/mnemonica"
	"github.com/wentout/mnemonica-go/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

// Fixture: the canonical User → Admin graph in a named, stamped collection.

type User struct {
	mnemonica.Node
	Name string
}

type Admin struct {
	mnemonica.Node
	*User
	Role string
}

type otelFixture struct {
	collection *mnemonica.Collection
	userT      *mnemonica.TypeDef[User, mnemonica.Root, string]
	adminT     *mnemonica.TypeDef[Admin, User, string]
}

func newOTelFixture(tb testing.TB) otelFixture {
	tb.Helper()
	return newOTelFixtureStamped(tb, true)
}

func newOTelFixtureStamped(tb testing.TB, stamp bool) otelFixture {
	tb.Helper()
	collection := mnemonica.NewCollection(mnemonica.WithCollectionName("otel"))
	userT, err := mnemonica.Define[User](collection, "User", func(u *User, name string) error {
		u.Name = name
		return nil
	})
	if err != nil {
		tb.Fatalf("Define: %v", err)
	}
	adminT, err := mnemonica.Sub[Admin](userT, "Admin", func(a *Admin, role string) error {
		a.Role = role
		return nil
	})
	if err != nil {
		tb.Fatalf("Sub: %v", err)
	}
	if stamp {
		otelx.StampConstructions(collection)
	}
	return otelFixture{collection: collection, userT: userT, adminT: adminT}
}

func newRecorder(t *testing.T) (*sdktrace.TracerProvider, *tracetest.SpanRecorder, trace.Tracer) {
	t.Helper()
	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	t.Cleanup(func() {
		_ = provider.Shutdown(context.Background())
	})
	return provider, recorder, provider.Tracer("otelx-test")
}

func attributesOf(span sdktrace.ReadOnlySpan) map[string]string {
	result := make(map[string]string, len(span.Attributes()))
	for _, attr := range span.Attributes() {
		result[string(attr.Key)] = attr.Value.AsString()
	}
	return result
}

func endedSpan(t *testing.T, recorder *tracetest.SpanRecorder, name string) sdktrace.ReadOnlySpan {
	t.Helper()
	for _, span := range recorder.Ended() {
		if span.Name() == name {
			return span
		}
	}
	t.Fatalf("span %q not found in ended spans", name)
	return nil
}

func TestStampConstructionsOnRequestSpan(t *testing.T) {
	fx := newOTelFixture(t)
	_, recorder, tracer := newRecorder(t)

	requestCtx, requestSpan := tracer.Start(context.Background(), "request")
	user, err := fx.userT.NewCtx(requestCtx, "ada")
	if err != nil {
		t.Fatalf("NewCtx: %v", err)
	}
	admin, err := fx.adminT.FromCtx(requestCtx, user, "root")
	if err != nil {
		t.Fatalf("FromCtx: %v", err)
	}
	requestSpan.End()

	request := attributesOf(endedSpan(t, recorder, "request"))
	if request[otelx.AttrInstanceID] != mnemonica.ID(admin) {
		t.Errorf("instance id = %q, want %q", request[otelx.AttrInstanceID], mnemonica.ID(admin))
	}
	if request[otelx.AttrParentID] != mnemonica.ID(user) {
		t.Errorf("parent id = %q, want %q", request[otelx.AttrParentID], mnemonica.ID(user))
	}
	if request[otelx.AttrTypeCollection] != "otel" || request[otelx.AttrTypePath] != "User.Admin" {
		t.Errorf("type attributes = %q/%q, want otel/User.Admin",
			request[otelx.AttrTypeCollection], request[otelx.AttrTypePath])
	}
}

func TestStampConstructionsRootHasNoParentAttribute(t *testing.T) {
	fx := newOTelFixture(t)
	_, recorder, tracer := newRecorder(t)

	requestCtx, requestSpan := tracer.Start(context.Background(), "request")
	user, err := fx.userT.NewCtx(requestCtx, "ada")
	if err != nil {
		t.Fatalf("NewCtx: %v", err)
	}
	requestSpan.End()

	request := attributesOf(endedSpan(t, recorder, "request"))
	if request[otelx.AttrInstanceID] != mnemonica.ID(user) || request[otelx.AttrTypePath] != "User" {
		t.Errorf("root attributes = %v", request)
	}
	if _, ok := request[otelx.AttrParentID]; ok {
		t.Error("a root must not carry a parent id attribute")
	}
}

func TestStampConstructionsWithoutSpan(t *testing.T) {
	fx := newOTelFixture(t)
	_, recorder, _ := newRecorder(t)

	// No ctx at all: the hooks must not panic and must not fabricate spans.
	user, err := fx.userT.New("ada")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	// A ctx without a span: same cheap no-op.
	other, err := fx.userT.NewCtx(context.Background(), "grace")
	if err != nil {
		t.Fatalf("NewCtx without span: %v", err)
	}
	if user.Name != "ada" || other.Name != "grace" {
		t.Error("constructions themselves broke")
	}
	if spans := recorder.Ended(); len(spans) != 0 {
		t.Errorf("noop stamping started %d spans, want 0", len(spans))
	}
}

func TestStampConstructionsErrored(t *testing.T) {
	fx := newOTelFixture(t)
	_, recorder, tracer := newRecorder(t)
	boom := errors.New("boom")
	failingT, err := mnemonica.Sub[Admin](fx.userT, "Failing",
		func(a *Admin, role string) error { return boom })
	if err != nil {
		t.Fatalf("Sub: %v", err)
	}

	requestCtx, requestSpan := tracer.Start(context.Background(), "request")
	user, err := fx.userT.NewCtx(requestCtx, "ada")
	if err != nil {
		t.Fatalf("NewCtx: %v", err)
	}
	_, constructErr := failingT.FromCtx(requestCtx, user, "root")
	var errored *mnemonica.ErroredInstance
	if !errors.As(constructErr, &errored) {
		t.Fatalf("construct error = %v, want an *ErroredInstance", constructErr)
	}
	requestSpan.End()

	request := attributesOf(endedSpan(t, recorder, "request"))
	if request[otelx.AttrInstanceID] != mnemonica.ID(errored.Instance) {
		t.Errorf("errored instance id = %q, want %q",
			request[otelx.AttrInstanceID], mnemonica.ID(errored.Instance))
	}
	if request[otelx.AttrTypePath] != "User.Failing" {
		t.Errorf("errored type path = %q", request[otelx.AttrTypePath])
	}
}

func TestStartLinkedSpan(t *testing.T) {
	fx := newOTelFixture(t)
	_, recorder, tracer := newRecorder(t)

	requestCtx, requestSpan := tracer.Start(context.Background(), "request")
	admin, err := fx.adminT.FromCtx(requestCtx, lawUser(t, fx), "root")
	if err != nil {
		t.Fatalf("FromCtx: %v", err)
	}
	requestSpan.End()
	requestContext := requestSpan.SpanContext()

	// Work that outlives the request: the worker links back to the
	// construction span through the recorded (detached) ctx.
	workerCtx, worker := otelx.StartLinkedSpan(context.Background(), tracer, "worker", admin)
	_ = workerCtx
	worker.End()

	workerSpan := endedSpan(t, recorder, "worker")
	if len(workerSpan.Links()) != 1 {
		t.Fatalf("worker links = %d, want 1", len(workerSpan.Links()))
	}
	linked := workerSpan.Links()[0].SpanContext
	if linked.TraceID() != requestContext.TraceID() || linked.SpanID() != requestContext.SpanID() {
		t.Errorf("worker link = %s/%s, want the request span %s/%s",
			linked.TraceID(), linked.SpanID(), requestContext.TraceID(), requestContext.SpanID())
	}

	// Without a recorded span context: a plain span, no links.
	plain, err := fx.userT.New("plain")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	_, plainWorker := otelx.StartLinkedSpan(context.Background(), tracer, "plain-worker", plain)
	plainWorker.End()
	if links := endedSpan(t, recorder, "plain-worker").Links(); len(links) != 0 {
		t.Errorf("plain worker links = %d, want 0", len(links))
	}
}

func lawUser(t *testing.T, fx otelFixture) *User {
	t.Helper()
	user, err := fx.userT.New("ada")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return user
}

func lineageEventOf(t *testing.T, span sdktrace.ReadOnlySpan) map[string]string {
	t.Helper()
	for _, event := range span.Events() {
		if event.Name == otelx.EventError {
			attributes := make(map[string]string, len(event.Attributes))
			for _, attr := range event.Attributes {
				attributes[string(attr.Key)] = attr.Value.AsString()
			}
			return attributes
		}
	}
	t.Fatal("mnemonica.error event not found")
	return nil
}

func assertGraphParses(t *testing.T, raw string) map[string]any {
	t.Helper()
	var graph map[string]any
	if err := json.Unmarshal([]byte(raw), &graph); err != nil {
		t.Fatalf("lineage graph does not parse: %v", err)
	}
	if graph["version"] != "1" {
		t.Errorf("graph version = %v, want 1", graph["version"])
	}
	return graph
}

func TestRecordLineageCarriers(t *testing.T) {
	fx := newOTelFixture(t)
	_, recorder, tracer := newRecorder(t)

	requestCtx, requestSpan := tracer.Start(context.Background(), "request")
	user, err := fx.userT.NewCtx(requestCtx, "ada")
	if err != nil {
		t.Fatalf("NewCtx: %v", err)
	}
	requestSpan.End()

	boom := errors.New("boom")
	failingT, err := mnemonica.Sub[Admin](fx.userT, "Failing",
		func(a *Admin, role string) error { return boom })
	if err != nil {
		t.Fatalf("Sub: %v", err)
	}
	_, erroredErr := failingT.From(lawUser(t, fx), "root")
	var errored *mnemonica.ErroredInstance
	if !errors.As(erroredErr, &errored) {
		t.Fatalf("construct error = %v, want an *ErroredInstance", erroredErr)
	}

	handleCtx, handleSpan := tracer.Start(context.Background(), "handle-error")
	if err := otelx.RecordLineage(handleCtx, erroredErr); err != nil {
		t.Fatalf("RecordLineage: %v", err)
	}
	handleSpan.End()
	event := lineageEventOf(t, endedSpan(t, recorder, "handle-error"))
	if event[otelx.AttrInstanceID] != mnemonica.ID(errored.Instance) {
		t.Errorf("event instance id = %q, want %q",
			event[otelx.AttrInstanceID], mnemonica.ID(errored.Instance))
	}
	graph := assertGraphParses(t, event[otelx.AttrLineageGraph])
	if _, ok := graph["nodes"].(map[string]any)[mnemonica.ID(errored.Instance)]; !ok {
		t.Error("the errored instance is not a node of its own lineage graph")
	}

	// Carrier 2: *mnemonica.Exception.
	exception := mnemonica.NewException(user, boom, "extra")
	handleCtx2, handleSpan2 := tracer.Start(context.Background(), "handle-exception")
	if err := otelx.RecordLineage(handleCtx2, exception); err != nil {
		t.Fatalf("RecordLineage: %v", err)
	}
	handleSpan2.End()
	event2 := lineageEventOf(t, endedSpan(t, recorder, "handle-exception"))
	if event2[otelx.AttrInstanceID] != mnemonica.ID(user) {
		t.Errorf("exception event instance id = %q, want %q",
			event2[otelx.AttrInstanceID], mnemonica.ID(user))
	}
	assertGraphParses(t, event2[otelx.AttrLineageGraph])

	// Unsupported carrier: ErrNotAnInstance, and no event recorded.
	if err := otelx.RecordLineage(context.Background(), boom); !errors.Is(err, mnemonica.ErrNotAnInstance) {
		t.Errorf("unsupported carrier error = %v, want ErrNotAnInstance", err)
	}
	// An Exception around an unconstructed instance surfaces the export error.
	badCarrier := mnemonica.NewException(&User{}, boom)
	if err := otelx.RecordLineage(context.Background(), badCarrier); !errors.Is(err, mnemonica.ErrNotAnInstance) {
		t.Errorf("unconstructed carrier error = %v, want ErrNotAnInstance", err)
	}
}

// TestEndToEnd is the L2 story in one flow: a request constructs typed
// data, the request ends, and only then do the worker goroutine (linked
// span) and the error path run.
func TestEndToEnd(t *testing.T) {
	fx := newOTelFixture(t)
	_, recorder, tracer := newRecorder(t)

	requestCtx, requestSpan := tracer.Start(context.Background(), "http-request")
	user, err := fx.userT.NewCtx(requestCtx, "ada")
	if err != nil {
		t.Fatalf("NewCtx: %v", err)
	}
	admin, err := fx.adminT.FromCtx(requestCtx, user, "root")
	if err != nil {
		t.Fatalf("FromCtx: %v", err)
	}
	requestSpan.End()
	requestContext := requestSpan.SpanContext()
	requestCtx = context.Background() // the request is over, its ctx unusable

	// The worker starts AFTER the request ended.
	workerDone := make(chan struct{})
	go func() {
		defer close(workerDone)
		_, worker := otelx.StartLinkedSpan(requestCtx, tracer, "background-job", admin)
		defer worker.End()
		// The job fails on data that carries its whole history:
		jobErr := mnemonica.NewException(admin, errors.New("job failed"))
		otelx.RecordLineage(trace.ContextWithSpan(requestCtx, worker), jobErr)
	}()
	<-workerDone

	workerSpan := endedSpan(t, recorder, "background-job")
	if len(workerSpan.Links()) != 1 ||
		workerSpan.Links()[0].SpanContext.TraceID() != requestContext.TraceID() {
		t.Fatalf("worker is not linked to the request span")
	}
	event := lineageEventOf(t, workerSpan)
	graph := assertGraphParses(t, event[otelx.AttrLineageGraph])
	nodes := graph["nodes"].(map[string]any)
	if _, ok := nodes[mnemonica.ID(admin)]; !ok {
		t.Error("admin missing from the error lineage graph")
	}
	if event[otelx.AttrInstanceID] != mnemonica.ID(admin) {
		t.Errorf("event instance id = %q, want the admin", event[otelx.AttrInstanceID])
	}

	// The request span carries the construction stamps.
	request := attributesOf(endedSpan(t, recorder, "http-request"))
	if request[otelx.AttrInstanceID] != mnemonica.ID(admin) ||
		request[otelx.AttrParentID] != mnemonica.ID(user) ||
		request[otelx.AttrTypePath] != "User.Admin" {
		t.Errorf("request stamps = %v", request)
	}
}

func BenchmarkStamping(b *testing.B) {
	// Stamped construction vs plain, with a valid (non-recording) span
	// context in the ctx so the full stamping path runs: two id formats
	// plus the attribute slice. Allocs are the honest signal on this
	// slow-clock host.
	fx := newOTelFixture(b)
	plain := newOTelFixtureStamped(b, false)
	spanContext := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    trace.TraceID{1, 2, 3},
		SpanID:     trace.SpanID{4, 5, 6},
		TraceFlags: trace.FlagsSampled,
	})
	stampedCtx := trace.ContextWithSpanContext(context.Background(), spanContext)

	b.Run("stamped", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			instance, err := fx.userT.NewCtx(stampedCtx, "ada")
			if err != nil {
				b.Fatal(err)
			}
			benchSink = instance
		}
	})
	b.Run("plain", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			instance, err := plain.userT.NewCtx(stampedCtx, "ada")
			if err != nil {
				b.Fatal(err)
			}
			benchSink = instance
		}
	})
}

var benchSink any

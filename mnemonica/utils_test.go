package mnemonica_test

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/wentout/mnemonica-go/mnemonica"
)

// Utils fixtures: a shadowing pair, the JS doc-example pipeline, the
// UTILS.md merge pair, and a type whose field cannot be marshaled.

type Gadget struct {
	mnemonica.Node
	Name  string
	Email string
}

type Job struct {
	mnemonica.Node
	*Gadget
	Name string // shadows Gadget.Name along the lineage
	Role string
}

// The FOR_HUMANS pipeline: Request → Route → Page → Response.
type Request struct {
	mnemonica.Node
	Method string
}

type Route struct {
	mnemonica.Node
	*Request
	Path string
}

type Page struct {
	mnemonica.Node
	*Route
	Template string
}

type Response struct {
	mnemonica.Node
	*Page
	Status int
}

// The UTILS.md merge example shapes.
type MergeUser struct {
	mnemonica.Node
	Name string
	Age  int
}

type MergeRole struct {
	mnemonica.Node
	Role string
}

// NamedRole conflicts with MergeUser on Name: A must win the merge.
type NamedRole struct {
	mnemonica.Node
	Name string
	Role string
}

// Unmarshalable carries a value encoding/json refuses.
type Unmarshalable struct {
	mnemonica.Node
	Fn func()
}

func newGadgetFixture() (*mnemonica.Collection, *mnemonica.TypeDef[Gadget, mnemonica.Root, string], *mnemonica.TypeDef[Job, Gadget, string]) {
	collection := mnemonica.NewCollection()
	gadgetT := mnemonica.Must(mnemonica.Define[Gadget](collection, "Gadget", func(a *Gadget, name string) error {
		a.Name = name
		a.Email = name + "@example.com"
		return nil
	}))
	jobT := mnemonica.Must(mnemonica.Sub[Job](gadgetT, "Job", func(m *Job, role string) error {
		m.Name = "member-" + role
		m.Role = role
		return nil
	}))
	return collection, gadgetT, jobT
}

func buildJob(t *testing.T) (*Gadget, *Job) {
	t.Helper()
	_, gadgetT, jobT := newGadgetFixture()
	account, err := gadgetT.New("ada")
	if err != nil {
		t.Fatalf("New account: %v", err)
	}
	member, err := jobT.From(account, "admin")
	if err != nil {
		t.Fatalf("From member: %v", err)
	}
	return account, member
}

func buildPipeline(t *testing.T) (request, route, page, response mnemonica.Instance) {
	t.Helper()
	collection := mnemonica.NewCollection()
	requestT := mnemonica.Must(mnemonica.Define[Request](collection, "Request", func(r *Request, method string) error {
		r.Method = method
		return nil
	}))
	routeT := mnemonica.Must(mnemonica.Sub[Route](requestT, "Route", func(r *Route, path string) error {
		r.Path = path
		return nil
	}))
	pageT := mnemonica.Must(mnemonica.Sub[Page](routeT, "Page", func(p *Page, template string) error {
		p.Template = template
		return nil
	}))
	responseT := mnemonica.Must(mnemonica.Sub[Response](pageT, "Response", func(res *Response, status int) error {
		res.Status = status
		return nil
	}))
	req, err := requestT.New("GET")
	if err != nil {
		t.Fatalf("New request: %v", err)
	}
	rt, err := routeT.From(req, "/home")
	if err != nil {
		t.Fatalf("From route: %v", err)
	}
	pg, err := pageT.From(rt, "default")
	if err != nil {
		t.Fatalf("From page: %v", err)
	}
	res, err := responseT.From(pg, 200)
	if err != nil {
		t.Fatalf("From response: %v", err)
	}
	return req, rt, pg, res
}

func TestExtractNearestWins(t *testing.T) {
	_, member := buildJob(t)
	got := mnemonica.Extract(member)
	want := map[string]any{
		"Name":  "member-admin", // the Job's own Name shadows the Gadget's
		"Email": "ada@example.com",
		"Role":  "admin",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Extract(member) = %v, want %v", got, want)
	}
}

func TestExtractSkipsUnexportedAndEmbedded(t *testing.T) {
	fx := newFixture()
	user, err := fx.userT.New("ada")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	userCacheWarm(user)
	got := mnemonica.Extract(user)
	if !reflect.DeepEqual(got, map[string]any{"Name": "ada"}) {
		t.Errorf("Extract(user) = %v, want map[Name:ada] only", got)
	}
}

// userCacheWarm sets the fixture's unexported field and must never leak
// into utils output.
func userCacheWarm(u *User) {
	u.cache = "warm"
}

func TestExtractUnconstructedInstance(t *testing.T) {
	// The JS extract of a plain object works; so does Extract of a Node
	// that never went through construction.
	got := mnemonica.Extract(&Gadget{Name: "bare", Email: "bare@example.com"})
	if !reflect.DeepEqual(got, map[string]any{"Name": "bare", "Email": "bare@example.com"}) {
		t.Errorf("Extract(&Gadget{}) = %v, want the bare fields", got)
	}
}

func TestExtractNilInputs(t *testing.T) {
	if got := mnemonica.Extract(nil); len(got) != 0 {
		t.Errorf("Extract(nil) = %v, want empty", got)
	}
	var nilUser *User
	if got := mnemonica.Extract(nilUser); len(got) != 0 {
		t.Errorf("Extract((*User)(nil)) = %v, want empty", got)
	}
}

func TestPick(t *testing.T) {
	_, member := buildJob(t)
	got := mnemonica.Pick(member, "Name", "Email", "Missing")
	want := map[string]any{
		"Name":  "member-admin",
		"Email": "ada@example.com",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Pick = %v, want %v (absent names stay absent)", got, want)
	}
	if got := mnemonica.Pick(member); len(got) != 0 {
		t.Errorf("Pick with no keys = %v, want empty", got)
	}
}

func TestParentDirectAndRoot(t *testing.T) {
	fx := newFixture()
	user, err := fx.userT.New("ada")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	admin, err := fx.adminT.From(user, "root")
	if err != nil {
		t.Fatalf("From: %v", err)
	}
	parent, ok := mnemonica.Parent(admin)
	if !ok || parent != user {
		t.Errorf("Parent(admin) = %v, %v; want the user", parent, ok)
	}
	if _, ok := mnemonica.Parent(user); ok {
		t.Error("Parent(root) found something, want (nil, false)")
	}
}

func TestParentByName(t *testing.T) {
	request, _, _, response := buildPipeline(t)
	for path, want := range map[string]mnemonica.Instance{
		"Request": request,
	} {
		got, ok := mnemonica.Parent(response, path)
		if !ok || got != want {
			t.Errorf("Parent(response, %q) = %v, %v; want the request", path, got, ok)
		}
	}
	if _, ok := mnemonica.Parent(response, "Nope"); ok {
		t.Error("Parent(response, \"Nope\") matched, want (nil, false)")
	}
	// The instance itself is never a candidate.
	if _, ok := mnemonica.Parent(request, "Request"); ok {
		t.Error("Parent(request, \"Request\") matched the instance itself, want (nil, false)")
	}
}

func TestParentDottedPath(t *testing.T) {
	_, route, page, response := buildPipeline(t)
	for path, want := range map[string]mnemonica.Instance{
		"Route":              route, // nearest ancestor named Route
		"Request.Route":      route, // contiguous: Request is Route's direct parent
		"Route.Page":         page,
		"Request.Route.Page": page,
	} {
		got, ok := mnemonica.Parent(response, path)
		if !ok || got != want {
			t.Errorf("Parent(response, %q) = %v, %v; want the documented match", path, got, ok)
		}
	}
	// Not contiguous: Page's direct parent is Route, not Request — the
	// dotted form must refuse even though both names exist up the chain.
	if _, ok := mnemonica.Parent(response, "Request.Page"); ok {
		t.Error("Parent(response, \"Request.Page\") matched a non-contiguous pair")
	}
	// Longer than the chain.
	if _, ok := mnemonica.Parent(response, "Root.Request.Route.Page"); ok {
		t.Error("Parent with a prefix above the root matched, want (nil, false)")
	}
	// An empty path means no filter.
	if got, ok := mnemonica.Parent(response, ""); !ok || got != page {
		t.Errorf("Parent(response, \"\") = %v, %v; want the direct parent", got, ok)
	}
}

func TestParentNilInputs(t *testing.T) {
	if _, ok := mnemonica.Parent(nil); ok {
		t.Error("Parent(nil) matched, want (nil, false)")
	}
	var nilUser *User
	if _, ok := mnemonica.Parent(nilUser); ok {
		t.Error("Parent((*User)(nil)) matched, want (nil, false)")
	}
}

func TestClone(t *testing.T) {
	fx := newFixture()
	user, err := fx.userT.New("ada")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	admin, err := fx.adminT.From(user, "root")
	if err != nil {
		t.Fatalf("From: %v", err)
	}
	// Clone re-runs construction: prove it with a postCreation hook.
	creations := 0
	if err := fx.adminT.RegisterHook(mnemonica.HookPostCreation, func(*mnemonica.HookData) error {
		creations++
		return nil
	}); err != nil {
		t.Fatalf("RegisterHook: %v", err)
	}
	clone, err := mnemonica.Clone(admin)
	if err != nil {
		t.Fatalf("Clone: %v", err)
	}
	if clone == admin {
		t.Error("Clone returned the same pointer, want a fresh instance")
	}
	if clone.Role != "root" || clone.Name != "ada" {
		t.Errorf("clone fields = (%q, %q), want (root, ada)", clone.Role, clone.Name)
	}
	record, err := mnemonica.Props(clone)
	if err != nil {
		t.Fatalf("Props: %v", err)
	}
	if record.Parent != user {
		t.Errorf("clone parent = %v, want the same user", record.Parent)
	}
	if record.Args != "root" {
		t.Errorf("clone args = %v, want %q", record.Args, "root")
	}
	if creations != 1 {
		t.Errorf("postCreation fired %d times, want 1 (clone re-runs the constructor)", creations)
	}
}

func TestCloneRoot(t *testing.T) {
	fx := newFixture()
	user, err := fx.userT.New("ada")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	clone, err := mnemonica.Clone(user)
	if err != nil {
		t.Fatalf("Clone root: %v", err)
	}
	if clone.Name != "ada" {
		t.Errorf("clone Name = %q, want %q", clone.Name, "ada")
	}
	record, err := mnemonica.Props(clone)
	if err != nil {
		t.Fatalf("Props: %v", err)
	}
	if record.Parent != nil {
		t.Errorf("clone root parent = %v, want nil", record.Parent)
	}
}

func TestCloneErrors(t *testing.T) {
	fx := newFixture()
	user, err := fx.userT.New("ada")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if _, err := mnemonica.Clone(&User{}); !errors.Is(err, mnemonica.ErrNotAnInstance) {
		t.Errorf("Clone(&User{}) error = %v, want ErrNotAnInstance", err)
	}
	var nilUser *User
	if _, err := mnemonica.Clone(nilUser); !errors.Is(err, mnemonica.ErrNotAnInstance) {
		t.Errorf("Clone((*User)(nil)) error = %v, want ErrNotAnInstance", err)
	}
	var noNodeValue noNode
	if _, err := mnemonica.Clone(&noNodeValue); !errors.Is(err, mnemonica.ErrNotAnInstance) {
		t.Errorf("Clone(&noNode{}) error = %v, want ErrNotAnInstance", err)
	}
	// A handler failure propagates through Clone — construction is real.
	fx.userT.SetHandler(func(u *User, name string) error {
		return errors.New("clone boom")
	})
	_, err = mnemonica.Clone(user)
	if err == nil || !strings.Contains(err.Error(), "clone boom") {
		t.Errorf("Clone with failing handler error = %v, want the handler error", err)
	}
}

func TestFork(t *testing.T) {
	fx := newFixture()
	user, err := fx.userT.New("ada")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	admin, err := fx.adminT.From(user, "root")
	if err != nil {
		t.Fatalf("From: %v", err)
	}
	forked, err := mnemonica.Fork(admin, "operator")
	if err != nil {
		t.Fatalf("Fork: %v", err)
	}
	if forked.Role != "operator" || forked.Name != "ada" {
		t.Errorf("forked fields = (%q, %q), want (operator, ada)", forked.Role, forked.Name)
	}
	record, err := mnemonica.Props(forked)
	if err != nil {
		t.Fatalf("Props: %v", err)
	}
	if record.Parent != user {
		t.Errorf("forked parent = %v, want the same user", record.Parent)
	}
	// Wrong args for the type surface ErrWrongArgumentsUsed.
	if _, err := mnemonica.Fork(admin, 42); !errors.Is(err, mnemonica.ErrWrongArgumentsUsed) {
		t.Errorf("Fork with wrong args error = %v, want ErrWrongArgumentsUsed", err)
	}
	// A root forks too.
	other, err := mnemonica.Fork(user, "grace")
	if err != nil {
		t.Fatalf("Fork root: %v", err)
	}
	if other.Name != "grace" {
		t.Errorf("forked root Name = %q, want %q", other.Name, "grace")
	}
}

func TestForkErrors(t *testing.T) {
	if _, err := mnemonica.Fork(&User{}, "x"); !errors.Is(err, mnemonica.ErrNotAnInstance) {
		t.Errorf("Fork(&User{}) error = %v, want ErrNotAnInstance", err)
	}
	var nilUser *User
	if _, err := mnemonica.Fork(nilUser, "x"); !errors.Is(err, mnemonica.ErrNotAnInstance) {
		t.Errorf("Fork(nil) error = %v, want ErrNotAnInstance", err)
	}
}

func TestForkOntoDAG(t *testing.T) {
	fx := newFixture()
	first, err := fx.userT.New("ada")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	second, err := fx.userT.New("grace")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	admin, err := fx.adminT.From(first, "root")
	if err != nil {
		t.Fatalf("From: %v", err)
	}
	dag, err := mnemonica.ForkOnto(admin, second, "operator")
	if err != nil {
		t.Fatalf("ForkOnto: %v", err)
	}
	record, err := mnemonica.Props(dag)
	if err != nil {
		t.Fatalf("Props: %v", err)
	}
	if record.Parent != second {
		t.Errorf("DAG parent = %v, want the second user %p", record.Parent, second)
	}
	if dag.Name != "grace" {
		t.Errorf("DAG promoted Name = %q, want %q (read-through from the new parent)", dag.Name, "grace")
	}
	// A wrong parent kind is still rejected for subtypes.
	widgetT := newWidget()
	widget, err := widgetT.New("sku-1")
	if err != nil {
		t.Fatalf("New widget: %v", err)
	}
	if _, err := mnemonica.ForkOnto(admin, widget, "x"); !errors.Is(err, mnemonica.ErrWrongModificationPattern) {
		t.Errorf("ForkOnto with wrong parent error = %v, want ErrWrongModificationPattern", err)
	}
}

func TestForkOntoReparentsRoots(t *testing.T) {
	// The JS merge/fork.call form: a ROOT-typed instance can be re-parented
	// onto any instance — no parent requirement exists for roots there.
	collection := mnemonica.NewCollection()
	gadgetT := mnemonica.Must(mnemonica.Define[Gadget](collection, "Gadget", func(a *Gadget, name string) error {
		a.Name = name
		a.Email = name + "@example.com"
		return nil
	}))
	widgetT := mnemonica.Must(mnemonica.Define[Widget](collection, "Widget", func(w *Widget, sku string) error {
		w.SKU = sku
		return nil
	}))
	account, err := gadgetT.New("ada")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	widget, err := widgetT.New("gadget")
	if err != nil {
		t.Fatalf("New widget: %v", err)
	}
	reparented, err := mnemonica.ForkOnto(account, widget, "grace")
	if err != nil {
		t.Fatalf("ForkOnto root: %v", err)
	}
	record, err := mnemonica.Props(reparented)
	if err != nil {
		t.Fatalf("Props: %v", err)
	}
	if record.Parent != widget {
		t.Errorf("re-parented root parent = %v, want the widget", record.Parent)
	}
}

func TestSibling(t *testing.T) {
	fx := newFixture()
	user, err := fx.userT.New("ada")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	first, err := fx.adminT.From(user, "root")
	if err != nil {
		t.Fatalf("From: %v", err)
	}
	sibling, err := mnemonica.Sibling(first, "operator")
	if err != nil {
		t.Fatalf("Sibling: %v", err)
	}
	if sibling.Role != "operator" {
		t.Errorf("sibling Role = %q, want %q", sibling.Role, "operator")
	}
	record, err := mnemonica.Props(sibling)
	if err != nil {
		t.Fatalf("Props: %v", err)
	}
	if record.Parent != user {
		t.Errorf("sibling parent = %v, want the shared user", record.Parent)
	}
	if record.Type.Path() != "User.Admin" {
		t.Errorf("sibling type = %q, want %q", record.Type.Path(), "User.Admin")
	}
	if _, err := mnemonica.Sibling(&User{}, "x"); !errors.Is(err, mnemonica.ErrNotAnInstance) {
		t.Errorf("Sibling(&User{}) error = %v, want ErrNotAnInstance", err)
	}
}

func TestMerge(t *testing.T) {
	// The UTILS.md example: merge(user, role) — a's fields primary, b's
	// fields filling the non-overlapping keys through read-through.
	collection := mnemonica.NewCollection()
	userT := mnemonica.Must(mnemonica.Define[MergeUser](collection, "User", func(u *MergeUser, name string) error {
		u.Name = name
		u.Age = 30
		return nil
	}))
	roleT := mnemonica.Must(mnemonica.Define[MergeRole](collection, "Role", func(r *MergeRole, role string) error {
		r.Role = role
		return nil
	}))
	user, err := userT.New("alice")
	if err != nil {
		t.Fatalf("New user: %v", err)
	}
	role, err := roleT.New("admin")
	if err != nil {
		t.Fatalf("New role: %v", err)
	}
	merged, err := mnemonica.Merge(user, role, "alice")
	if err != nil {
		t.Fatalf("Merge: %v", err)
	}
	got := mnemonica.Extract(merged)
	want := map[string]any{"Name": "alice", "Age": 30, "Role": "admin"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Extract(merged) = %v, want %v", got, want)
	}
	record, err := mnemonica.Props(merged)
	if err != nil {
		t.Fatalf("Props: %v", err)
	}
	if record.Parent != role {
		t.Errorf("merged parent = %v, want the role instance", record.Parent)
	}

	// A wins on conflicts: the merged type's own Name shadows b's.
	namedT := mnemonica.Must(mnemonica.Define[NamedRole](collection, "NamedRole", func(n *NamedRole, pair string) error {
		n.Name = "role-" + pair
		n.Role = pair
		return nil
	}))
	named, err := namedT.New("admin")
	if err != nil {
		t.Fatalf("New named role: %v", err)
	}
	mergedConflict, err := mnemonica.Merge(user, named, "alice")
	if err != nil {
		t.Fatalf("Merge conflict: %v", err)
	}
	if got := mnemonica.Extract(mergedConflict); got["Name"] != "alice" {
		t.Errorf("conflict Name = %v, want alice (a wins)", got["Name"])
	}

	if _, err := mnemonica.Merge(&MergeUser{}, role, "x"); !errors.Is(err, mnemonica.ErrNotAnInstance) {
		t.Errorf("Merge(&MergeUser{}) error = %v, want ErrNotAnInstance", err)
	}
}

func TestException(t *testing.T) {
	fx := newFixture()
	user, err := fx.userT.New("ada")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	boom := errors.New("boom")
	exception := mnemonica.NewException(user, boom, 1, 2)
	if !errors.Is(exception, boom) {
		t.Error("errors.Is(exception, boom) = false, want true (Unwrap)")
	}
	if exception.Instance() != user {
		t.Errorf("Instance() = %v, want the user", exception.Instance())
	}
	if !reflect.DeepEqual(exception.Args(), []any{1, 2}) {
		t.Errorf("Args() = %v, want [1 2]", exception.Args())
	}
	if !strings.Contains(exception.Error(), "boom") {
		t.Errorf("Error() = %q, want it to mention the wrapped error", exception.Error())
	}
	// A nil original is allowed and does not panic.
	plain := mnemonica.NewException(user, nil)
	if !strings.Contains(plain.Error(), "exception") {
		t.Errorf("Error() = %q, want a usable message", plain.Error())
	}
	if plain.Unwrap() != nil {
		t.Errorf("Unwrap() = %v, want nil", plain.Unwrap())
	}
	if plain.Args() != nil {
		t.Errorf("Args() = %v, want nil", plain.Args())
	}
}

func TestParse(t *testing.T) {
	fx := newFixture()
	user, err := fx.userT.New("ada")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	admin, err := fx.adminT.From(user, "root")
	if err != nil {
		t.Fatalf("From: %v", err)
	}
	parsed := mnemonica.Parse(admin)
	if parsed.Name != "Admin" {
		t.Errorf("Name = %q, want %q", parsed.Name, "Admin")
	}
	if !reflect.DeepEqual(parsed.Props, mnemonica.Extract(admin)) {
		t.Errorf("Props = %v, want the Extract", parsed.Props)
	}
	if parsed.Self != admin {
		t.Errorf("Self = %v, want the admin", parsed.Self)
	}
	// Parent is the parent INSTANCE (the JS fix), not a name.
	if parsed.Parent != user {
		t.Errorf("Parent = %v, want the user instance", parsed.Parent)
	}

	rootParsed := mnemonica.Parse(user)
	if rootParsed.Name != "User" || rootParsed.Parent != nil {
		t.Errorf("root parse = (%q, %v), want (User, nil parent)", rootParsed.Name, rootParsed.Parent)
	}

	zero := mnemonica.Parse(&User{})
	if zero.Name != "" || zero.Props != nil || zero.Self != nil || zero.Parent != nil {
		t.Errorf("Parse(&User{}) = %+v, want the zero Parsed", zero)
	}
	if zero := mnemonica.Parse(nil); zero.Name != "" {
		t.Errorf("Parse(nil) = %+v, want the zero Parsed", zero)
	}
}

func TestToJSON(t *testing.T) {
	_, member := buildJob(t)
	encoded, err := mnemonica.ToJSON(member)
	if err != nil {
		t.Fatalf("ToJSON: %v", err)
	}
	// Round-trip: the output is valid JSON of the Extract.
	var decoded map[string]any
	if err := json.Unmarshal([]byte(encoded), &decoded); err != nil {
		t.Fatalf("ToJSON output is not valid JSON: %v", err)
	}
	if decoded["Name"] != "member-admin" || decoded["Email"] != "ada@example.com" || decoded["Role"] != "admin" {
		t.Errorf("decoded = %v, want the extracted fields", decoded)
	}
}

func TestToJSONEmptyAlwaysValid(t *testing.T) {
	collection := mnemonica.NewCollection()
	orphanT := mnemonica.Must(mnemonica.Define[Orphan](collection, "Orphan", func(o *Orphan, label string) error {
		return nil
	}))
	orphan, err := orphanT.New("label")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	encoded, err := mnemonica.ToJSON(orphan)
	if err != nil {
		t.Fatalf("ToJSON: %v", err)
	}
	if encoded != "{}" {
		t.Errorf("ToJSON of a field-less instance = %q, want %q", encoded, "{}")
	}
}

func TestToJSONUnmarshalable(t *testing.T) {
	collection := mnemonica.NewCollection()
	badT := mnemonica.Must(mnemonica.Define[Unmarshalable](collection, "Unmarshalable", func(u *Unmarshalable, label string) error {
		return nil
	}))
	bad, err := badT.New("label")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	bad.Fn = func() {}
	if _, err := mnemonica.ToJSON(bad); err == nil {
		t.Error("ToJSON of an unmarshalable field succeeded, want an error")
	}
}

func TestConstructorSequence(t *testing.T) {
	_, _, _, response := buildPipeline(t)
	sequence := mnemonica.ConstructorSequence(response)
	want := []string{"Response", "Page", "Route", "Request"} // nearest first
	if !reflect.DeepEqual(sequence, want) {
		t.Errorf("ConstructorSequence = %v, want %v", sequence, want)
	}
	fx := newFixture()
	user, err := fx.userT.New("ada")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if got := mnemonica.ConstructorSequence(user); !reflect.DeepEqual(got, []string{"User"}) {
		t.Errorf("root sequence = %v, want [User]", got)
	}
	if got := mnemonica.ConstructorSequence(&User{}); got != nil {
		t.Errorf("unconstructed sequence = %v, want nil", got)
	}
	var nilUser *User
	if got := mnemonica.ConstructorSequence(nilUser); got != nil {
		t.Errorf("typed-nil sequence = %v, want nil", got)
	}
}

func TestReconstructingUtilsRejectBadInputs(t *testing.T) {
	fx := newFixture()
	user, err := fx.userT.New("ada")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	var nilUser *User
	if _, err := mnemonica.ForkOnto(&User{}, user, "x"); !errors.Is(err, mnemonica.ErrNotAnInstance) {
		t.Errorf("ForkOnto(&User{}) error = %v, want ErrNotAnInstance", err)
	}
	if _, err := mnemonica.ForkOnto(nilUser, user, "x"); !errors.Is(err, mnemonica.ErrNotAnInstance) {
		t.Errorf("ForkOnto(nil) error = %v, want ErrNotAnInstance", err)
	}
	if _, err := mnemonica.Sibling(nilUser, "x"); !errors.Is(err, mnemonica.ErrNotAnInstance) {
		t.Errorf("Sibling(nil) error = %v, want ErrNotAnInstance", err)
	}
	if _, err := mnemonica.Merge(nilUser, user, "x"); !errors.Is(err, mnemonica.ErrNotAnInstance) {
		t.Errorf("Merge(nil) error = %v, want ErrNotAnInstance", err)
	}
}

func TestReconstructingUtilsPropagateHandlerErrors(t *testing.T) {
	// Every reconstructing util re-runs the constructor: a swapped failing
	// handler fails the reconstruction too.
	collection := mnemonica.NewCollection()
	gadgetT := mnemonica.Must(mnemonica.Define[Gadget](collection, "Gadget", func(a *Gadget, name string) error {
		a.Name = name
		a.Email = name + "@example.com"
		return nil
	}))
	widgetT := mnemonica.Must(mnemonica.Define[Widget](collection, "Widget", func(w *Widget, sku string) error {
		w.SKU = sku
		return nil
	}))
	account, err := gadgetT.New("ada")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	widget, err := widgetT.New("gadget")
	if err != nil {
		t.Fatalf("New widget: %v", err)
	}
	gadgetT.SetHandler(func(a *Gadget, name string) error {
		return errors.New("account boom")
	})
	if _, err := mnemonica.ForkOnto(account, widget, "grace"); err == nil || !strings.Contains(err.Error(), "account boom") {
		t.Errorf("ForkOnto with failing root handler error = %v, want the handler error", err)
	}
	fx := newFixture()
	user, err := fx.userT.New("ada")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	admin, err := fx.adminT.From(user, "root")
	if err != nil {
		t.Fatalf("From: %v", err)
	}
	fx.adminT.SetHandler(func(a *Admin, role string) error {
		return errors.New("admin boom")
	})
	if _, err := mnemonica.Sibling(admin, "x"); err == nil || !strings.Contains(err.Error(), "admin boom") {
		t.Errorf("Sibling with failing handler error = %v, want the handler error", err)
	}
	if _, err := mnemonica.Merge(admin, user, "x"); err == nil || !strings.Contains(err.Error(), "admin boom") {
		t.Errorf("Merge with failing handler error = %v, want the handler error", err)
	}
}

func TestCollectConstructors(t *testing.T) {
	_, _, _, response := buildPipeline(t)
	constructors := mnemonica.CollectConstructors(response)
	for _, name := range []string{"Response", "Page", "Route", "Request"} {
		if !constructors[name] {
			t.Errorf("CollectConstructors missing %q: %v", name, constructors)
		}
	}
	if constructors["Nope"] {
		t.Error("CollectConstructors contains an unrelated name")
	}
}

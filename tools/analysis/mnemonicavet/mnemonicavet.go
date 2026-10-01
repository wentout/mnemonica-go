// Command mnemonicavet is the static half of the C2.4 adapted guard: an
// analyzer reporting assignments THROUGH a promoted field of an embedded
// lineage parent — `admin.Name = x` where Name belongs to the embedded
// *User — because Go cannot shadow promoted writes and the write reaches
// the shared parent. The runtime/test half is mnemonicatest
// (WithParentSnapshots + AssertParentsUnchanged); this is the vet half.
//
// Not flagged: assignments to a type's OWN fields, assignments in
// non-mnemonica structs, fields promoted from a NON-lineage embed, fields
// promoted through a VALUE embed of a lineage struct (the write stays on
// a copy), assignments to the parent itself, and method calls (this
// analyzer looks at assignments only).
package main

import (
	"go/ast"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/singlechecker"
)

// mnemonicaPath is the runtime module's import path (placeholder module
// path, owned by viktor).
const mnemonicaPath = "mnemonica/mnemonica"

// Analyzer reports promoted-field assignments into shared lineage parents.
var Analyzer = &analysis.Analyzer{
	Name: "mnemonicavet",
	Doc:  "reports assignments through promoted fields of embedded lineage parents (C2.4 write-local guard)",
	Run:  run,
}

// singlecheckerMain is singlechecker.Main under a name, so tests can swap
// it out and cover main without exiting the test binary.
var singlecheckerMain = singlechecker.Main

func main() {
	singlecheckerMain(Analyzer)
}

func run(pass *analysis.Pass) (any, error) {
	for _, file := range pass.Files {
		ast.Inspect(file, func(node ast.Node) bool {
			assign, ok := node.(*ast.AssignStmt)
			if !ok || assign.Tok != token.ASSIGN {
				return true
			}
			for _, lhs := range assign.Lhs {
				checkLHS(pass, lhs)
			}
			return true
		})
	}
	return nil, nil
}

// checkLHS reports a single assignment target when it writes through a
// promoted field of an embedded lineage parent.
func checkLHS(pass *analysis.Pass, lhs ast.Expr) {
	selector, ok := lhs.(*ast.SelectorExpr)
	if !ok {
		return // plain identifiers, indexes, etc.
	}
	// A selector in assignment position has a types.Selection when its base
	// is a value; package selectors (pkg.Var = x) have none.
	selection, ok := pass.TypesInfo.Selections[selector]
	if !ok {
		return
	}
	// FieldVal is the only possible selection kind on an assignment target:
	// method values cannot be assigned to, so no kind check is needed.
	receiverStructure := structOf(selection.Recv())
	firstIndex := selection.Index()[0]
	firstHop := receiverStructure.Field(firstIndex)
	if !firstHop.Embedded() {
		return // the type's OWN field
	}
	index := selection.Index()
	if len(index) == 1 {
		// The assignment targets the embedded parent POINTER itself —
		// the sanctioned wiring shape (WithWireFunc and the generated
		// wire funcs) or an explicit parent swap. Deeper paths are the
		// violation.
		return
	}
	if !embedsNode(selection.Recv(), make(map[*types.Named]bool)) {
		return // not a mnemonica instance: not this analyzer's business
	}
	pointer, ok := firstHop.Type().(*types.Pointer)
	if !ok {
		return // value embed: the write stays on a copy, not the shared parent
	}
	// An embedded *T always names T, so parentNamed cannot be nil.
	parentNamed := namedOf(pointer.Elem())
	if !embedsNode(pointer.Elem(), make(map[*types.Named]bool)) {
		return // promoted from a NON-lineage embed
	}
	pass.Reportf(selector.Pos(), "assignment to promoted field %s through the embedded lineage parent *%s: the write reaches the shared parent (adapted C2.4); assign to a local field instead",
		selection.Obj().Name(), parentNamed.Obj().Name())
}

// namedOf resolves a type to its named type (through a pointer), or nil
// for anonymous types.
func namedOf(t types.Type) *types.Named {
	if pointer, ok := t.(*types.Pointer); ok {
		t = pointer.Elem()
	}
	named, _ := t.(*types.Named)
	return named
}

// structOf resolves a type to its underlying struct (through a pointer).
// A FieldVal selection guarantees a struct receiver, so the result is
// never nil here.
func structOf(t types.Type) *types.Struct {
	if pointer, ok := t.(*types.Pointer); ok {
		t = pointer.Elem()
	}
	structure, _ := t.Underlying().(*types.Struct)
	return structure
}

// embedsNode reports whether the type transitively embeds mnemonica.Node,
// tracking named types against pointer-embed cycles. Embedded fields are
// always named types in Go, so namedOf cannot return nil here; named source
// types always carry a package, so the path comparison needs no nil guard.
func embedsNode(t types.Type, seen map[*types.Named]bool) bool {
	named := namedOf(t)
	if seen[named] {
		return false // pointer-embed cycle
	}
	if named.Obj().Name() == "Node" && named.Obj().Pkg().Path() == mnemonicaPath {
		return true
	}
	seen[named] = true
	structure, ok := named.Underlying().(*types.Struct)
	if !ok {
		return false // named non-struct embed: nothing below it
	}
	for index := 0; index < structure.NumFields(); index++ {
		field := structure.Field(index)
		if !field.Embedded() {
			continue
		}
		if embedsNode(field.Type(), seen) {
			return true
		}
	}
	return false
}

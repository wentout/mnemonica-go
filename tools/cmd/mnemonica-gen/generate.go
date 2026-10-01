package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/token"
	"go/types"
	"path/filepath"
	"sort"
	"strings"

	"golang.org/x/tools/go/packages"
)

// outputName is the generated file. It is excluded from the generator's
// own input so re-running after a previous generation does not collide
// with the methods it produced.
const outputName = "mnemonica_gen.go"

// mnemonicaPath is the runtime module's import path (placeholder module
// path, owned by viktor).
const mnemonicaPath = "mnemonica/mnemonica"

// subtype is one mnemonica.Sub call the generator turns into a wire func
// and a parent-struct method pair.
type subtype struct {
	varName    string // the package-level TypeDef var (jobT)
	name       string // the subtype's declared name ("Job")
	child      string // the subtype's struct type name
	parent     string // the parent struct type name
	argsType   string // rendered args type name
	embedField string // the child struct's embedded *Parent field name
}

// defineCall is a package-level var bound to a Define or Sub call.
type defineCall struct {
	child string
}

// Generate renders the mnemonica-gen output for the package in dir. It is
// the testable core; main is a thin wrapper. The output depends only on
// the package's source (sorted), never on absolute paths.
func Generate(dir string) ([]byte, error) {
	result, err := generate(dir, nil)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// generate is Generate with an optional file overlay (a packages.Config
// feature) — the seam tests use to exercise paths real sources cannot
// reach.
func generate(dir string, overlay map[string][]byte) ([]byte, error) {
	fset := token.NewFileSet()
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedCompiledGoFiles |
			packages.NeedSyntax | packages.NeedTypes |
			packages.NeedTypesInfo | packages.NeedImports | packages.NeedDeps,
		Dir:     dir,
		Fset:    fset,
		Overlay: overlay,
	}
	pkgs, err := packages.Load(cfg, ".")
	if err != nil {
		return nil, err
	}
	// go/packages returns at least one package or an error, so pkgs[0] is
	// safe without a length guard.
	pkg := pkgs[0]
	if len(pkg.Errors) > 0 {
		return nil, fmt.Errorf("package %s has errors: %v", pkg.PkgPath, pkg.Errors[0])
	}

	files := syntaxFiles(pkg)
	varTable := collectVars(pkg, files)
	subtypes, imports, err := collectSubtypes(pkg, files, varTable)
	if err != nil {
		return nil, err
	}
	result, err := render(pkg.Name, subtypes, imports)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// syntaxFiles returns the package's files minus a previously generated
// output file, so re-running does not collide with the methods it made.
func syntaxFiles(pkg *packages.Package) []*ast.File {
	var files []*ast.File
	for index, path := range pkg.CompiledGoFiles {
		if filepath.Base(path) == outputName {
			continue
		}
		files = append(files, pkg.Syntax[index])
	}
	return files
}

// mnemonicaCall reports whether call invokes the given mnemonica runtime
// function, looking through the generic indices (explicit or inferred) and
// the Must wrapper, and returns the effective inner call. Identification
// goes through types (Uses of the selector's X resolves to the PkgName), so
// aliased imports do not fool it.
func mnemonicaCall(info *types.Info, call *ast.CallExpr, want string) *ast.CallExpr {
	inner := unwrapMust(info, call)
	switch fun := inner.Fun.(type) {
	case *ast.SelectorExpr:
		if matchSelector(info, fun, want) {
			return inner
		}
	case *ast.IndexExpr:
		if matchSelector(info, fun.X, want) {
			return inner
		}
	case *ast.IndexListExpr:
		if matchSelector(info, fun.X, want) {
			return inner
		}
	}
	return nil
}

// unwrapMust peels mnemonica.Must(...) down to the wrapped call. The
// two-argument form Must(td, err) and non-call arguments pass through.
func unwrapMust(info *types.Info, call *ast.CallExpr) *ast.CallExpr {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Must" || !selectorIsMnemonica(info, sel) {
		return call
	}
	if len(call.Args) == 1 {
		if wrapped, ok := call.Args[0].(*ast.CallExpr); ok {
			return wrapped
		}
	}
	return call
}

// matchSelector reports whether x is pkg.want with pkg = mnemonica.
func matchSelector(info *types.Info, x ast.Expr, want string) bool {
	sel, ok := x.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != want {
		return false
	}
	return selectorIsMnemonica(info, sel)
}

// selectorIsMnemonica reports whether sel.X names the mnemonica package,
// without branching on shapes that cannot occur in compiling code (the
// zero values do the work).
func selectorIsMnemonica(info *types.Info, sel *ast.SelectorExpr) bool {
	ident, isIdent := sel.X.(*ast.Ident)
	pkgName, isPkg := info.Uses[ident].(*types.PkgName)
	return isIdent && isPkg && pkgName.Imported().Path() == mnemonicaPath
}

// collectVars maps package-level var names to their Define/Sub call's
// child struct type, for parent-handle resolution and init wiring.
func collectVars(pkg *packages.Package, files []*ast.File) map[string]defineCall {
	table := make(map[string]defineCall)
	for _, file := range files {
		for _, decl := range file.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok || gen.Tok != token.VAR {
				continue
			}
			for _, spec := range gen.Specs {
				valueSpec, ok := spec.(*ast.ValueSpec)
				if !ok || len(valueSpec.Names) == 0 || len(valueSpec.Values) != 1 {
					continue
				}
				call, ok := valueSpec.Values[0].(*ast.CallExpr)
				if !ok {
					continue
				}
				inner := mnemonicaCall(pkg.TypesInfo, call, "Define")
				if inner == nil {
					inner = mnemonicaCall(pkg.TypesInfo, call, "Sub")
				}
				if inner == nil {
					continue
				}
				if child := callChildType(pkg, inner); child != "" {
					table[valueSpec.Names[0].Name] = defineCall{child: child}
				}
			}
		}
	}
	return table
}

// callChildType resolves a Define/Sub call's first type parameter (or its
// handler's instance parameter, when the type argument is inferred) to a
// struct type name.
func callChildType(pkg *packages.Package, call *ast.CallExpr) string {
	indices := callIndices(call)
	if len(indices) > 0 {
		if name := namedNameOf(pkg.TypesInfo.TypeOf(indices[0])); name != "" {
			return name
		}
	}
	handler := handlerSignature(pkg.TypesInfo, call)
	return namedNameOf(handler.Params().At(0).Type())
}

// handlerSignature resolves the call's handler argument to its func type;
// the runtime's generics guarantee it is a func with two parameters.
func handlerSignature(info *types.Info, call *ast.CallExpr) *types.Signature {
	sig, _ := info.TypeOf(call.Args[2]).(*types.Signature)
	return sig
}

// callIndices returns the explicit type arguments of a generic call.
func callIndices(call *ast.CallExpr) []ast.Expr {
	switch fun := call.Fun.(type) {
	case *ast.IndexExpr:
		return []ast.Expr{fun.Index}
	case *ast.IndexListExpr:
		return fun.Indices
	}
	return nil
}

// namedNameOf returns the name of a named type (through pointers), or "".
func namedNameOf(t types.Type) string {
	if ptr, ok := t.(*types.Pointer); ok {
		t = ptr.Elem()
	}
	named, ok := t.(*types.Named)
	if !ok {
		return ""
	}
	result := named.Obj().Name()
	return result
}

// renderType renders a type with same-package references unqualified, so
// the generated code reads naturally next to the hand-written code.
func renderType(pkg *packages.Package, t types.Type) string {
	qualifier := func(p *types.Package) string {
		if p == pkg.Types {
			return ""
		}
		return p.Name()
	}
	result := types.TypeString(t, qualifier)
	return result
}

// collectSubtypes walks the package's Sub calls and builds the generation
// set, sorted for deterministic output, plus the sorted foreign import
// paths the generated file needs.
func collectSubtypes(pkg *packages.Package, files []*ast.File, varTable map[string]defineCall) ([]subtype, []string, error) {
	var subs []subtype
	found := make(map[string]bool)
	for _, file := range files {
		for _, decl := range file.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok || gen.Tok != token.VAR {
				continue
			}
			for _, spec := range gen.Specs {
				valueSpec, ok := spec.(*ast.ValueSpec)
				if !ok || len(valueSpec.Names) == 0 || len(valueSpec.Values) != 1 {
					continue
				}
				call, ok := valueSpec.Values[0].(*ast.CallExpr)
				if !ok {
					continue
				}
				sub := mnemonicaCall(pkg.TypesInfo, call, "Sub")
				if sub == nil {
					continue
				}
				built, err := buildSubtype(pkg, files, varTable, found, valueSpec.Names[0].Name, sub)
				if err != nil {
					return nil, nil, err
				}
				subs = append(subs, built)
			}
		}
	}
	sort.Slice(subs, func(i, j int) bool {
		if subs[i].parent != subs[j].parent {
			return subs[i].parent < subs[j].parent
		}
		return subs[i].name < subs[j].name
	})
	imports := make([]string, 0, len(found))
	for path := range found {
		imports = append(imports, path)
	}
	sort.Strings(imports)
	return subs, imports, nil
}

// buildSubtype validates one Sub call and resolves everything the render
// needs. It errs rather than guesses: a silent skip would lose wiring.
// found collects the import paths of foreign packages referenced by the
// args type, for the generated file's import block.
func buildSubtype(pkg *packages.Package, files []*ast.File, varTable map[string]defineCall, found map[string]bool, varName string, call *ast.CallExpr) (subtype, error) {
	var zero subtype
	name, ok := constString(pkg.TypesInfo, call.Args[1])
	if !ok {
		return zero, fmt.Errorf("%s: Sub name must be a constant string", varName)
	}
	parentIdent, ok := call.Args[0].(*ast.Ident)
	if !ok {
		return zero, fmt.Errorf("%s: parent must be a package-level handle (an identifier)", varName)
	}
	parentCall, foundInTable := varTable[parentIdent.Name]
	if !foundInTable {
		return zero, fmt.Errorf("%s: parent handle %q is not defined by a Define or Sub in this package; mnemonica-gen only wires same-package handles", varName, parentIdent.Name)
	}
	child := callChildType(pkg, call)
	if child == "" {
		return zero, fmt.Errorf("%s: the subtype must be a named struct type", varName)
	}
	// child and parentCall.child are non-empty here (checked above, and the
	// var table only stores non-empty names), so both resolve in the
	// package scope — findNamed cannot miss.
	childObj := findNamed(pkg, child)
	embedField, err := embeddedParentField(childObj, parentCall.child)
	if err != nil {
		return zero, fmt.Errorf("%s: %w", varName, err)
	}
	if declaresMethod(files, parentCall.child, name) {
		return zero, fmt.Errorf("%s: *%s already declares a method %s; mnemonica-gen refuses to overwrite it", varName, parentCall.child, name)
	}
	args := resolveArgsType(pkg, call)
	foreignPackages(args, pkg.Types, found)
	return subtype{
		varName:    varName,
		name:       name,
		child:      child,
		parent:     parentCall.child,
		argsType:   renderType(pkg, args),
		embedField: embedField,
	}, nil
}

// resolveArgsType resolves the args type the generated method must name:
// the last explicit type argument when several are given, otherwise the
// handler's args parameter.
func resolveArgsType(pkg *packages.Package, call *ast.CallExpr) types.Type {
	indices := callIndices(call)
	if len(indices) > 1 {
		return pkg.TypesInfo.TypeOf(indices[len(indices)-1])
	}
	handler := handlerSignature(pkg.TypesInfo, call)
	return handler.Params().At(1).Type()
}

// foreignPackages records the import paths of named types inside t that
// live outside the current package. The walk covers the shapes args types
// realistically take; exotic nesting (channels, func types) is documented
// as a generator limitation.
func foreignPackages(t types.Type, current *types.Package, found map[string]bool) {
	switch typed := t.(type) {
	case *types.Named:
		if pkg := typed.Obj().Pkg(); pkg != nil && pkg != current {
			found[pkg.Path()] = true
		}
	case *types.Pointer:
		foreignPackages(typed.Elem(), current, found)
	case *types.Slice:
		foreignPackages(typed.Elem(), current, found)
	case *types.Map:
		foreignPackages(typed.Key(), current, found)
		foreignPackages(typed.Elem(), current, found)
	case *types.Struct:
		for index := 0; index < typed.NumFields(); index++ {
			foreignPackages(typed.Field(index).Type(), current, found)
		}
	}
}

// constString extracts an exact constant string value.
func constString(info *types.Info, expr ast.Expr) (string, bool) {
	value := info.Types[expr].Value
	if value == nil {
		return "", false
	}
	result := strings.Trim(value.ExactString(), `"`)
	return result, true
}

// findNamed resolves a named type in the package by name; the names fed
// here come from the package's own type information, so a miss means the
// source did not compile — which Generate already checked.
func findNamed(pkg *packages.Package, name string) *types.Named {
	obj := pkg.Types.Scope().Lookup(name)
	typeName, _ := obj.(*types.TypeName)
	named, _ := typeName.Type().(*types.Named)
	return named
}

// embeddedParentField returns the child's embedded field of type
// *parentName, reporting an error when there is none (including the
// WithWireFunc case, where the shape is the user's own assertion and the
// generator has nothing to attach).
func embeddedParentField(child *types.Named, parentName string) (string, error) {
	structure, ok := child.Underlying().(*types.Struct)
	if !ok {
		return "", fmt.Errorf("type %s is not a struct", child.Obj().Name())
	}
	for index := 0; index < structure.NumFields(); index++ {
		field := structure.Field(index)
		if !field.Embedded() {
			continue
		}
		if namedNameOf(field.Type()) == parentName {
			return field.Name(), nil
		}
	}
	return "", fmt.Errorf("type %s has no embedded *%s field (is the subtype wired by a custom WithWireFunc?)", child.Obj().Name(), parentName)
}

// declaresMethod reports whether the hand-written syntax already declares
// a method with the given name on the receiver type — the generator must
// never overwrite user code. The check is AST-based (not a types method
// set) so a previous generated file, present in the type information but
// filtered out of the syntax, cannot collide with itself on re-runs.
func declaresMethod(files []*ast.File, receiver, name string) bool {
	for _, file := range files {
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv == nil || fn.Name.Name != name {
				continue
			}
			if methodReceiverName(fn) == receiver {
				return true
			}
		}
	}
	return false
}

// methodReceiverName resolves a method's receiver type name ("*" stripped).
func methodReceiverName(fn *ast.FuncDecl) string {
	receiverType := fn.Recv.List[0].Type
	if star, ok := receiverType.(*ast.StarExpr); ok {
		receiverType = star.X
	}
	ident, _ := receiverType.(*ast.Ident)
	return ident.Name
}

// render produces the gofmt'd generated file. Output is fully determined
// by the (sorted) generation set and imports.
func render(packageName string, subs []subtype, imports []string) ([]byte, error) {
	var buffer bytes.Buffer
	buffer.WriteString("// Code generated by mnemonica-gen; DO NOT EDIT.\n\n")
	buffer.WriteString("package " + packageName + "\n\n")
	if len(subs) > 0 {
		buffer.WriteString("import (\n\t\"context\"\n")
		for _, path := range imports {
			fmt.Fprintf(&buffer, "\t%q\n", path)
		}
		buffer.WriteString(")\n\n")
		for _, sub := range subs {
			fmt.Fprintf(&buffer, "func wire%sTo%s(c *%s, p *%s) {\n\tc.%s = p\n}\n\n",
				sub.child, sub.parent, sub.child, sub.parent, sub.embedField)
		}
		buffer.WriteString("func init() {\n")
		for _, sub := range subs {
			fmt.Fprintf(&buffer, "\t%s.AttachWire(wire%sTo%s)\n", sub.varName, sub.child, sub.parent)
		}
		buffer.WriteString("}\n\n")
		for _, sub := range subs {
			receiver := lowerFirst(sub.parent)
			fmt.Fprintf(&buffer, "func (%s *%s) %s(args %s) (*%s, error) {\n\treturn %s.From(%s, args)\n}\n\n",
				receiver, sub.parent, sub.name, sub.argsType, sub.child, sub.varName, receiver)
			fmt.Fprintf(&buffer, "func (%s *%s) %sCtx(ctx context.Context, args %s) (*%s, error) {\n\treturn %s.FromCtx(ctx, %s, args)\n}\n\n",
				receiver, sub.parent, sub.name, sub.argsType, sub.child, sub.varName, receiver)
		}
	}
	formatted, err := format.Source(buffer.Bytes())
	if err != nil {
		return nil, fmt.Errorf("generated source does not format: %v", err)
	}
	return formatted, nil
}

// lowerFirst lowercases the first rune of a type name for a receiver.
func lowerFirst(name string) string {
	if name == "" {
		return "x"
	}
	return strings.ToLower(name[:1]) + name[1:]
}

package mnemonica

import "errors"

// Sentinel errors, checkable with errors.Is. Names follow the JS core
// constants (src/constants) where a counterpart exists; statically
// impossible ones (TYPENAME_MUST_BE_A_STRING, HANDLER_MUST_BE_A_FUNCTION,
// WRONG_ARGUMENTS_USED, OPTIONS_ERROR — the config surface is typed in Go)
// have no Go sentinel. See docs/conformance.md, C8.
var (
	// ErrAlreadyDeclared is ALREADY_DECLARED: a type path was declared
	// twice in one collection.
	ErrAlreadyDeclared = errors.New("mnemonica: type already declared in collection")

	// ErrWrongTypeDefinition is WRONG_TYPE_DEFINITION: a Define/Sub target
	// is structurally wrong — the struct does not embed mnemonica.Node, a
	// subtype does not embed its parent by pointer, or a name is invalid.
	ErrWrongTypeDefinition = errors.New("mnemonica: wrong type definition")

	// ErrWrongModificationPattern is WRONG_MODIFICATION_PATTERN /
	// WRONG_INSTANCE_INVOCATION: From received a parent that is not an
	// instance of the subtype's declared parent type (C2.2, C4.1), a root
	// was asked to construct From a parent, or a subtype was asked to
	// construct without one.
	ErrWrongModificationPattern = errors.New("mnemonica: wrong modification pattern")

	// ErrNotAnInstance is the Props counterpart of WRONG_INSTANCE_INVOCATION:
	// Props was called on a value that carries no construction record.
	ErrNotAnInstance = errors.New("mnemonica: value is not a mnemonica instance")

	// ErrBlocked is the Go-port counterpart of the JS blockErrors refusal:
	// construction was attempted from an instance whose lineage carries an
	// errored instance while the constructing type's blockErrors is true
	// (C4.2). JS has no dedicated constant for this; the refusal is
	// documented behavior of the blockErrors config.
	ErrBlocked = errors.New("mnemonica: construction blocked: errored instance in the lineage")

	// ErrWrongHookType is WRONG_HOOK_TYPE: RegisterHook received a kind
	// that is not one of preCreation, postCreation, creationError (C5.3).
	ErrWrongHookType = errors.New("mnemonica: wrong hook type")

	// ErrMissingHookCallback is MISSING_HOOK_CALLBACK: RegisterHook
	// received a nil hook function (C5.3).
	ErrMissingHookCallback = errors.New("mnemonica: missing hook callback")

	// ErrWrongArgumentsUsed is WRONG_ARGUMENTS_USED: reachable only through
	// the untyped Lookup-returned Type, whose args cross as any — a wrong
	// args value fails here instead of panicking in the typed handler.
	ErrWrongArgumentsUsed = errors.New("mnemonica: wrong arguments used")
)

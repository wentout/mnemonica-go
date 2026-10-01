package mnemonica

import "fmt"

// ErroredInstance is the C4.2 error produced when a constructor fails while
// its type's blockErrors config is true. It carries the half-built instance
// so callers can inspect or recover what the constructor managed to set.
// The instance is marked errored: constructing FROM it — or from any
// instance descending from it — is refused with ErrBlocked while the
// constructing type's blockErrors is true.
type ErroredInstance struct {
	Type     Type     // the type that failed to construct
	Instance Instance // the half-built instance (handler may have set fields)
	Err      error    // the constructor's original error
}

// Error describes the errored instance and its original error.
func (e *ErroredInstance) Error() string {
	result := fmt.Sprintf("mnemonica: errored instance of %q: %v", e.Type.Path(), e.Err)
	return result
}

// Unwrap exposes the constructor's original error to errors.Is and
// errors.As.
func (e *ErroredInstance) Unwrap() error {
	result := e.Err
	return result
}

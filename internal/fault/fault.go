package fault

import "errors"

var (
	// ErrNotFound reports that the requested application resource does not exist.
	ErrNotFound = errors.New("not found")
	// ErrAlreadyExists reports that an application resource conflicts with an existing resource.
	ErrAlreadyExists = errors.New("already exists")
	// ErrConflict reports that current application state prevents an operation.
	ErrConflict = errors.New("conflict")
	// ErrInvalidInput reports that an operation received invalid input.
	ErrInvalidInput = errors.New("invalid input")
)

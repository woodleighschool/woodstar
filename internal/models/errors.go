package models

import "errors"

// Store errors are shared by model stores.
var (
	ErrNotFound      = errors.New("not found")
	ErrAlreadyExists = errors.New("already exists")
)

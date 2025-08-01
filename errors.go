package rbac

import "errors"

// Custom errors
var (
	ErrInvalidInput     = errors.New("invalid input")
	ErrNotFound         = errors.New("resource not found")
	ErrPermissionDenied = errors.New("permission denied")
)

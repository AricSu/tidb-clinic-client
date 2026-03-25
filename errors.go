package clinicapi

import (
	"errors"
	"fmt"
)

// ErrorClass describes the high-level class of an SDK error.
type ErrorClass string

const (
	// ErrInvalidRequest indicates that the request is malformed or missing
	// required inputs.
	ErrInvalidRequest ErrorClass = "invalid_request"
	// ErrAuth indicates authentication or authorization failure.
	ErrAuth ErrorClass = "auth"
	// ErrNotFound indicates that the remote resource does not exist.
	ErrNotFound ErrorClass = "not_found"
	// ErrNoData indicates that the request was valid but no relevant data was
	// available.
	ErrNoData ErrorClass = "no_data"
	// ErrTimeout indicates request or backend timeout.
	ErrTimeout ErrorClass = "timeout"
	// ErrRateLimit indicates backend throttling.
	ErrRateLimit ErrorClass = "rate_limit"
	// ErrDecode indicates response decode failure.
	ErrDecode ErrorClass = "decode"
	// ErrBackend indicates a non-retryable backend failure that does not fit a
	// more specific class.
	ErrBackend ErrorClass = "backend"
	// ErrTransient indicates a retryable transport or backend failure.
	ErrTransient ErrorClass = "transient"
)

// Error is the typed SDK error returned by Clinic client operations.
type Error struct {
	Class      ErrorClass
	Retryable  bool
	StatusCode int
	Endpoint   string
	Message    string
	Cause      error
}

// Error returns a readable error string.
func (e *Error) Error() string {
	if e == nil {
		return "<nil>"
	}
	switch {
	case e.Endpoint != "" && e.Message != "":
		return fmt.Sprintf("%s: %s", e.Endpoint, e.Message)
	case e.Message != "":
		return e.Message
	case e.Endpoint != "":
		return e.Endpoint
	default:
		return string(e.Class)
	}
}

// Unwrap exposes the underlying cause, if any.
func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

// IsRetryable reports whether err unwraps to a retryable Clinic error.
func IsRetryable(err error) bool {
	var clinicErr *Error
	if !errors.As(err, &clinicErr) || clinicErr == nil {
		return false
	}
	return clinicErr.Retryable
}

// ClassOf returns the ErrorClass for err, if err unwraps to a Clinic error.
func ClassOf(err error) ErrorClass {
	var clinicErr *Error
	if !errors.As(err, &clinicErr) || clinicErr == nil {
		return ""
	}
	return clinicErr.Class
}

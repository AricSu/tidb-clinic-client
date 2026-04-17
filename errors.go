package clinicapi

import "github.com/AricSu/tidb-clinic-client/internal/model"

type (
	ErrorClass = model.ErrorClass
	Error      = model.Error
)

const (
	ErrInvalidRequest = model.ErrInvalidRequest
	ErrUnsupported    = model.ErrUnsupported
	ErrAuth           = model.ErrAuth
	ErrNotFound       = model.ErrNotFound
	ErrNoData         = model.ErrNoData
	ErrTimeout        = model.ErrTimeout
	ErrRateLimit      = model.ErrRateLimit
	ErrDecode         = model.ErrDecode
	ErrBackend        = model.ErrBackend
	ErrTransient      = model.ErrTransient
)

func IsRetryable(err error) bool {
	return model.IsRetryable(err)
}
func ClassOf(err error) ErrorClass {
	return model.ClassOf(err)
}

package fetch

import (
	"fmt"

	"github.com/KonishchevDmitry/feedsd/internal/util"
)

type HTTPStatusError struct {
	Status  int
	message string
}

func newHTTPStatusError(status int, format string, args ...any) *HTTPStatusError {
	return &HTTPStatusError{
		Status:  status,
		message: fmt.Sprintf(format, args...),
	}
}

func (e *HTTPStatusError) Error() string {
	return e.message
}

type temporaryError struct {
	error error
}

var _ util.Temporary = temporaryError{}

func makeTemporaryError(err error) temporaryError {
	return temporaryError{error: err}
}

func (e temporaryError) Temporary() bool {
	return true
}

func (e temporaryError) Error() string {
	return e.error.Error()
}

func (e temporaryError) Unwrap() error {
	return e.error
}

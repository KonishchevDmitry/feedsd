package fetch

import "github.com/KonishchevDmitry/newslib/internal/util"

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

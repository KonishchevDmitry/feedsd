package util

import (
	"errors"
)

type Temporary interface {
	Temporary() bool
}

func IsTemporaryError(err error) bool {
	for err := err; err != nil; err = errors.Unwrap(err) {
		if err, ok := err.(Temporary); ok && err.Temporary() {
			return true
		}
	}
	return false
}

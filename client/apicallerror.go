package client

import (
	"errors"
	"strings"
)

type ApiCallError []ApiCallRc

func (e ApiCallError) Error() string {
	var finalErr string
	for i, r := range e {
		finalErr += strings.TrimSpace(r.String())
		if i < len(e)-1 {
			finalErr += " next error: "
		}
	}
	return finalErr
}

// Is is a shorthand for checking all ApiCallRcs of an ApiCallError against
// a given mask.
func (e ApiCallError) Is(mask uint64) bool {
	for _, r := range e {
		if r.Is(mask) {
			return true
		}
	}

	return false
}

// IsApiCallError checks if an error is a specific type of LINSTOR error.
func IsApiCallError(err error, mask uint64) bool {
	var e ApiCallError
	ok := errors.As(err, &e)
	if !ok {
		return false
	}

	return e.Is(mask)
}

package client

import (
	"errors"
	"fmt"
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

// ApiCallRcErr is a LINSTOR return code mask that satisfies the error
// interface. Use it as the target of errors.Is to check whether an
// ApiCallError contains a return code matching the mask:
//
//	if errors.Is(err, client.ApiCallRcErr(linstor.FailNotConnected)) {
//		// handle "not connected"
//	}
//
// Refer to apiconsts.go in package linstor for the list of possible mask
// values.
type ApiCallRcErr uint64

func (c ApiCallRcErr) Error() string {
	return fmt.Sprintf("LINSTOR API return code %#018x", uint64(c))
}

// Is reports whether any of the contained ApiCallRcs matches target. It only
// matches an ApiCallRcErr target, which lets ApiCallError be used with the
// standard errors.Is helper.
func (e ApiCallError) Is(target error) bool {
	mask, ok := target.(ApiCallRcErr)
	if !ok {
		return false
	}

	for _, r := range e {
		if r.Is(uint64(mask)) {
			return true
		}
	}

	return false
}

// IsApiCallError checks if an error is a specific type of LINSTOR error.
// It is a convenience wrapper around errors.Is(err, ApiCallRcErr(mask)).
func IsApiCallError(err error, mask uint64) bool {
	return errors.Is(err, ApiCallRcErr(mask))
}

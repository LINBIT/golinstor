package client

import (
	"encoding/json"
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

// maxErrorBodyKept limits how much of an unexpected error body is kept in an
// UnexpectedErrorBody, so that e.g. an HTML error page does not flood logs.
const maxErrorBodyKept = 4096

// maxErrorBodyRead limits how much of an error response body is read at all. A
// (hypothetical) ApiCallRc list larger than this is cut and therefore returned
// as an UnexpectedErrorBody.
const maxErrorBodyRead = 1 << 20

// genericErrorRetCode is linstor.MaskError (0xC000000000000000) as the signed
// value of ApiCallRc.RetCode. It marks an ApiCallRc as an error for messages
// that came without a LINSTOR return code.
const genericErrorRetCode = -0x4000000000000000

// UnexpectedErrorBody is returned for an HTTP error response whose body is
// neither a list of ApiCallRc nor one of the known object forms. The body is
// kept (truncated to maxErrorBodyKept bytes) so that the caller still sees
// what the server or a proxy in front of it sent.
type UnexpectedErrorBody struct {
	StatusCode int
	Body       string
}

func (e *UnexpectedErrorBody) Error() string {
	return fmt.Sprintf("unexpected error response (HTTP %d): %s", e.StatusCode, e.Body)
}

// decodeErrorBody turns the body of an HTTP error response into an error.
// The controller usually sends a list of ApiCallRc, but some endpoints send
// an object: a failed clone sends a ResourceDefinitionCloneStarted with the
// ApiCallRcs in "messages", others send {"message": ...} or {"error": ...}.
// Proxies may send anything. The controller's information must never be lost
// to a JSON decoding error.
func decodeErrorBody(statusCode int, body []byte) error {
	var list ApiCallError
	if err := json.Unmarshal(body, &list); err == nil && len(list) > 0 {
		return list
	}

	var obj struct {
		Messages ApiCallError `json:"messages"`
		RetCode  *int64       `json:"ret_code"`
		Message  string       `json:"message"`
		Error    string       `json:"error"`
	}
	if err := json.Unmarshal(body, &obj); err == nil {
		switch {
		case len(obj.Messages) > 0:
			return obj.Messages
		case obj.RetCode != nil:
			// a single ApiCallRc, as the OpenAPI spec documents for "OperationFailed"
			var rc ApiCallRc
			if err := json.Unmarshal(body, &rc); err == nil {
				return ApiCallError{rc}
			}
		case obj.Message != "":
			return ApiCallError{{RetCode: genericErrorRetCode, Message: obj.Message}}
		case obj.Error != "":
			return ApiCallError{{RetCode: genericErrorRetCode, Message: obj.Error}}
		}
	}

	kept := body
	if len(kept) > maxErrorBodyKept {
		kept = kept[:maxErrorBodyKept]
	}
	return &UnexpectedErrorBody{StatusCode: statusCode, Body: strings.ToValidUTF8(string(kept), "")}
}

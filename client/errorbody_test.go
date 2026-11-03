package client

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	linstor "github.com/LINBIT/golinstor"
)

func TestDecodeErrorBody(t *testing.T) {
	huge := strings.Repeat("x", maxErrorBodyRead+1000)

	cases := []struct {
		descr      string
		status     int
		body       string
		expectRcs  ApiCallError
		expectBody *UnexpectedErrorBody
	}{{
		descr:     "list, as before",
		status:    500,
		body:      `[{"ret_code":-4611686018427386906,"message":"boom"}]`,
		expectRcs: ApiCallError{{RetCode: -4611686018427386906, Message: "boom"}},
	}, {
		descr:  "clone started object with messages",
		status: 500,
		body: `{"location":"/v1/resource-definitions/a/clone/b","source_name":"a","clone_name":"b",` +
			`"messages":[{"ret_code":-4611686018427386906,"message":"boom","error_report_ids":["X"]}]}`,
		expectRcs: ApiCallError{{RetCode: -4611686018427386906, Message: "boom", ErrorReportIds: []string{"X"}}},
	}, {
		descr:     "message object",
		status:    503,
		body:      `{"message": "Satellite not connected"}`,
		expectRcs: ApiCallError{{RetCode: genericErrorRetCode, Message: "Satellite not connected"}},
	}, {
		descr:     "error object",
		status:    400,
		body:      `{"error":"'nodes' must be a non-empty list"}`,
		expectRcs: ApiCallError{{RetCode: genericErrorRetCode, Message: "'nodes' must be a non-empty list"}},
	}, {
		descr:     "single ApiCallRc object keeps its return code",
		status:    500,
		body:      `{"ret_code":-4611686018427386906,"message":"boom","cause":"why"}`,
		expectRcs: ApiCallError{{RetCode: -4611686018427386906, Message: "boom", Cause: "why"}},
	}, {
		descr:     "messages win over a top-level message",
		status:    500,
		body:      `{"message":"outer","messages":[{"ret_code":-4611686018427386906,"message":"inner"}]}`,
		expectRcs: ApiCallError{{RetCode: -4611686018427386906, Message: "inner"}},
	}, {
		descr:      "null",
		status:     500,
		body:       `null`,
		expectBody: &UnexpectedErrorBody{StatusCode: 500, Body: `null`},
	}, {
		descr:      "list of non-objects",
		status:     500,
		body:       `["boom"]`,
		expectBody: &UnexpectedErrorBody{StatusCode: 500, Body: `["boom"]`},
	}, {
		descr:      "empty list",
		status:     500,
		body:       `[]`,
		expectBody: &UnexpectedErrorBody{StatusCode: 500, Body: `[]`},
	}, {
		descr:      "html from a proxy",
		status:     502,
		body:       `<html>502 Bad Gateway</html>`,
		expectBody: &UnexpectedErrorBody{StatusCode: 502, Body: `<html>502 Bad Gateway</html>`},
	}, {
		descr:      "empty body",
		status:     500,
		body:       ``,
		expectBody: &UnexpectedErrorBody{StatusCode: 500, Body: ``},
	}, {
		descr:      "huge body is kept truncated to 4 KiB",
		status:     500,
		body:       huge,
		expectBody: &UnexpectedErrorBody{StatusCode: 500, Body: huge[:maxErrorBodyKept]},
	}}

	for _, c := range cases {
		t.Run(c.descr, func(t *testing.T) {
			err := decodeErrorBody(c.status, []byte(c.body))
			assert.Error(t, err)

			if c.expectRcs != nil {
				var rcs ApiCallError
				assert.True(t, errors.As(err, &rcs), "expected an ApiCallError, got %T", err)
				assert.Equal(t, c.expectRcs, rcs)
				for _, rc := range rcs {
					assert.Equal(t, uint64(linstor.MaskError), uint64(rc.RetCode)&linstor.MaskError, "error bits must be set")
				}
			} else {
				var ueb *UnexpectedErrorBody
				assert.True(t, errors.As(err, &ueb), "expected an UnexpectedErrorBody, got %T", err)
				assert.Equal(t, c.expectBody, ueb)
			}
		})
	}
}

func TestCloneErrorKeepsControllerMessage(t *testing.T) {
	// what the controller answers for a failed clone (ResourceDefinitions.mapToCloneStarted)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"location":"/v1/resource-definitions/src/clone/dst","source_name":"src",` +
			`"clone_name":"dst","messages":[{"ret_code":-4611686018427386906,` +
			`"message":"(node B) Failed to access DRBD super-block of volume dst/0"}]}`))
	}))
	defer srv.Close()

	u, err := url.Parse(srv.URL)
	assert.NoError(t, err)
	cl, err := NewClient(HTTPClient(srv.Client()), BaseURL(u))
	assert.NoError(t, err)

	_, err = cl.ResourceDefinitions.Clone(context.Background(), "src", ResourceDefinitionCloneRequest{Name: "dst"})

	var rcs ApiCallError
	assert.True(t, errors.As(err, &rcs), "expected an ApiCallError, got %T: %v", err, err)
	assert.Contains(t, err.Error(), "Failed to access DRBD super-block of volume dst/0")
}

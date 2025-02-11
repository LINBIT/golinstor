package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

const TestCaCert = `-----BEGIN CERTIFICATE-----
MIIB1DCCAXqgAwIBAgIUZJyvXb6pJ9tHxKOlVkjTaqjpAAIwCgYIKoZIzj0EAwIw
SDELMAkGA1UEBhMCVVMxCzAJBgNVBAgTAkNBMRYwFAYDVQQHEw1TYW4gRnJhbmNp
c2NvMRQwEgYDVQQDEwtleGFtcGxlLm5ldDAeFw0yMDA1MDgxMjUyMDBaFw0yNTA1
MDcxMjUyMDBaMEgxCzAJBgNVBAYTAlVTMQswCQYDVQQIEwJDQTEWMBQGA1UEBxMN
U2FuIEZyYW5jaXNjbzEUMBIGA1UEAxMLZXhhbXBsZS5uZXQwWTATBgcqhkjOPQIB
BggqhkjOPQMBBwNCAAT7uWd8BeFIcw64pRJheVh6tKrsqSLF4z9LAQKEaH5pg34+
06T2Ed7hKUSca3R8zEuP9EZcHNpYXKoeuF1QjZt5o0IwQDAOBgNVHQ8BAf8EBAMC
AQYwDwYDVR0TAQH/BAUwAwEB/zAdBgNVHQ4EFgQUyK850E3ZE9Jb3JPDq3BtttXd
wE0wCgYIKoZIzj0EAwIDSAAwRQIge+3tsEmO/WbjxmQoA+kDoSpOQDnkDckqvEXD
1H939HUCIQDHp0oAvI1sMM0ksAl7D0Bpxjtha2kpzbqsAf4yDy9rWw==
-----END CERTIFICATE-----
`

func TestNewClient_ViaEnv(t *testing.T) {
	testcases := []struct {
		name        string
		env         map[string]string
		expectedUrl string
		hasError    bool
	}{
		{
			name:        "default",
			expectedUrl: "http://localhost:3370",
		},
		{
			name:        "just-domain-name",
			env:         map[string]string{"LS_CONTROLLERS": "just.domain"},
			expectedUrl: "http://just.domain:3370",
		},
		{
			name:        "linstor-protocol",
			env:         map[string]string{"LS_CONTROLLERS": "linstor://just.domain"},
			expectedUrl: "http://just.domain:3370",
		},
		{
			name:        "linstor-ssl-protocol",
			env:         map[string]string{"LS_CONTROLLERS": "linstor+ssl://just.domain"},
			expectedUrl: "https://just.domain:3371",
		},
		{
			name:        "just-domain-with-port",
			env:         map[string]string{"LS_CONTROLLERS": "just.domain:4000"},
			expectedUrl: "http://just.domain:4000",
		},
		{
			name:        "domain-with-protocol",
			env:         map[string]string{"LS_CONTROLLERS": "http://just.domain"},
			expectedUrl: "http://just.domain:3370",
		},
		{
			name:        "just-domain-with-https-protocol",
			env:         map[string]string{"LS_CONTROLLERS": "https://just.domain"},
			expectedUrl: "https://just.domain:3371",
		},
		{
			name:        "just-domain-with-client-secrets",
			env:         map[string]string{"LS_CONTROLLERS": "just.domain", "LS_ROOT_CA": TestCaCert},
			expectedUrl: "https://just.domain:3371",
		},
		{
			name:        "just-domain-with-client-secrets-and-port",
			env:         map[string]string{"LS_CONTROLLERS": "just.domain:4000", "LS_ROOT_CA": TestCaCert},
			expectedUrl: "https://just.domain:4000",
		},
		{
			name:     "parse-error-multi-scheme",
			env:      map[string]string{"LS_CONTROLLERS": "https://http://just.domain:4000"},
			hasError: true,
		},
		{
			name:     "parse-error-multi-port",
			env:      map[string]string{"LS_CONTROLLERS": "https://just.domain:4000:5000"},
			hasError: true,
		},
		{
			name:     "parse-error-inconsistent-env",
			env:      map[string]string{"LS_CONTROLLERS": "https://just.domain:4000", "LS_USER_CERTIFICATE": "stuff"},
			hasError: true,
		},
		{
			name:     "parse-error-inconsistent-env-other",
			env:      map[string]string{"LS_CONTROLLERS": "https://just.domain:4000", "LS_USER_KEY": "stuff"},
			hasError: true,
		},
	}

	for _, item := range testcases {
		test := item
		t.Run(test.name, func(t *testing.T) {
			os.Clearenv()
			defer os.Clearenv()
			for k, v := range test.env {
				_ = os.Setenv(k, v)
			}

			actual, err := NewClient()

			if actual == nil {
				if !test.hasError {
					t.Errorf("expected no error, got error: %v", err)
				}
				return
			}

			if test.expectedUrl != actual.BaseURL().String() {
				t.Errorf("expected url: %v, got url: %v", test.expectedUrl, actual.BaseURL().String())
			}
		})
	}
}

func fakeVersionHandler(version string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintf(w, `{"version":"%s"}`, version)
	})
}

func TestBaseURLFailover(t *testing.T) {
	first := httptest.NewServer(fakeVersionHandler("first"))
	second := httptest.NewServer(fakeVersionHandler("second"))
	third := httptest.NewTLSServer(fakeVersionHandler("third"))

	defer first.Close()
	defer second.Close()
	defer third.Close()

	firstUrl := url.URL{
		Scheme: "http",
		Host:   first.Listener.Addr().String(),
	}
	secondUrl := url.URL{
		Scheme: "http",
		Host:   second.Listener.Addr().String(),
	}
	thirdUrl := url.URL{
		Scheme: "https",
		Host:   third.Listener.Addr().String(),
	}

	client, err := NewClient(
		BaseURL(&firstUrl, &secondUrl, &thirdUrl),
		HTTPClient(third.Client()),
	)
	assert.NoError(t, err)

	// Take the first URL as is.
	version, err := client.Controller.GetVersion(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, "first", version.Version)
	assert.Equal(t, &firstUrl, client.BaseURL())

	// Stop the first server -> should fail-over to second or third.
	first.Close()

	version, err = client.Controller.GetVersion(context.Background())
	assert.NoError(t, err)
	switch version.Version {
	case "second":
		second.Close()
		assert.Equal(t, &secondUrl, client.BaseURL())

		version, err = client.Controller.GetVersion(context.Background())
		assert.NoError(t, err)
		assert.Equal(t, "third", version.Version)
		assert.Equal(t, &thirdUrl, client.BaseURL())
		third.Close()
	case "third":
		third.Close()
		assert.Equal(t, &thirdUrl, client.BaseURL())

		version, err = client.Controller.GetVersion(context.Background())
		assert.NoError(t, err)
		assert.Equal(t, "second", version.Version)
		assert.Equal(t, &secondUrl, client.BaseURL())
		second.Close()
	default:
		t.Fatalf("unexpected version: %s", version.Version)
	}

	_, err = client.Controller.GetVersion(context.Background())
	assert.Error(t, err)
}

func TestBearerTokenOpt(t *testing.T) {
	const Token = "AbCdEfg1234567890"
	var FakeVersion = ControllerVersion{
		BuildTime:      "#buildtime",
		GitHash:        "#git",
		RestApiVersion: "#rest",
		Version:        "#v1",
	}

	fakeHttpServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Header.Get("Authorization") != "Bearer "+Token {
			writer.WriteHeader(http.StatusUnauthorized)
		} else {
			writer.WriteHeader(http.StatusOK)
			enc := json.NewEncoder(writer)
			_ = enc.Encode(&FakeVersion)
		}
	}))
	defer fakeHttpServer.Close()

	u, err := url.Parse(fakeHttpServer.URL)
	assert.NoError(t, err)

	cl, err := NewClient(BearerToken(Token), HTTPClient(fakeHttpServer.Client()), BaseURL(u))
	assert.NoError(t, err)

	actualVersion, err := cl.Controller.GetVersion(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, FakeVersion, actualVersion)
}

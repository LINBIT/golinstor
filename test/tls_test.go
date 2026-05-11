package test

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/LINBIT/golinstor/client"
)

// generateCert creates a self-signed ECDSA certificate with the given serial number.
func generateCert(t *testing.T, serial int64) (certPEM, keyPEM []byte) {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	template := &x509.Certificate{
		SerialNumber: big.NewInt(serial),
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)

	certPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	keyDER, err := x509.MarshalECPrivateKey(key)
	require.NoError(t, err)

	keyPEM = pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})

	return
}

// TestDynamicTls verifies that when client cert files are updated on disk, the next
// connection uses the new certificate without recreating the client.
func TestDynamicTls(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := client.ControllerVersion{Version: "just-for-test"}

		if r.TLS != nil && len(r.TLS.PeerCertificates) > 0 {
			response.BuildTime = r.TLS.PeerCertificates[0].SerialNumber.String()
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(response)
	})

	// RequireAnyClientCert asks for a client cert but skips CA verification, so
	// our self-signed test certs are accepted without a shared CA chain.
	ts := httptest.NewUnstartedServer(handler)
	ts.Config.SetKeepAlivesEnabled(false) // force new TCP+TLS connection per request
	ts.TLS = &tls.Config{ClientAuth: tls.RequireAnyClientCert}
	ts.StartTLS()
	defer ts.Close()

	// The httptest server cert is self-signed, so it doubles as its own root CA.
	serverCertPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: ts.TLS.Certificates[0].Certificate[0],
	})

	tmpDir := t.TempDir()
	caCertPath := filepath.Join(tmpDir, "ca.pem")
	certPath := filepath.Join(tmpDir, "client-cert.pem")
	keyPath := filepath.Join(tmpDir, "client-key.pem")

	require.NoError(t, os.WriteFile(caCertPath, serverCertPEM, 0o600))

	cert1PEM, key1PEM := generateCert(t, 1)
	require.NoError(t, os.WriteFile(certPath, cert1PEM, 0o600))
	require.NoError(t, os.WriteFile(keyPath, key1PEM, 0o600))

	t.Setenv(client.RootCAFileEnv, caCertPath)
	t.Setenv(client.UserCertFileEnv, certPath)
	t.Setenv(client.UserKeyFileEnv, keyPath)

	tsURL, err := url.Parse(ts.URL)
	require.NoError(t, err)

	lc, err := client.NewClient(client.BaseURL(tsURL))
	require.NoError(t, err)

	version, err := lc.Controller.GetVersion(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "1", version.BuildTime)

	// Overwrite the cert files — the client must pick up the new cert on the
	// next connection because GetClientCertificate re-reads the files.
	cert2PEM, key2PEM := generateCert(t, 2)
	require.NoError(t, os.WriteFile(certPath, cert2PEM, 0o600))
	require.NoError(t, os.WriteFile(keyPath, key2PEM, 0o600))

	version, err = lc.Controller.GetVersion(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "2", version.BuildTime)
}

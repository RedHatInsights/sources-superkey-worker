package sources

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/redhatinsights/sources-superkey-worker/config"
)

func TestBuildHTTPClient_HTTP_ReturnsDefaultClient(t *testing.T) {
	cfg := &config.SuperKeyWorkerConfig{
		SourcesScheme: "http",
	}

	client := buildHTTPClient(cfg)

	if client != http.DefaultClient {
		t.Error("expected http.DefaultClient for http scheme")
	}
}

func TestBuildHTTPClient_HTTPS_ReturnsTLSClient(t *testing.T) {
	cfg := &config.SuperKeyWorkerConfig{
		SourcesScheme: "https",
	}

	client := buildHTTPClient(cfg)

	if client == http.DefaultClient {
		t.Error("expected custom client for https scheme, got DefaultClient")
	}

	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatal("expected *http.Transport")
	}

	if transport.TLSClientConfig == nil {
		t.Fatal("expected TLS config to be set")
	}

	if transport.TLSClientConfig.MinVersion != tls.VersionTLS12 {
		t.Errorf("expected TLS 1.2 minimum, got %d", transport.TLSClientConfig.MinVersion)
	}
}

func TestBuildHTTPClient_HTTPS_WithCAPath(t *testing.T) {
	// Generate a self-signed CA certificate for testing.
	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate CA key: %s", err)
	}

	caTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "Test CA"},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(time.Hour),
		IsCA:         true,
		KeyUsage:     x509.KeyUsageCertSign,
	}

	caCertDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		t.Fatalf("failed to create CA cert: %s", err)
	}

	caFile := filepath.Join(t.TempDir(), "ca.pem")
	caPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caCertDER})
	if err := os.WriteFile(caFile, caPEM, 0600); err != nil {
		t.Fatalf("failed to write CA file: %s", err)
	}

	cfg := &config.SuperKeyWorkerConfig{
		SourcesScheme:    "https",
		SourcesTLSCAPath: caFile,
	}

	client := buildHTTPClient(cfg)

	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatal("expected *http.Transport")
	}

	if transport.TLSClientConfig.RootCAs == nil {
		t.Error("expected custom RootCAs to be set when CA path is provided")
	}
}

func TestBuildHTTPClient_HTTPS_InvalidCAPath(t *testing.T) {
	cfg := &config.SuperKeyWorkerConfig{
		SourcesScheme:    "https",
		SourcesTLSCAPath: "/nonexistent/ca.pem",
	}

	// Should not panic; falls back to system CA pool.
	client := buildHTTPClient(cfg)

	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatal("expected *http.Transport")
	}

	// RootCAs should be nil (system default) since the file doesn't exist.
	if transport.TLSClientConfig.RootCAs != nil {
		t.Error("expected nil RootCAs for invalid CA path (fallback to system pool)")
	}
}

func TestBuildHTTPClient_HTTPS_ClonesDefaultTransport(t *testing.T) {
	cfg := &config.SuperKeyWorkerConfig{
		SourcesScheme: "https",
	}

	client := buildHTTPClient(cfg)

	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatal("expected *http.Transport")
	}

	defaultTransport, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		t.Fatal("expected http.DefaultTransport to be *http.Transport")
	}

	// Verify cloned transport inherits key settings from DefaultTransport.
	if transport.MaxIdleConns != defaultTransport.MaxIdleConns {
		t.Errorf("expected MaxIdleConns %d, got %d", defaultTransport.MaxIdleConns, transport.MaxIdleConns)
	}

	if transport.IdleConnTimeout != defaultTransport.IdleConnTimeout {
		t.Errorf("expected IdleConnTimeout %v, got %v", defaultTransport.IdleConnTimeout, transport.IdleConnTimeout)
	}
}

func TestNewSourcesClient_SetsHTTPClient(t *testing.T) {
	cfg := &config.SuperKeyWorkerConfig{
		SourcesScheme: "https",
		SourcesHost:   "localhost",
		SourcesPort:   8000,
	}

	sc := NewSourcesClient(cfg)

	if sc.httpClient == nil {
		t.Error("expected httpClient to be set")
	}

	if sc.httpClient == http.DefaultClient {
		t.Error("expected custom client for https scheme")
	}
}

func TestNewSourcesClient_HTTP_UsesDefaultClient(t *testing.T) {
	cfg := &config.SuperKeyWorkerConfig{
		SourcesScheme: "http",
		SourcesHost:   "localhost",
		SourcesPort:   8000,
	}

	sc := NewSourcesClient(cfg)

	if sc.httpClient != http.DefaultClient {
		t.Error("expected http.DefaultClient for http scheme")
	}
}

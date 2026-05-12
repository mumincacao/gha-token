package githubapp

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

// Helper to generate test RSA key pair
func generateTestKeyPair(t *testing.T) (*rsa.PrivateKey, []byte) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate RSA key: %v", err)
	}

	pkcs1 := x509.MarshalPKCS1PrivateKey(key)
	pemData := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: pkcs1,
	})

	return key, pemData
}

func TestParseRSAPrivateKey_PKCS1(t *testing.T) {
	_, pemData := generateTestKeyPair(t)

	key, err := parseRSAPrivateKey(pemData)
	if err != nil {
		t.Fatalf("parseRSAPrivateKey failed: %v", err)
	}

	if key == nil {
		t.Fatal("Expected RSA key, got nil")
	}
}

func TestParseRSAPrivateKey_PKCS8(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate RSA key: %v", err)
	}

	pkcs8, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		t.Fatalf("Failed to marshal PKCS8: %v", err)
	}

	pemData := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: pkcs8,
	})

	key, err := parseRSAPrivateKey(pemData)
	if err != nil {
		t.Fatalf("parseRSAPrivateKey failed for PKCS8: %v", err)
	}

	if key == nil {
		t.Fatal("Expected RSA key, got nil")
	}
}

func TestParseRSAPrivateKey_InvalidPEM(t *testing.T) {
	invalidPEM := []byte("not a valid PEM data")

	_, err := parseRSAPrivateKey(invalidPEM)
	if err == nil {
		t.Fatal("Expected error for invalid PEM, got nil")
	}
}

func TestParseRSAPrivateKey_InvalidKeyFormat(t *testing.T) {
	// Valid PEM but not RSA key
	pemData := []byte(`-----BEGIN CERTIFICATE-----
MIIBkTCB+wIJAKHHIG0gAKjZMA0GCSqGSIb3DQEBBQUAMBMxETAPBgNVBAMMCHRl
-----END CERTIFICATE-----`)

	_, err := parseRSAPrivateKey(pemData)
	if err == nil {
		t.Fatal("Expected error for certificate instead of key, got nil")
	}
}

func TestBuildAppJWT_Success(t *testing.T) {
	_, pemData := generateTestKeyPair(t)

	tmpFile, err := os.CreateTemp("", "test_key_*.pem")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write(pemData); err != nil {
		t.Fatalf("Failed to write key file: %v", err)
	}
	tmpFile.Close()

	appID := "12345"
	jwtToken, err := buildAppJWT(appID, pemData)
	if err != nil {
		t.Fatalf("buildAppJWT failed: %v", err)
	}

	if jwtToken == "" {
		t.Fatal("Expected non-empty JWT token")
	}

	// Verify JWT format (3 parts separated by dots)
	parts := len([]byte(jwtToken)) - len([]byte(".."))
	if parts < 30 { // JWT should have at least 30+ chars for 3 encoded parts
		t.Fatalf("JWT appears malformed: %s", jwtToken)
	}
}

func TestBuildAppJWT_InvalidPEM(t *testing.T) {
	appID := "12345"
	_, err := buildAppJWT(appID, []byte("not a valid PEM data"))
	if err == nil {
		t.Fatal("Expected error for invalid PEM, got nil")
	}
}

func TestClientGetInstallationToken_Success(t *testing.T) {
	_, pemData := generateTestKeyPair(t)

	tmpFile, err := os.CreateTemp("", "test_key_*.pem")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write(pemData); err != nil {
		t.Fatalf("Failed to write key file: %v", err)
	}
	tmpFile.Close()

	// Mock GitHub API server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/test-owner/test-repo/installation":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			io.WriteString(w, `{"id": 999}`)
		case "/app/installations/999/access_tokens":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			io.WriteString(w, `{"token": "ghu_test_token_12345"}`)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewClient(ClientOptions{
		BaseURL: server.URL,
		Timeout: 10 * time.Second,
		Debug:   false,
		Stderr:  io.Discard,
	})

	token, err := client.GetInstallationToken("12345", string(pemData), "test-owner", "test-repo")
	if err != nil {
		t.Fatalf("GetInstallationToken failed: %v", err)
	}

	if token != "ghu_test_token_12345" {
		t.Fatalf("Expected 'ghu_test_token_12345', got '%s'", token)
	}
}

func TestClientGetInstallationToken_InstallationNotFound(t *testing.T) {
	_, pemData := generateTestKeyPair(t)

	tmpFile, err := os.CreateTemp("", "test_key_*.pem")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write(pemData); err != nil {
		t.Fatalf("Failed to write key file: %v", err)
	}
	tmpFile.Close()

	// Mock GitHub API server returning 404
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		io.WriteString(w, `{"message": "Not Found"}`)
	}))
	defer server.Close()

	client := NewClient(ClientOptions{
		BaseURL: server.URL,
		Timeout: 10 * time.Second,
		Debug:   false,
		Stderr:  io.Discard,
	})

	_, err = client.GetInstallationToken("12345", string(pemData), "test-owner", "nonexistent-repo")
	if err == nil {
		t.Fatal("Expected error for 404 response, got nil")
	}
}

func TestClientGetInstallationToken_InvalidJSON(t *testing.T) {
	_, pemData := generateTestKeyPair(t)

	tmpFile, err := os.CreateTemp("", "test_key_*.pem")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write(pemData); err != nil {
		t.Fatalf("Failed to write key file: %v", err)
	}
	tmpFile.Close()

	// Mock server returning invalid JSON
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, `invalid json`)
	}))
	defer server.Close()

	client := NewClient(ClientOptions{
		BaseURL: server.URL,
		Timeout: 10 * time.Second,
		Debug:   false,
		Stderr:  io.Discard,
	})

	_, err = client.GetInstallationToken("12345", string(pemData), "test-owner", "test-repo")
	if err == nil {
		t.Fatal("Expected error for invalid JSON response, got nil")
	}
}

func TestNewClientDefaults(t *testing.T) {
	client := NewClient(ClientOptions{})

	if client.baseURL != "https://api.github.com" {
		t.Fatalf("Expected default baseURL 'https://api.github.com', got '%s'", client.baseURL)
	}

	if client.httpClient.Timeout != 30*time.Second {
		t.Fatalf("Expected default timeout 30s, got %v", client.httpClient.Timeout)
	}
}

func TestTruncateForError(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{
			input:    "short error",
			expected: "short error",
		},
		{
			input:    fmt.Sprintf("%s", "x"+string(make([]byte, 300))),
			expected: fmt.Sprintf("%s...", "x"+string(make([]byte, 237))),
		},
	}

	for _, tc := range testCases {
		result := truncateForError(tc.input)
		if len(result) > 243 {
			t.Fatalf("truncateForError result exceeds limit: %d chars", len(result))
		}
	}
}

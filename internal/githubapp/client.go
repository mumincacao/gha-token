package githubapp

import (
	"bytes"
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type ClientOptions struct {
	BaseURL string
	Timeout time.Duration
	Debug   bool
	Stderr  io.Writer
}

type Client struct {
	baseURL    string
	httpClient *http.Client
	debug      bool
	stderr     io.Writer
}

type installationResponse struct {
	ID int64 `json:"id"`
}

type accessTokenResponse struct {
	Token string `json:"token"`
}

func NewClient(opts ClientOptions) *Client {
	baseURL := strings.TrimRight(opts.BaseURL, "/")
	if baseURL == "" {
		baseURL = "https://api.github.com"
	}

	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	stderr := opts.Stderr
	if stderr == nil {
		stderr = os.Stderr
	}

	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		debug:  opts.Debug,
		stderr: stderr,
	}
}

func (c *Client) GetInstallationToken(appID, privateKeyPath, owner, repository string) (string, error) {
	ctx := context.Background()

	jwtToken, err := buildAppJWT(appID, privateKeyPath)
	if err != nil {
		return "", err
	}

	installationID, err := c.getInstallationID(ctx, jwtToken, owner, repository)
	if err != nil {
		return "", err
	}

	token, err := c.createInstallationToken(ctx, jwtToken, installationID)
	if err != nil {
		return "", err
	}

	return token, nil
}

func buildAppJWT(appID, privateKeyPath string) (string, error) {
	keyBytes, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return "", fmt.Errorf("failed to read private key file: %w", err)
	}

	privateKey, err := parseRSAPrivateKey(keyBytes)
	if err != nil {
		return "", err
	}

	now := time.Now().UTC()
	claims := jwt.MapClaims{
		"iss": appID,
		"iat": now.Add(-30 * time.Second).Unix(),
		"exp": now.Add(9 * time.Minute).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	signed, err := token.SignedString(privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign JWT: %w", err)
	}

	return signed, nil
}

func parseRSAPrivateKey(data []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New("invalid PEM data in private key file")
	}

	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}

	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key (PKCS1/PKCS8): %w", err)
	}

	key, ok := parsed.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("private key is not an RSA key")
	}
	return key, nil
}

func (c *Client) getInstallationID(ctx context.Context, jwtToken, owner, repository string) (int64, error) {
	endpoint := c.apiURL(path.Join("repos", owner, repository, "installation"))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to build installation request: %w", err)
	}
	addAuthHeaders(req, jwtToken)

	var result installationResponse
	status, body, err := c.doJSON(req, &result)
	if err != nil {
		return 0, err
	}

	if status != http.StatusOK {
		return 0, fmt.Errorf("failed to resolve installation for %s/%s: status=%d body=%s", owner, repository, status, truncateForError(body))
	}
	if result.ID == 0 {
		return 0, errors.New("installation id not found in GitHub response")
	}

	return result.ID, nil
}

func (c *Client) createInstallationToken(ctx context.Context, jwtToken string, installationID int64) (string, error) {
	endpoint := c.apiURL(path.Join("app", "installations", fmt.Sprintf("%d", installationID), "access_tokens"))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewBufferString("{}"))
	if err != nil {
		return "", fmt.Errorf("failed to build access token request: %w", err)
	}
	addAuthHeaders(req, jwtToken)
	req.Header.Set("Content-Type", "application/json")

	var result accessTokenResponse
	status, body, err := c.doJSON(req, &result)
	if err != nil {
		return "", err
	}

	if status != http.StatusCreated {
		return "", fmt.Errorf("failed to create installation access token: status=%d body=%s", status, truncateForError(body))
	}
	if result.Token == "" {
		return "", errors.New("access token not found in GitHub response")
	}

	return result.Token, nil
}

func (c *Client) doJSON(req *http.Request, out interface{}) (int, string, error) {
	if c.debug {
		fmt.Fprintf(c.stderr, "[debug] %s %s\n", req.Method, req.URL.String())
	}

	res, err := c.httpClient.Do(req)
	if err != nil {
		return 0, "", fmt.Errorf("request failed: %w", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(io.LimitReader(res.Body, 64*1024))
	if err != nil {
		return 0, "", fmt.Errorf("failed to read response body: %w", err)
	}

	if c.debug {
		fmt.Fprintf(c.stderr, "[debug] status=%d\n", res.StatusCode)
	}

	if out != nil && len(body) > 0 {
		if err := json.Unmarshal(body, out); err != nil {
			return res.StatusCode, string(body), fmt.Errorf("failed to parse GitHub API response: %w", err)
		}
	}

	return res.StatusCode, string(body), nil
}

func (c *Client) apiURL(p string) string {
	return c.baseURL + "/" + strings.TrimLeft(p, "/")
}

func addAuthHeaders(req *http.Request, jwtToken string) {
	req.Header.Set("Authorization", "Bearer "+jwtToken)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
}

func truncateForError(body string) string {
	const maxLen = 240
	b := strings.TrimSpace(body)
	if len(b) <= maxLen {
		return b
	}
	return b[:maxLen] + "..."
}

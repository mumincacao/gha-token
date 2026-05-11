package main

import (
	"os"
	"testing"
	"time"
)

func TestParseFlagsRequired(t *testing.T) {
	testCases := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "all flags provided",
			args:    []string{"prog", "--app-id", "123", "--private-key-path", "key.pem", "--owner", "org", "--repository", "repo"},
			wantErr: false,
		},
		{
			name:    "missing app-id",
			args:    []string{"prog", "--private-key-path", "key.pem", "--owner", "org", "--repository", "repo"},
			wantErr: true,
		},
		{
			name:    "missing private-key-path",
			args:    []string{"prog", "--app-id", "123", "--owner", "org", "--repository", "repo"},
			wantErr: true,
		},
		{
			name:    "missing owner",
			args:    []string{"prog", "--app-id", "123", "--private-key-path", "key.pem", "--repository", "repo"},
			wantErr: true,
		},
		{
			name:    "missing repository",
			args:    []string{"prog", "--app-id", "123", "--private-key-path", "key.pem", "--owner", "org"},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			os.Args = tc.args
			_, err := parseFlags()
			if (err != nil) != tc.wantErr {
				t.Fatalf("parseFlags() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestParseFlagsTimeout(t *testing.T) {
	testCases := []struct {
		name        string
		timeoutArg  string
		wantErr     bool
		wantTimeout time.Duration
	}{
		{
			name:        "default timeout",
			timeoutArg:  "",
			wantErr:     false,
			wantTimeout: 30 * time.Second,
		},
		{
			name:        "custom timeout",
			timeoutArg:  "60",
			wantErr:     false,
			wantTimeout: 60 * time.Second,
		},
		{
			name:       "invalid timeout",
			timeoutArg: "abc",
			wantErr:    true,
		},
		{
			name:       "negative timeout",
			timeoutArg: "-5",
			wantErr:    true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			args := []string{"prog", "--app-id", "123", "--private-key-path", "key.pem", "--owner", "org", "--repository", "repo"}
			if tc.timeoutArg != "" {
				args = append(args, "--timeout", tc.timeoutArg)
			}

			os.Args = args
			cfg, err := parseFlags()

			if (err != nil) != tc.wantErr {
				t.Fatalf("parseFlags() error = %v, wantErr %v", err, tc.wantErr)
			}

			if !tc.wantErr && cfg.timeout != tc.wantTimeout {
				t.Fatalf("Expected timeout %v, got %v", tc.wantTimeout, cfg.timeout)
			}
		})
	}
}

func TestParseFlagsDebug(t *testing.T) {
	os.Args = []string{"prog", "--app-id", "123", "--private-key-path", "key.pem", "--owner", "org", "--repository", "repo", "--debug"}
	cfg, err := parseFlags()

	if err != nil {
		t.Fatalf("parseFlags() error = %v", err)
	}

	if !cfg.debug {
		t.Fatal("Expected debug to be true")
	}
}

func TestParseFlagsDefaultDebug(t *testing.T) {
	os.Args = []string{"prog", "--app-id", "123", "--private-key-path", "key.pem", "--owner", "org", "--repository", "repo"}
	cfg, err := parseFlags()

	if err != nil {
		t.Fatalf("parseFlags() error = %v", err)
	}

	if cfg.debug {
		t.Fatal("Expected debug to be false by default")
	}
}

func TestParseFlagsVersion(t *testing.T) {
	os.Args = []string{"prog", "--app-id", "123", "--private-key-path", "key.pem", "--owner", "org", "--repository", "repo", "--version"}
	cfg, err := parseFlags()

	if err != nil {
		t.Fatalf("parseFlags() error = %v", err)
	}

	if !cfg.version {
		t.Fatal("Expected version to be true")
	}
}

func TestParseFlagsDefaultVersion(t *testing.T) {
	os.Args = []string{"prog", "--app-id", "123", "--private-key-path", "key.pem", "--owner", "org", "--repository", "repo"}
	cfg, err := parseFlags()

	if err != nil {
		t.Fatalf("parseFlags() error = %v", err)
	}

	if cfg.version {
		t.Fatal("Expected version to be false by default")
	}
}

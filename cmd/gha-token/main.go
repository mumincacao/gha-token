package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/mumincacao/gha-token/internal/githubapp"
	"github.com/mumincacao/gha-token/internal/version"
)

const privateKeyEnvVar = "GITHUB_APP_PRIVATE_KEY"

type config struct {
	appID      string
	keyPath    string
	owner      string
	repository string
	debug      bool
	timeout    time.Duration
	version    bool
}

func main() {
	cfg, err := parseFlags()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if cfg.version {
		fmt.Printf("gha-token version %s\n", version.Version)
		os.Exit(0)
	}

	privateKeyPEM, err := resolvePrivateKey(cfg.keyPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	client := githubapp.NewClient(githubapp.ClientOptions{
		BaseURL: "https://api.github.com",
		Timeout: cfg.timeout,
		Debug:   cfg.debug,
		Stderr:  os.Stderr,
	})

	token, err := client.GetInstallationToken(cfg.appID, privateKeyPEM, cfg.owner, cfg.repository)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fmt.Println(token)
}

func parseFlags() (config, error) {
	fs := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	var cfg config
	var timeoutSec int

	fs.StringVar(&cfg.appID, "app-id", "", "GitHub App ID")
	fs.StringVar(&cfg.keyPath, "private-key-path", "", "Path to GitHub App private key PEM file")
	fs.StringVar(&cfg.owner, "owner", "", "GitHub owner (organization or user)")
	fs.StringVar(&cfg.repository, "repository", "", "GitHub repository name")
	fs.BoolVar(&cfg.debug, "debug", false, "Enable debug logs (without secrets)")
	fs.BoolVar(&cfg.version, "version", false, "Show version and exit")
	fs.IntVar(&timeoutSec, "timeout", 30, "HTTP timeout in seconds")

	if err := fs.Parse(os.Args[1:]); err != nil {
		return config{}, err
	}

	// If version flag is set, skip other required field validation
	if cfg.version {
		return cfg, nil
	}

	if cfg.appID == "" {
		return config{}, errors.New("--app-id is required")
	}
	if cfg.keyPath == "" && os.Getenv(privateKeyEnvVar) == "" {
		return config{}, errors.New("--private-key-path or GITHUB_APP_PRIVATE_KEY is required")
	}
	if cfg.owner == "" {
		return config{}, errors.New("--owner is required")
	}
	if cfg.repository == "" {
		return config{}, errors.New("--repository is required")
	}
	if timeoutSec <= 0 {
		return config{}, errors.New("--timeout must be greater than 0")
	}

	cfg.timeout = time.Duration(timeoutSec) * time.Second
	return cfg, nil
}

func resolvePrivateKey(privateKeyPath string) (string, error) {
	if privateKeyPath != "" {
		keyBytes, err := os.ReadFile(privateKeyPath)
		if err != nil {
			return "", fmt.Errorf("failed to read private key file: %w", err)
		}

		return string(keyBytes), nil
	}

	privateKeyPEM := os.Getenv(privateKeyEnvVar)
	if privateKeyPEM == "" {
		return "", errors.New("--private-key-path or GITHUB_APP_PRIVATE_KEY is required")
	}

	return privateKeyPEM, nil
}

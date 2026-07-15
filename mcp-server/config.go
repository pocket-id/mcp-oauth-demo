package main

import (
	"fmt"
	"net/url"
	"os"
	"strings"
)

const (
	defaultListenAddr  = ":8080"
	defaultResourceURL = "http://mcp:8080/mcp"
	defaultIssuerURL   = "http://localhost:1411"
	defaultJWKSURL     = "https://id.elias.li/.well-known/jwks.json"
)

var defaultScopes = []string{"notes:read", "notes:write"}

type config struct {
	ListenAddr     string
	ResourceURL    string
	IssuerURL      string
	JWKSURL        string
	RequiredScopes []string
	SigningAlgs    []string
}

func loadConfig() (config, error) {
	cfg := config{
		ListenAddr:     envOrDefault("LISTEN_ADDR", defaultListenAddr),
		ResourceURL:    envOrDefault("MCP_RESOURCE_URL", defaultResourceURL),
		IssuerURL:      strings.TrimRight(envOrDefault("OAUTH_ISSUER_URL", defaultIssuerURL), "/"),
		JWKSURL:        envOrDefault("OAUTH_JWKS_URL", defaultJWKSURL),
		RequiredScopes: splitList(envOrDefault("OAUTH_REQUIRED_SCOPES", strings.Join(defaultScopes, " "))),
		SigningAlgs:    splitList(envOrDefault("OAUTH_SIGNING_ALGS", "RS256")),
	}

	if cfg.ListenAddr == "" {
		return config{}, fmt.Errorf("LISTEN_ADDR must not be empty")
	}
	if err := validateHTTPURL("MCP_RESOURCE_URL", cfg.ResourceURL, true); err != nil {
		return config{}, err
	}
	if err := validateHTTPURL("OAUTH_ISSUER_URL", cfg.IssuerURL, false); err != nil {
		return config{}, err
	}
	if err := validateHTTPURL("OAUTH_JWKS_URL", cfg.JWKSURL, true); err != nil {
		return config{}, err
	}
	if len(cfg.RequiredScopes) == 0 {
		return config{}, fmt.Errorf("OAUTH_REQUIRED_SCOPES must contain at least one scope")
	}
	if len(cfg.SigningAlgs) == 0 {
		return config{}, fmt.Errorf("OAUTH_SIGNING_ALGS must contain at least one algorithm")
	}
	for _, alg := range cfg.SigningAlgs {
		if !isAllowedSigningAlgorithm(alg) {
			return config{}, fmt.Errorf("OAUTH_SIGNING_ALGS contains unsupported or unsafe algorithm %q", alg)
		}
	}

	return cfg, nil
}

func envOrDefault(name, fallback string) string {
	if value, ok := os.LookupEnv(name); ok {
		return strings.TrimSpace(value)
	}
	return fallback
}

func splitList(value string) []string {
	return strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || r == ' ' || r == '\t' || r == '\n'
	})
}

func validateHTTPURL(name, value string, allowPath bool) error {
	parsed, err := url.Parse(value)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return fmt.Errorf("%s must be an absolute HTTP(S) URL", name)
	}
	if parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return fmt.Errorf("%s must not contain user info, a query, or a fragment", name)
	}
	if !allowPath && parsed.Path != "" && parsed.Path != "/" {
		return fmt.Errorf("%s must not contain a path", name)
	}
	return nil
}

func isAllowedSigningAlgorithm(alg string) bool {
	switch alg {
	case "RS256", "RS384", "RS512", "PS256", "PS384", "PS512", "ES256", "ES384", "ES512", "EdDSA":
		return true
	default:
		return false
	}
}

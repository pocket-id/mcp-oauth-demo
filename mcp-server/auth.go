package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/modelcontextprotocol/go-sdk/auth"
)

type signedTokenVerifier interface {
	Verify(context.Context, string) (*oidc.IDToken, error)
}

func newAccessTokenVerifier(ctx context.Context, cfg config) auth.TokenVerifier {
	keySet := oidc.NewRemoteKeySet(ctx, cfg.JWKSURL)
	verifier := oidc.NewVerifier(cfg.IssuerURL, keySet, &oidc.Config{
		ClientID:             cfg.ResourceURL,
		SupportedSigningAlgs: cfg.SigningAlgs,
	})
	return accessTokenVerifier(verifier)
}

func accessTokenVerifier(verifier signedTokenVerifier) auth.TokenVerifier {
	return func(ctx context.Context, rawToken string, _ *http.Request) (*auth.TokenInfo, error) {
		token, err := verifier.Verify(ctx, rawToken)
		if err != nil {
			return nil, fmt.Errorf("%w: access token verification failed", auth.ErrInvalidToken)
		}
		if token.Subject == "" {
			return nil, fmt.Errorf("%w: access token has no subject", auth.ErrInvalidToken)
		}

		var claims accessTokenClaims
		if err := token.Claims(&claims); err != nil {
			return nil, fmt.Errorf("%w: access token claims are invalid", auth.ErrInvalidToken)
		}

		extra := map[string]any{}
		if claims.Email != "" {
			extra["email"] = claims.Email
		}
		if claims.PreferredUsername != "" {
			extra["preferred_username"] = claims.PreferredUsername
		}

		return &auth.TokenInfo{
			Scopes:     claims.Scope,
			Expiration: token.Expiry,
			UserID:     token.Subject,
			Extra:      extra,
		}, nil
	}
}

type accessTokenClaims struct {
	Scope             scopeClaim `json:"scope"`
	Email             string     `json:"email"`
	PreferredUsername string     `json:"preferred_username"`
}

type scopeClaim []string

func (s *scopeClaim) UnmarshalJSON(data []byte) error {
	var scopeString string
	if err := json.Unmarshal(data, &scopeString); err == nil {
		*s = strings.Fields(scopeString)
		return nil
	}

	var scopes []string
	if err := json.Unmarshal(data, &scopes); err != nil {
		return fmt.Errorf("scope must be a string or an array of strings")
	}
	*s = scopes
	return nil
}

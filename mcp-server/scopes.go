package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/auth"
)

const (
	notesReadScope  = "notes:read"
	notesWriteScope = "notes:write"
)

var toolScopes = map[string][]string{
	"list_notes":  {notesReadScope},
	"add_note":    {notesWriteScope},
	"clear_notes": {notesWriteScope},
}

// requireToolScopes authorizes each tools/call request against that tool's
// scopes. Other MCP methods only require a valid bearer token.
func requireToolScopes(resourceMetadataURL string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requiredScopes := requiredScopesForRequest(r)
			token := auth.TokenInfoFromContext(r.Context())
			for _, requiredScope := range requiredScopes {
				if token == nil || !slices.Contains(token.Scopes, requiredScope) {
					w.Header().Set("WWW-Authenticate", bearerScopeChallenge(resourceMetadataURL, requiredScopes))
					http.Error(w, "insufficient scope", http.StatusForbidden)
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

func requiredScopesForRequest(r *http.Request) []string {
	if r.Method != http.MethodPost || r.Body == nil {
		return nil
	}

	body, err := io.ReadAll(r.Body)
	r.Body = io.NopCloser(bytes.NewReader(body))
	if err != nil {
		return nil
	}

	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 {
		return nil
	}

	var requests []mcpRequestEnvelope
	if trimmed[0] == '[' {
		if err := json.Unmarshal(trimmed, &requests); err != nil {
			return nil
		}
	} else {
		var request mcpRequestEnvelope
		if err := json.Unmarshal(trimmed, &request); err != nil {
			return nil
		}
		requests = []mcpRequestEnvelope{request}
	}

	var requiredScopes []string
	for _, request := range requests {
		if request.Method != "tools/call" {
			continue
		}
		var params struct {
			Name string `json:"name"`
		}
		if err := json.Unmarshal(request.Params, &params); err != nil {
			continue
		}
		for _, scope := range toolScopes[params.Name] {
			if !slices.Contains(requiredScopes, scope) {
				requiredScopes = append(requiredScopes, scope)
			}
		}
	}
	return requiredScopes
}

type mcpRequestEnvelope struct {
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
}

func bearerScopeChallenge(resourceMetadataURL string, scopes []string) string {
	challenge := `Bearer error="insufficient_scope"`
	if len(scopes) > 0 {
		challenge += ", scope=" + strconv.Quote(strings.Join(scopes, " "))
	}
	if resourceMetadataURL != "" {
		challenge += ", resource_metadata=" + strconv.Quote(resourceMetadataURL)
	}
	return challenge
}

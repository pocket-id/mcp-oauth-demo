package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/modelcontextprotocol/go-sdk/oauthex"
)

const maxNoteLength = 500

type addNoteInput struct {
	Text string `json:"text" jsonschema:"the note to remember"`
}

type addNoteOutput struct {
	Note note `json:"note"`
}

type listNotesInput struct{}

type listNotesOutput struct {
	Notes []note `json:"notes"`
}

type clearNotesInput struct{}

type clearNotesOutput struct {
	Deleted int `json:"deleted" jsonschema:"number of notes deleted"`
}

func newHTTPHandler(cfg config, verifier auth.TokenVerifier, store *noteStore) http.Handler {
	metadata := &oauthex.ProtectedResourceMetadata{
		Resource:               cfg.ResourceURL,
		AuthorizationServers:   []string{cfg.IssuerURL},
		ScopesSupported:        cfg.RequiredScopes,
		BearerMethodsSupported: []string{"header"},
		ResourceName:           "Pocket ID OAuth Notes Demo",
	}
	metadataHandler := auth.ProtectedResourceMetadataHandler(metadata)

	mcpServer := newMCPServer(store)
	mcpHandler := mcp.NewStreamableHTTPHandler(
		func(*http.Request) *mcp.Server { return mcpServer },
		&mcp.StreamableHTTPOptions{Stateless: true, JSONResponse: true},
	)
	protectedMCPHandler := auth.RequireBearerToken(verifier, &auth.RequireBearerTokenOptions{
		ResourceMetadataURL: resourceMetadataURL(cfg.ResourceURL),
		Scopes:              cfg.RequiredScopes,
	})(mcpHandler)

	mux := http.NewServeMux()
	mux.Handle("/.well-known/oauth-protected-resource", metadataHandler)
	pathMetadata := resourceMetadataPath(cfg.ResourceURL)
	if pathMetadata != "/.well-known/oauth-protected-resource" {
		mux.Handle(pathMetadata, metadataHandler)
	}
	mux.Handle("/mcp", protectedMCPHandler)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	return mux
}

func newMCPServer(store *noteStore) *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "pocket-id-oauth-notes-demo",
		Version: "1.0.0",
	}, nil)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "add_note",
		Description: "Store a short note for the authenticated user. Notes are kept in memory and disappear when the server restarts.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input addNoteInput) (*mcp.CallToolResult, addNoteOutput, error) {
		userID, result := authenticatedUser(ctx)
		if result != nil {
			return result, addNoteOutput{}, nil
		}

		text := strings.TrimSpace(input.Text)
		if text == "" {
			return toolError("text must not be empty"), addNoteOutput{}, nil
		}
		if len(text) > maxNoteLength {
			return toolError(fmt.Sprintf("text must not exceed %d bytes", maxNoteLength)), addNoteOutput{}, nil
		}

		created, ok := store.add(userID, text)
		if !ok {
			return toolError(fmt.Sprintf("the in-memory limit of %d notes has been reached", maxNotesPerUser)), addNoteOutput{}, nil
		}
		return nil, addNoteOutput{Note: created}, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_notes",
		Description: "List only the notes owned by the authenticated user.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, _ listNotesInput) (*mcp.CallToolResult, listNotesOutput, error) {
		userID, result := authenticatedUser(ctx)
		if result != nil {
			return result, listNotesOutput{}, nil
		}
		return nil, listNotesOutput{Notes: store.list(userID)}, nil
	})

	mcp.AddTool(server, &mcp.Tool{
		Name:        "clear_notes",
		Description: "Delete all in-memory notes owned by the authenticated user.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, _ clearNotesInput) (*mcp.CallToolResult, clearNotesOutput, error) {
		userID, result := authenticatedUser(ctx)
		if result != nil {
			return result, clearNotesOutput{}, nil
		}
		return nil, clearNotesOutput{Deleted: store.clear(userID)}, nil
	})

	return server
}

func authenticatedUser(ctx context.Context) (string, *mcp.CallToolResult) {
	token := auth.TokenInfoFromContext(ctx)
	if token == nil || token.UserID == "" {
		return "", toolError("the OAuth subject is unavailable")
	}
	return token.UserID, nil
}

func toolError(message string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{&mcp.TextContent{Text: message}},
	}
}

func resourceMetadataPath(resource string) string {
	parsed, err := url.Parse(resource)
	if err != nil || parsed.Path == "" || parsed.Path == "/" {
		return "/.well-known/oauth-protected-resource"
	}
	return "/.well-known/oauth-protected-resource" + parsed.EscapedPath()
}

func resourceMetadataURL(resource string) string {
	parsed, err := url.Parse(resource)
	if err != nil {
		return ""
	}
	return parsed.Scheme + "://" + parsed.Host + resourceMetadataPath(resource)
}

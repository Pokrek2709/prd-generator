package ai

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/your-org/prd-generator/backend/internal/config"
	"github.com/your-org/prd-generator/backend/internal/domain"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestGenerateUsesStructuredProviderResponse(t *testing.T) {
	provider := New(config.Config{AIAPIKey: "test-key", AIBaseURL: "https://provider.invalid/v1", AIModel: "test-model", AITimeout: time.Second})
	provider.client = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("authorization header missing")
		}
		body := `{"choices":[{"message":{"content":"{\"executiveSummary\":\"Summary\",\"productOverview\":\"Overview\",\"problemStatement\":\"Problem\",\"functionalRequirements\":[]}"}}],"usage":{"prompt_tokens":12,"completion_tokens":34}}`
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: r}, nil
	})}
	document, inputTokens, outputTokens, err := provider.Generate(context.Background(), domain.Idea{ProductName: "Test", Description: "A product idea"})
	if err != nil {
		t.Fatal(err)
	}
	if document.ExecutiveSummary != "Summary" || inputTokens != 12 || outputTokens != 34 {
		t.Fatalf("unexpected result: %#v, %d, %d", document, inputTokens, outputTokens)
	}
}

func TestActionRejectsUnknownActionWithoutProviderCall(t *testing.T) {
	provider := New(config.Config{})
	if _, _, _, err := provider.Act(context.Background(), "drop_database", nil); err == nil {
		t.Fatal("unknown action was accepted")
	}
}

func TestGenerateStreamWithEventStream(t *testing.T) {
	provider := New(config.Config{AIAPIKey: "test-key", AIBaseURL: "https://provider.invalid/v1", AIModel: "test-model", AITimeout: time.Second})
	provider.client = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		respHeader := make(http.Header)
		respHeader.Set("Content-Type", "text/event-stream")
		sseBody := "data: {\"choices\":[{\"delta\":{\"content\":\"{\\\"executiveSummary\\\":\\\"Streamed Summary\\\",\\\"productOverview\\\":\\\"Streamed Overview\\\",\\\"problemStatement\\\":\\\"Streamed Problem\\\",\\\"functionalRequirements\\\":[]}\"}}]}\n\ndata: [DONE]\n\n"
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     respHeader,
			Body:       io.NopCloser(strings.NewReader(sseBody)),
			Request:    r,
		}, nil
	})}
	var chunks []string
	onChunk := func(chunk string) {
		chunks = append(chunks, chunk)
	}
	doc, _, _, err := provider.GenerateStream(context.Background(), domain.Idea{ProductName: "Test", Description: "A product idea"}, onChunk)
	if err != nil {
		t.Fatal(err)
	}
	if doc.ExecutiveSummary != "Streamed Summary" {
		t.Fatalf("expected 'Streamed Summary', got %q", doc.ExecutiveSummary)
	}
	if len(chunks) == 0 {
		t.Fatalf("expected onChunk to be called with chunks")
	}
}

func TestGetProvidersCatalog(t *testing.T) {
	catalog := GetProvidersCatalog()
	if len(catalog) < 6 {
		t.Fatalf("expected at least 6 providers, got %d", len(catalog))
	}
	foundGemini := false
	foundOllama := false
	for _, p := range catalog {
		if p.ID == "gemini" {
			foundGemini = true
			if p.DefaultModel != "gemini-2.0-flash" {
				t.Errorf("expected default model gemini-2.0-flash, got %s", p.DefaultModel)
			}
		}
		if p.ID == "ollama" {
			foundOllama = true
			if p.RequiresAPIKey {
				t.Errorf("ollama should not require api key")
			}
		}
	}
	if !foundGemini || !foundOllama {
		t.Errorf("gemini or ollama missing from catalog")
	}
}

func TestResolveSettings(t *testing.T) {
	cfg := config.Config{
		AIModel:   "gpt-4o-mini",
		AIBaseURL: "https://api.openai.com/v1",
		AIAPIKey:  "server-key",
	}

	// Case 1: default config
	provider, baseURL, apiKey, model := ResolveSettings(nil, "", "", cfg)
	if provider != "openai" || model != "gpt-4o-mini" || apiKey != "server-key" {
		t.Errorf("unexpected default resolve: %s, %s, %s, %s", provider, baseURL, apiKey, model)
	}

	// Case 2: user settings override provider and model
	userSettings := &domain.UserAISettings{
		Provider: "gemini",
		Model:    "gemini-1.5-pro",
		APIKey:   "user-gemini-key",
	}
	provider, baseURL, apiKey, model = ResolveSettings(userSettings, "", "", cfg)
	if provider != "gemini" || model != "gemini-1.5-pro" || apiKey != "user-gemini-key" || baseURL != "https://generativelanguage.googleapis.com/v1beta/openai" {
		t.Errorf("unexpected user resolve: %s, %s, %s, %s", provider, baseURL, apiKey, model)
	}

	// Case 3: request override model
	provider, baseURL, apiKey, model = ResolveSettings(userSettings, "", "gemini-2.0-flash", cfg)
	if model != "gemini-2.0-flash" {
		t.Errorf("expected model override to take precedence: got %s", model)
	}
}



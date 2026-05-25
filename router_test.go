package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

type rewriteTransport struct {
	base   http.RoundTripper
	target string
}

func (r rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	cloned := req.Clone(req.Context())
	url := r.target + req.URL.Path
	if req.URL.RawQuery != "" {
		url += "?" + req.URL.RawQuery
	}
	parsedReq, err := http.NewRequest(req.Method, url, req.Body)
	if err != nil {
		return nil, err
	}
	parsedReq.Header = cloned.Header.Clone()
	return r.base.RoundTrip(parsedReq)
}

func withPatchedDefaultClient(t *testing.T, serverURL string) {
	t.Helper()
	original := http.DefaultClient.Transport
	base := original
	if base == nil {
		base = http.DefaultTransport
	}
	http.DefaultClient.Transport = rewriteTransport{
		base:   base,
		target: serverURL,
	}
	t.Cleanup(func() {
		http.DefaultClient.Transport = original
	})
}

func TestResolveTarget_DirectTarget(t *testing.T) {
	config := &Config{
		Targets: []Target{
			{Name: "ChatGPT", Type: "mac_app", App: "ChatGPT"},
		},
	}

	target, chosenName, err := ResolveTarget(config, "ChatGPT", "hello")
	if err != nil {
		t.Fatalf("ResolveTarget returned error: %v", err)
	}
	if target == nil {
		t.Fatalf("ResolveTarget returned nil target")
	}
	if chosenName != "ChatGPT" {
		t.Fatalf("chosenName = %q, want %q", chosenName, "ChatGPT")
	}
}

func TestResolveTarget_UnknownSelection(t *testing.T) {
	config := &Config{
		Targets: []Target{
			{Name: "ChatGPT", Type: "mac_app", App: "ChatGPT"},
		},
	}

	_, _, err := ResolveTarget(config, "Claude", "hello")
	if err == nil {
		t.Fatalf("expected error for unknown target")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Fatalf("error = %q, want not found", err.Error())
	}
}

func TestResolveTarget_AutoWithoutRouterConfig(t *testing.T) {
	config := &Config{
		Targets: []Target{
			{Name: "Auto", Type: "auto"},
			{Name: "ChatGPT", Type: "mac_app", App: "ChatGPT"},
		},
	}

	_, _, err := ResolveTarget(config, "Auto", "hello")
	if err == nil {
		t.Fatalf("expected error for missing router config")
	}
	if !strings.Contains(err.Error(), "router config is required") {
		t.Fatalf("error = %q, want router config is required", err.Error())
	}
}

func TestRoutePrompt_HTTPScenarios(t *testing.T) {
	tests := []struct {
		name        string
		statusCode  int
		response    string
		wantTarget  string
		errContains string
	}{
		{
			name:        "non-2xx",
			statusCode:  http.StatusBadRequest,
			response:    `{"error":{"message":"bad request"}}`,
			errContains: "router request failed with status 400",
		},
		{
			name:       "completed valid target",
			statusCode: http.StatusOK,
			response: `{
			  "status":"completed",
			  "output":[{"type":"message","content":[{"type":"output_text","text":"{\"target\":\"Claude\"}"}]}]
			}`,
			wantTarget: "Claude",
		},
		{
			name:       "refusal",
			statusCode: http.StatusOK,
			response: `{
			  "status":"completed",
			  "output":[{"type":"message","content":[{"type":"refusal","refusal":"cannot do that"}]}]
			}`,
			errContains: "router refused request: cannot do that",
		},
		{
			name:       "empty output",
			statusCode: http.StatusOK,
			response: `{
			  "status":"completed",
			  "output":[{"type":"message","content":[{"type":"output_text","text":"   "}]}]
			}`,
			errContains: "router returned empty output",
		},
		{
			name:       "malformed json output",
			statusCode: http.StatusOK,
			response: `{
			  "status":"completed",
			  "output":[{"type":"message","content":[{"type":"output_text","text":"not json"}]}]
			}`,
			errContains: "decode router choice",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/v1/responses" {
					http.Error(w, "bad path", http.StatusNotFound)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.response))
			}))
			defer server.Close()

			withPatchedDefaultClient(t, server.URL)

			t.Setenv("TEST_OPENAI_KEY", "abc123")
			config := &Config{
				Router: &Router{
					Provider:  "openai",
					Model:     "gpt-4o-mini",
					APIKeyEnv: "TEST_OPENAI_KEY",
				},
				Targets: []Target{
					{Name: "Auto", Type: "auto"},
					{Name: "ChatGPT", Type: "mac_app", App: "ChatGPT"},
					{Name: "Claude", Type: "mac_app", App: "Claude"},
				},
			}

			got, err := RoutePrompt(config, "route this")
			if tt.errContains != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.errContains)
				}
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Fatalf("error = %q, want substring %q", err.Error(), tt.errContains)
				}
				return
			}

			if err != nil {
				t.Fatalf("RoutePrompt returned error: %v", err)
			}
			if got != tt.wantTarget {
				t.Fatalf("RoutePrompt target = %q, want %q", got, tt.wantTarget)
			}
		})
	}
}

func TestRoutePrompt_APIKeyMissing(t *testing.T) {
	config := &Config{
		Router: &Router{
			Provider:  "openai",
			Model:     "gpt-4o-mini",
			APIKeyEnv: "MISSING_KEY",
		},
		Targets: []Target{
			{Name: "ChatGPT", Type: "mac_app", App: "ChatGPT"},
		},
	}

	_ = os.Unsetenv("MISSING_KEY")
	_, err := RoutePrompt(config, "hello")
	if err == nil {
		t.Fatalf("expected missing key error")
	}
	if !strings.Contains(err.Error(), `router API key env var "MISSING_KEY" is not set`) {
		t.Fatalf("error = %q", err.Error())
	}
}

func TestRoutePrompt_RequiresNonAutoTarget(t *testing.T) {
	t.Setenv("TEST_OPENAI_KEY", "abc123")
	config := &Config{
		Router: &Router{
			Provider:  "openai",
			Model:     "gpt-4o-mini",
			APIKeyEnv: "TEST_OPENAI_KEY",
		},
		Targets: []Target{
			{Name: "Auto", Type: "auto"},
		},
	}

	_, err := RoutePrompt(config, "hello")
	if err == nil {
		t.Fatalf("expected non-auto target error")
	}
	if !strings.Contains(err.Error(), "at least one non-auto target") {
		t.Fatalf("error = %q", err.Error())
	}
}

func TestExtractRouterOutputText(t *testing.T) {
	items := []openAIOutputItem{
		{
			Type: "message",
			Content: []openAIOutputContent{
				{Type: "output_text", Text: "  "},
				{Type: "output_text", Text: "{\"target\":\"ChatGPT\"}"},
			},
		},
	}

	got := extractRouterOutputText(items)
	if got != "{\"target\":\"ChatGPT\"}" {
		t.Fatalf("extractRouterOutputText = %q", got)
	}
}

func TestExtractRouterRefusalText(t *testing.T) {
	items := []openAIOutputItem{
		{
			Type: "message",
			Content: []openAIOutputContent{
				{Type: "refusal", Refusal: "cannot route"},
			},
		},
	}

	got := extractRouterRefusalText(items)
	if got != "cannot route" {
		t.Fatalf("extractRouterRefusalText = %q", got)
	}
}

func TestBuildRouterTextHelpers(t *testing.T) {
	targets := []Target{
		{Name: "ChatGPT", Description: "generalist"},
		{Name: "Claude", Description: "careful reasoning"},
	}

	instructions := buildDefaultRouterInstructions(targets)
	if !strings.Contains(instructions, "Choose the single best target") {
		t.Fatalf("instructions missing expected guidance: %q", instructions)
	}
	if !strings.Contains(instructions, "- ChatGPT: generalist") {
		t.Fatalf("instructions missing target listing: %q", instructions)
	}

	input := buildRouterInput("hello", targets)
	for _, want := range []string{"User prompt:", "hello", "Targets:", "- Claude: careful reasoning"} {
		if !strings.Contains(input, want) {
			t.Fatalf("router input missing %q: %q", want, input)
		}
	}
}

func TestResolveTarget_AutoChoosesKnownTarget(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"status":"completed","output":[{"type":"message","content":[{"type":"output_text","text":"{\"target\":\"Claude\"}"}]}]}`)
	}))
	defer server.Close()
	withPatchedDefaultClient(t, server.URL)

	t.Setenv("TEST_OPENAI_KEY", "abc123")
	config := &Config{
		Router: &Router{
			Provider:  "openai",
			Model:     "gpt-4o-mini",
			APIKeyEnv: "TEST_OPENAI_KEY",
		},
		Targets: []Target{
			{Name: "Auto", Type: "auto"},
			{Name: "ChatGPT", Type: "mac_app", App: "ChatGPT"},
			{Name: "Claude", Type: "mac_app", App: "Claude"},
		},
	}

	target, name, err := ResolveTarget(config, "Auto", "hello")
	if err != nil {
		t.Fatalf("ResolveTarget returned error: %v", err)
	}
	if target == nil || target.Name != "Claude" || name != "Claude" {
		t.Fatalf("ResolveTarget auto result unexpected: target=%v name=%q", target, name)
	}
}

func TestRoutePrompt_Gemini(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, "generateContent") {
			http.Error(w, "bad path", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, `{
			"candidates": [
				{
					"content": {
						"parts": [
							{
								"text": "{\"target\": \"ChatGPT\"}"
							}
						]
					},
					"finishReason": "STOP"
				}
			]
		}`)
	}))
	defer server.Close()
	withPatchedDefaultClient(t, server.URL)

	t.Setenv("TEST_GEMINI_KEY", "gemini123")
	config := &Config{
		Router: &Router{
			Provider:  "gemini",
			Model:     "gemini-1.5-flash",
			APIKeyEnv: "TEST_GEMINI_KEY",
		},
		Targets: []Target{
			{Name: "Auto", Type: "auto"},
			{Name: "ChatGPT", Type: "mac_app", App: "ChatGPT"},
			{Name: "Claude", Type: "mac_app", App: "Claude"},
		},
	}

	got, err := RoutePrompt(config, "route me please")
	if err != nil {
		t.Fatalf("RoutePrompt (gemini) returned error: %v", err)
	}
	if got != "ChatGPT" {
		t.Fatalf("RoutePrompt (gemini) = %q, want %q", got, "ChatGPT")
	}
}

func TestRoutePrompt_Anthropic(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/messages" {
			http.Error(w, "bad path", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, `{
			"content": [
				{
					"type": "tool_use",
					"id": "toolu_123",
					"name": "route_choice",
					"input": {
						"target": "Claude"
					}
				}
			]
		}`)
	}))
	defer server.Close()
	withPatchedDefaultClient(t, server.URL)

	t.Setenv("TEST_ANTHROPIC_KEY", "claude123")
	config := &Config{
		Router: &Router{
			Provider:  "anthropic",
			Model:     "claude-3-5-sonnet-latest",
			APIKeyEnv: "TEST_ANTHROPIC_KEY",
		},
		Targets: []Target{
			{Name: "Auto", Type: "auto"},
			{Name: "ChatGPT", Type: "mac_app", App: "ChatGPT"},
			{Name: "Claude", Type: "mac_app", App: "Claude"},
		},
	}

	got, err := RoutePrompt(config, "route me please")
	if err != nil {
		t.Fatalf("RoutePrompt (anthropic) returned error: %v", err)
	}
	if got != "Claude" {
		t.Fatalf("RoutePrompt (anthropic) = %q, want %q", got, "Claude")
	}
}

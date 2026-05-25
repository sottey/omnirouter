package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCallLLMAPI_AllProviders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		if strings.Contains(r.URL.Path, "/v1/chat/completions") {
			// OpenAI mock response
			_, _ = fmt.Fprint(w, `{
				"choices": [
					{
						"message": {
							"role": "assistant",
							"content": "OpenAI API response text"
						}
					}
				]
			}`)
			return
		}

		if strings.Contains(r.URL.Path, "generateContent") {
			// Gemini mock response
			_, _ = fmt.Fprint(w, `{
				"candidates": [
					{
						"content": {
							"parts": [
								{
									"text": "Gemini API response text"
								}
							]
						}
					}
				]
			}`)
			return
		}

		if strings.Contains(r.URL.Path, "/v1/messages") {
			// Anthropic mock response
			_, _ = fmt.Fprint(w, `{
				"content": [
					{
						"type": "text",
						"text": "Anthropic API response text"
					}
				]
			}`)
			return
		}

		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer server.Close()

	// Patch default client to redirect to mock server
	withPatchedDefaultClient(t, server.URL)

	t.Setenv("TEST_OPENAI_KEY", "op123")
	t.Setenv("TEST_GEMINI_KEY", "gem123")
	t.Setenv("TEST_ANTHROPIC_KEY", "ant123")

	// Test 1: OpenAI
	gotOpenAI, err := CallLLMAPI("openai", "gpt-4o-mini", "TEST_OPENAI_KEY", "system instructions", "hello")
	if err != nil {
		t.Fatalf("CallLLMAPI (openai) failed: %v", err)
	}
	if gotOpenAI != "OpenAI API response text" {
		t.Fatalf("got = %q, want %q", gotOpenAI, "OpenAI API response text")
	}

	// Test 2: Gemini
	gotGemini, err := CallLLMAPI("gemini", "gemini-1.5-flash", "TEST_GEMINI_KEY", "system instructions", "hello")
	if err != nil {
		t.Fatalf("CallLLMAPI (gemini) failed: %v", err)
	}
	if gotGemini != "Gemini API response text" {
		t.Fatalf("got = %q, want %q", gotGemini, "Gemini API response text")
	}

	// Test 3: Anthropic
	gotAnthropic, err := CallLLMAPI("anthropic", "claude-3-5-sonnet-latest", "TEST_ANTHROPIC_KEY", "system instructions", "hello")
	if err != nil {
		t.Fatalf("CallLLMAPI (anthropic) failed: %v", err)
	}
	if gotAnthropic != "Anthropic API response text" {
		t.Fatalf("got = %q, want %q", gotAnthropic, "Anthropic API response text")
	}
}

func TestCallLLMAPI_MissingKey(t *testing.T) {
	_, err := CallLLMAPI("openai", "gpt-4o-mini", "NON_EXISTENT_KEY", "system", "prompt")
	if err == nil {
		t.Fatalf("expected error for missing key env var")
	}
	if !strings.Contains(err.Error(), "env var \"NON_EXISTENT_KEY\" is not set") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

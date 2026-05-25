package core

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	omnirouterconfig "omnirouter/internal/config"
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

func TestCallLLMAPI_AllProviders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		switch {
		case strings.Contains(r.URL.Path, "/v1/chat/completions"):
			_, _ = fmt.Fprint(w, `{"choices":[{"message":{"role":"assistant","content":"OpenAI API response text"}}]}`)
		case strings.Contains(r.URL.Path, "generateContent"):
			_, _ = fmt.Fprint(w, `{"candidates":[{"content":{"parts":[{"text":"Gemini API response text"}]}}]}`)
		case strings.Contains(r.URL.Path, "/v1/messages"):
			_, _ = fmt.Fprint(w, `{"content":[{"type":"text","text":"Anthropic API response text"}]}`)
		default:
			http.Error(w, "not found", http.StatusNotFound)
		}
	}))
	defer server.Close()

	withPatchedDefaultClient(t, server.URL)
	t.Setenv("TEST_OPENAI_KEY", "op123")
	t.Setenv("TEST_GEMINI_KEY", "gem123")
	t.Setenv("TEST_ANTHROPIC_KEY", "ant123")

	gotOpenAI, err := CallLLMAPI("openai", "gpt-4o-mini", "TEST_OPENAI_KEY", "system instructions", "hello")
	if err != nil || gotOpenAI != "OpenAI API response text" {
		t.Fatalf("openai result=%q err=%v", gotOpenAI, err)
	}
	gotGemini, err := CallLLMAPI("gemini", "gemini-1.5-flash", "TEST_GEMINI_KEY", "system instructions", "hello")
	if err != nil || gotGemini != "Gemini API response text" {
		t.Fatalf("gemini result=%q err=%v", gotGemini, err)
	}
	gotAnthropic, err := CallLLMAPI("anthropic", "claude-3-5-sonnet-latest", "TEST_ANTHROPIC_KEY", "system instructions", "hello")
	if err != nil || gotAnthropic != "Anthropic API response text" {
		t.Fatalf("anthropic result=%q err=%v", gotAnthropic, err)
	}
}

func TestCallLLMAPI_MissingKeyAndUnsupportedProvider(t *testing.T) {
	if _, err := CallLLMAPI("openai", "gpt-4o-mini", "NON_EXISTENT_KEY", "system", "prompt"); err == nil || !strings.Contains(err.Error(), `env var "NON_EXISTENT_KEY" is not set`) {
		t.Fatalf("error = %v", err)
	}
	t.Setenv("KEY", "abc123")
	if _, err := CallLLMAPI("bad", "model", "KEY", "", "prompt"); err == nil || !strings.Contains(err.Error(), "unsupported API provider") {
		t.Fatalf("error = %v", err)
	}
}

func TestResolveTargetAndRoutePrompt(t *testing.T) {
	config := &omnirouterconfig.Config{
		Targets: []omnirouterconfig.Target{
			{Name: "ChatGPT", Type: "mac_app", App: "ChatGPT"},
		},
	}
	target, chosenName, err := ResolveTarget(config, "ChatGPT", "hello")
	if err != nil || target == nil || chosenName != "ChatGPT" {
		t.Fatalf("ResolveTarget result=%v chosen=%q err=%v", target, chosenName, err)
	}

	if _, _, err := ResolveTarget(config, "Claude", "hello"); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("error = %v", err)
	}

	configAuto := &omnirouterconfig.Config{
		Targets: []omnirouterconfig.Target{
			{Name: "Auto", Type: "auto"},
			{Name: "ChatGPT", Type: "mac_app", App: "ChatGPT"},
		},
	}
	if _, _, err := ResolveTarget(configAuto, "Auto", "hello"); err == nil || !strings.Contains(err.Error(), "router config is required") {
		t.Fatalf("error = %v", err)
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
		{name: "non-2xx", statusCode: http.StatusBadRequest, response: `{"error":{"message":"bad request"}}`, errContains: "router request failed with status 400"},
		{name: "completed valid target", statusCode: http.StatusOK, response: `{"status":"completed","output":[{"type":"message","content":[{"type":"output_text","text":"{\"target\":\"Claude\"}"}]}]}`, wantTarget: "Claude"},
		{name: "refusal", statusCode: http.StatusOK, response: `{"status":"completed","output":[{"type":"message","content":[{"type":"refusal","refusal":"cannot do that"}]}]}`, errContains: "router refused request: cannot do that"},
		{name: "empty output", statusCode: http.StatusOK, response: `{"status":"completed","output":[{"type":"message","content":[{"type":"output_text","text":"   "}]}]}`, errContains: "router returned empty output"},
		{name: "malformed json output", statusCode: http.StatusOK, response: `{"status":"completed","output":[{"type":"message","content":[{"type":"output_text","text":"not json"}]}]}`, errContains: "decode router choice"},
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
			config := &omnirouterconfig.Config{
				Router: &omnirouterconfig.Router{Provider: "openai", Model: "gpt-4o-mini", APIKeyEnv: "TEST_OPENAI_KEY"},
				Targets: []omnirouterconfig.Target{
					{Name: "Auto", Type: "auto"},
					{Name: "ChatGPT", Type: "mac_app", App: "ChatGPT"},
					{Name: "Claude", Type: "mac_app", App: "Claude"},
				},
			}

			got, err := RoutePrompt(config, "route this")
			if tt.errContains != "" {
				if err == nil || !strings.Contains(err.Error(), tt.errContains) {
					t.Fatalf("error = %v", err)
				}
				return
			}
			if err != nil || got != tt.wantTarget {
				t.Fatalf("got=%q err=%v", got, err)
			}
		})
	}
}

func TestRoutePrompt_APIKeyMissingAndRequiresNonAutoTarget(t *testing.T) {
	config := &omnirouterconfig.Config{
		Router: &omnirouterconfig.Router{Provider: "openai", Model: "gpt-4o-mini", APIKeyEnv: "MISSING_KEY"},
		Targets: []omnirouterconfig.Target{
			{Name: "ChatGPT", Type: "mac_app", App: "ChatGPT"},
		},
	}
	_ = os.Unsetenv("MISSING_KEY")
	if _, err := RoutePrompt(config, "hello"); err == nil || !strings.Contains(err.Error(), `router API key env var "MISSING_KEY" is not set`) {
		t.Fatalf("error = %v", err)
	}

	t.Setenv("TEST_OPENAI_KEY", "abc123")
	config = &omnirouterconfig.Config{
		Router: &omnirouterconfig.Router{Provider: "openai", Model: "gpt-4o-mini", APIKeyEnv: "TEST_OPENAI_KEY"},
		Targets: []omnirouterconfig.Target{
			{Name: "Auto", Type: "auto"},
		},
	}
	if _, err := RoutePrompt(config, "hello"); err == nil || !strings.Contains(err.Error(), "at least one non-auto target") {
		t.Fatalf("error = %v", err)
	}
}

func TestRouterHelperTextBuilders(t *testing.T) {
	items := []openAIOutputItem{
		{Type: "message", Content: []openAIOutputContent{{Type: "output_text", Text: "  "}, {Type: "output_text", Text: "{\"target\":\"ChatGPT\"}"}}},
	}
	if got := extractRouterOutputText(items); got != "{\"target\":\"ChatGPT\"}" {
		t.Fatalf("extractRouterOutputText = %q", got)
	}

	items = []openAIOutputItem{
		{Type: "message", Content: []openAIOutputContent{{Type: "refusal", Refusal: "cannot route"}}},
	}
	if got := extractRouterRefusalText(items); got != "cannot route" {
		t.Fatalf("extractRouterRefusalText = %q", got)
	}

	targets := []omnirouterconfig.Target{
		{Name: "ChatGPT", Description: "generalist"},
		{Name: "Claude", Description: "careful reasoning"},
	}
	instructions := buildDefaultRouterInstructions(targets)
	if !strings.Contains(instructions, "Choose the single best target") || !strings.Contains(instructions, "- ChatGPT: generalist") {
		t.Fatalf("instructions = %q", instructions)
	}
	input := buildRouterInput("hello", targets)
	for _, want := range []string{"User prompt:", "hello", "Targets:", "- Claude: careful reasoning"} {
		if !strings.Contains(input, want) {
			t.Fatalf("router input missing %q: %q", want, input)
		}
	}
}

func TestRoutePrompt_GeminiAndAnthropic(t *testing.T) {
	t.Run("gemini", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !strings.Contains(r.URL.Path, "generateContent") {
				http.Error(w, "bad path", http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprint(w, `{"candidates":[{"content":{"parts":[{"text":"{\"target\":\"ChatGPT\"}"}]},"finishReason":"STOP"}]}`)
		}))
		defer server.Close()
		withPatchedDefaultClient(t, server.URL)
		t.Setenv("TEST_GEMINI_KEY", "gemini123")
		config := &omnirouterconfig.Config{
			Router: &omnirouterconfig.Router{Provider: "gemini", Model: "gemini-1.5-flash", APIKeyEnv: "TEST_GEMINI_KEY"},
			Targets: []omnirouterconfig.Target{
				{Name: "Auto", Type: "auto"},
				{Name: "ChatGPT", Type: "mac_app", App: "ChatGPT"},
			},
		}
		got, err := RoutePrompt(config, "route me please")
		if err != nil || got != "ChatGPT" {
			t.Fatalf("got=%q err=%v", got, err)
		}
	})

	t.Run("anthropic", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/v1/messages" {
				http.Error(w, "bad path", http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprint(w, `{"content":[{"type":"tool_use","id":"toolu_123","name":"route_choice","input":{"target":"Claude"}}]}`)
		}))
		defer server.Close()
		withPatchedDefaultClient(t, server.URL)
		t.Setenv("TEST_ANTHROPIC_KEY", "claude123")
		config := &omnirouterconfig.Config{
			Router: &omnirouterconfig.Router{Provider: "anthropic", Model: "claude-3-5-sonnet-latest", APIKeyEnv: "TEST_ANTHROPIC_KEY"},
			Targets: []omnirouterconfig.Target{
				{Name: "Auto", Type: "auto"},
				{Name: "Claude", Type: "mac_app", App: "Claude"},
			},
		}
		got, err := RoutePrompt(config, "route me please")
		if err != nil || got != "Claude" {
			t.Fatalf("got=%q err=%v", got, err)
		}
	})
}

func TestService_GetSaveReloadAndSendAPI(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	cfg := &omnirouterconfig.Config{
		Targets: []omnirouterconfig.Target{
			{Name: "OpenAI API", Type: "api", Provider: "openai", Model: "gpt-4o-mini", APIKeyEnv: "TEST_OPENAI_KEY"},
		},
	}
	if err := cfg.ApplyDefaultsAndValidate(); err != nil {
		t.Fatalf("ApplyDefaultsAndValidate returned error: %v", err)
	}
	if err := omnirouterconfig.WriteConfig(configPath, cfg); err != nil {
		t.Fatalf("WriteConfig returned error: %v", err)
	}

	service := NewService(configPath, cfg)
	gotConfig, err := service.GetConfig()
	if err != nil || len(gotConfig.Targets) != 1 {
		t.Fatalf("GetConfig config=%#v err=%v", gotConfig, err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"choices":[{"message":{"role":"assistant","content":"API service response"}}]}`)
	}))
	defer server.Close()
	withPatchedDefaultClient(t, server.URL)
	t.Setenv("TEST_OPENAI_KEY", "abc123")

	result, err := service.SendPrompt("OpenAI API", "hello")
	if err != nil || !result.IsAPI || result.ResponseText != "API service response" {
		t.Fatalf("result=%#v err=%v", result, err)
	}

	nextCfg := omnirouterconfig.Config{
		DefaultTarget: "OpenAI API",
		Targets: []omnirouterconfig.Target{
			{Name: "OpenAI API", Type: "api", Provider: "openai", Model: "gpt-4o-mini", APIKeyEnv: "TEST_OPENAI_KEY"},
		},
	}
	if err := service.SaveConfig(nextCfg); err != nil {
		t.Fatalf("SaveConfig returned error: %v", err)
	}

	editedCfg := &omnirouterconfig.Config{
		DefaultTarget: "OpenAI API",
		Targets: []omnirouterconfig.Target{
			{Name: "OpenAI API", Type: "api", Provider: "openai", Model: "gpt-4o-mini", APIKeyEnv: "TEST_OPENAI_KEY", SystemPrompt: "updated"},
		},
	}
	if err := editedCfg.ApplyDefaultsAndValidate(); err != nil {
		t.Fatalf("ApplyDefaultsAndValidate returned error: %v", err)
	}
	if err := omnirouterconfig.WriteConfig(configPath, editedCfg); err != nil {
		t.Fatalf("WriteConfig returned error: %v", err)
	}
	if err := service.ReloadConfig(); err != nil {
		t.Fatalf("ReloadConfig returned error: %v", err)
	}
	gotConfig, err = service.GetConfig()
	if err != nil || gotConfig.Targets[0].SystemPrompt != "updated" {
		t.Fatalf("GetConfig config=%#v err=%v", gotConfig, err)
	}
}

func TestServiceAndAutomationErrors(t *testing.T) {
	service := NewService("config.json", nil)
	if _, err := service.GetConfig(); err == nil || !strings.Contains(err.Error(), "config not loaded") {
		t.Fatalf("error = %v", err)
	}
	if _, err := service.SendPrompt("ChatGPT", "hello"); err == nil || !strings.Contains(err.Error(), "config not loaded") {
		t.Fatalf("error = %v", err)
	}
	if err := service.TestTarget("ChatGPT"); err == nil || !strings.Contains(err.Error(), "config not loaded") {
		t.Fatalf("error = %v", err)
	}

	service = NewService("config.json", &omnirouterconfig.Config{
		Targets: []omnirouterconfig.Target{
			{Name: "Auto", Type: "auto"},
			{Name: "ChatGPT", Type: "mac_app", App: "ChatGPT"},
		},
	})
	if err := service.TestTarget("Auto"); err == nil || !strings.Contains(err.Error(), "must not be auto") {
		t.Fatalf("error = %v", err)
	}
	if err := service.TestTarget("Missing"); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("error = %v", err)
	}

	if err := SendPromptToTarget(omnirouterconfig.Target{Type: "mac_app", App: "ChatGPT"}, "   "); err == nil || !strings.Contains(err.Error(), "prompt is required") {
		t.Fatalf("error = %v", err)
	}
	if err := SendPromptToTarget(omnirouterconfig.Target{Type: "bad"}, "hello"); err == nil || !strings.Contains(err.Error(), "unsupported target type") {
		t.Fatalf("error = %v", err)
	}
	if err := sendToMacApp(omnirouterconfig.Target{SendMode: "bad"}, "hello"); err == nil || !strings.Contains(err.Error(), "unsupported sendMode") {
		t.Fatalf("error = %v", err)
	}
}

func TestSendToMacApp_CommandFlows(t *testing.T) {
	originalExecCommand := execCommand
	originalSleep := sleep
	t.Cleanup(func() {
		execCommand = originalExecCommand
		sleep = originalSleep
	})
	sleep = func(time.Duration) {}

	makeRunner := func(mode string) func(string, ...string) *exec.Cmd {
		return func(name string, args ...string) *exec.Cmd {
			switch name {
			case "pbpaste":
				return exec.Command("sh", "-c", "printf old-clipboard")
			case "open":
				if mode == "open-fail" {
					return exec.Command("sh", "-c", "exit 1")
				}
				return exec.Command("sh", "-c", "exit 0")
			case "pbcopy":
				if mode == "copy-fail" {
					return exec.Command("sh", "-c", "exit 1")
				}
				return exec.Command("sh", "-c", "cat >/dev/null")
			case "osascript":
				if len(args) >= 2 && strings.Contains(args[1], "keystroke") {
					if mode == "paste-fail" {
						return exec.Command("sh", "-c", "exit 1")
					}
					return exec.Command("sh", "-c", "exit 0")
				}
				if mode == "enter-fail" {
					return exec.Command("sh", "-c", "exit 1")
				}
				return exec.Command("sh", "-c", "exit 0")
			default:
				return exec.Command("sh", "-c", "exit 0")
			}
		}
	}

	execCommand = makeRunner("success")
	if err := sendToMacApp(omnirouterconfig.Target{App: "ChatGPT", SendMode: omnirouterconfig.SendModePasteEnter, StartupDelayMs: 1}, "hello"); err != nil {
		t.Fatalf("sendToMacApp success returned error: %v", err)
	}

	tests := []struct {
		name        string
		mode        string
		errContains string
	}{
		{name: "open fail", mode: "open-fail", errContains: "open app"},
		{name: "copy fail", mode: "copy-fail", errContains: "copy prompt to clipboard"},
		{name: "paste fail", mode: "paste-fail", errContains: "paste prompt"},
		{name: "enter fail", mode: "enter-fail", errContains: "submit prompt"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			execCommand = makeRunner(tt.mode)
			err := sendToMacApp(omnirouterconfig.Target{App: "ChatGPT", SendMode: omnirouterconfig.SendModePasteEnter, StartupDelayMs: 1}, "hello")
			if err == nil || !strings.Contains(err.Error(), tt.errContains) {
				t.Fatalf("error = %v", err)
			}
		})
	}

	execCommand = makeRunner("success")
	if err := sendToMacApp(omnirouterconfig.Target{App: "ChatGPT", SendMode: omnirouterconfig.SendModePasteOnly, StartupDelayMs: 1}, "hello"); err != nil {
		t.Fatalf("sendToMacApp paste_only returned error: %v", err)
	}
}

package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadAndWriteConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	cfg := &Config{
		Targets: []Target{
			{Name: "ChatGPT", Type: "mac_app", App: "ChatGPT"},
		},
	}

	if err := WriteConfig(path, cfg); err != nil {
		t.Fatalf("WriteConfig returned error: %v", err)
	}

	loaded, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig returned error: %v", err)
	}

	if loaded.Targets[0].Name != "ChatGPT" {
		t.Fatalf("loaded target = %#v", loaded.Targets[0])
	}
}

func TestLoadConfigErrors(t *testing.T) {
	dir := t.TempDir()
	missingPath := filepath.Join(dir, "missing.json")
	if _, err := LoadConfig(missingPath); err == nil || !strings.Contains(err.Error(), "read config") {
		t.Fatalf("error = %v", err)
	}

	invalidPath := filepath.Join(dir, "invalid.json")
	if err := os.WriteFile(invalidPath, []byte("{"), 0644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if _, err := LoadConfig(invalidPath); err == nil || !strings.Contains(err.Error(), "parse config") {
		t.Fatalf("error = %v", err)
	}
}

func TestApplyDefaultsAndValidate_SetsDefaults(t *testing.T) {
	config := Config{
		Targets: []Target{
			{
				Name: "ChatGPT",
				Type: "mac_app",
				App:  "ChatGPT",
			},
		},
	}

	if err := config.ApplyDefaultsAndValidate(); err != nil {
		t.Fatalf("ApplyDefaultsAndValidate returned error: %v", err)
	}

	if config.ShowWindowOnStartup == nil || !*config.ShowWindowOnStartup {
		t.Fatalf("ShowWindowOnStartup default not applied")
	}
	if config.AutoHideAfterSend == nil || *config.AutoHideAfterSend {
		t.Fatalf("AutoHideAfterSend default not applied")
	}
	if config.HotkeyMode != "toggle" {
		t.Fatalf("HotkeyMode default = %q, want %q", config.HotkeyMode, "toggle")
	}
	if config.Targets[0].SendMode != "paste_enter" {
		t.Fatalf("SendMode default = %q, want %q", config.Targets[0].SendMode, "paste_enter")
	}
	if config.Targets[0].StartupDelayMs != 900 {
		t.Fatalf("StartupDelayMs default = %d, want %d", config.Targets[0].StartupDelayMs, 900)
	}
}

func TestApplyDefaultsAndValidate_RouterDefaults(t *testing.T) {
	configOpenAI := Config{
		Router: &Router{},
		Targets: []Target{
			{Name: "Auto", Type: "auto"},
			{Name: "ChatGPT", Type: "mac_app", App: "ChatGPT"},
		},
	}
	if err := configOpenAI.ApplyDefaultsAndValidate(); err != nil {
		t.Fatalf("ApplyDefaultsAndValidate returned error: %v", err)
	}
	if configOpenAI.Router.Provider != "openai" {
		t.Fatalf("Router.Provider default = %q, want %q", configOpenAI.Router.Provider, "openai")
	}
	if configOpenAI.Router.Model != "gpt-4o-mini" {
		t.Fatalf("Router.Model default = %q, want %q", configOpenAI.Router.Model, "gpt-4o-mini")
	}
	if configOpenAI.Router.APIKeyEnv != "OPENAI_API_KEY" {
		t.Fatalf("Router.APIKeyEnv default = %q, want %q", configOpenAI.Router.APIKeyEnv, "OPENAI_API_KEY")
	}

	configGemini := Config{
		Router: &Router{Provider: "gemini"},
		Targets: []Target{
			{Name: "Auto", Type: "auto"},
			{Name: "ChatGPT", Type: "mac_app", App: "ChatGPT"},
		},
	}
	if err := configGemini.ApplyDefaultsAndValidate(); err != nil {
		t.Fatalf("ApplyDefaultsAndValidate returned error: %v", err)
	}
	if configGemini.Router.Model != "gemini-1.5-flash" {
		t.Fatalf("Router.Model default = %q", configGemini.Router.Model)
	}

	configAnthropic := Config{
		Router: &Router{Provider: "anthropic"},
		Targets: []Target{
			{Name: "Auto", Type: "auto"},
			{Name: "ChatGPT", Type: "mac_app", App: "ChatGPT"},
		},
	}
	if err := configAnthropic.ApplyDefaultsAndValidate(); err != nil {
		t.Fatalf("ApplyDefaultsAndValidate returned error: %v", err)
	}
	if configAnthropic.Router.Model != "claude-3-5-sonnet-latest" {
		t.Fatalf("Router.Model default = %q", configAnthropic.Router.Model)
	}
}

func TestApplyDefaultsAndValidate_APIDefaults(t *testing.T) {
	tests := []struct {
		name       string
		provider   string
		wantModel  string
		wantKeyEnv string
	}{
		{name: "openai", provider: "openai", wantModel: "gpt-4o-mini", wantKeyEnv: "OPENAI_API_KEY"},
		{name: "gemini", provider: "gemini", wantModel: "gemini-1.5-flash", wantKeyEnv: "GEMINI_API_KEY"},
		{name: "anthropic", provider: "anthropic", wantModel: "claude-3-5-sonnet-latest", wantKeyEnv: "ANTHROPIC_API_KEY"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := Config{
				Targets: []Target{
					{Name: "API Target", Type: "api", Provider: tt.provider},
				},
			}
			if err := config.ApplyDefaultsAndValidate(); err != nil {
				t.Fatalf("ApplyDefaultsAndValidate returned error: %v", err)
			}
			if config.Targets[0].Model != tt.wantModel {
				t.Fatalf("API model default = %q, want %q", config.Targets[0].Model, tt.wantModel)
			}
			if config.Targets[0].APIKeyEnv != tt.wantKeyEnv {
				t.Fatalf("API key env default = %q, want %q", config.Targets[0].APIKeyEnv, tt.wantKeyEnv)
			}
		})
	}
}

func TestApplyDefaultsAndValidate_NormalizesFields(t *testing.T) {
	config := Config{
		DefaultTarget: "  ChatGPT  ",
		HotkeyMode:    "  LaUnChEr  ",
		Router: &Router{
			Provider: "  OPENAI  ",
		},
		Targets: []Target{
			{
				Name:     "  ChatGPT  ",
				Type:     "  mac_app  ",
				App:      "  ChatGPT  ",
				SendMode: "  paste_enter  ",
			},
		},
	}

	if err := config.ApplyDefaultsAndValidate(); err != nil {
		t.Fatalf("ApplyDefaultsAndValidate returned error: %v", err)
	}

	if config.DefaultTarget != "ChatGPT" || config.HotkeyMode != "launcher" || config.Router.Provider != "openai" {
		t.Fatalf("normalization unexpected: %#v", config)
	}
}

func TestApplyDefaultsAndValidate_Errors(t *testing.T) {
	tests := []struct {
		name        string
		config      Config
		errContains string
	}{
		{name: "missing targets", config: Config{}, errContains: "at least one target"},
		{name: "missing target name", config: Config{Targets: []Target{{Type: "mac_app", App: "ChatGPT"}}}, errContains: "missing name"},
		{name: "missing target type", config: Config{Targets: []Target{{Name: "ChatGPT", App: "ChatGPT"}}}, errContains: "missing type"},
		{name: "mac app missing app", config: Config{Targets: []Target{{Name: "ChatGPT", Type: "mac_app"}}}, errContains: "requires app"},
		{name: "auto target without router", config: Config{Targets: []Target{{Name: "Auto", Type: "auto"}}}, errContains: "requires router config"},
		{name: "invalid hotkey mode", config: Config{HotkeyMode: "bad", Targets: []Target{{Name: "ChatGPT", Type: "mac_app", App: "ChatGPT"}}}, errContains: "hotkeyMode must be one of"},
		{name: "invalid router provider", config: Config{Router: &Router{Provider: "invalid-provider"}, Targets: []Target{{Name: "ChatGPT", Type: "mac_app", App: "ChatGPT"}}}, errContains: "router.provider must be one of"},
		{name: "api target missing provider", config: Config{Targets: []Target{{Name: "API Target", Type: "api"}}}, errContains: "requires provider for type api"},
		{name: "api target invalid provider", config: Config{Targets: []Target{{Name: "API Target", Type: "api", Provider: "invalid-provider"}}}, errContains: "has unsupported provider"},
		{name: "default target not found", config: Config{DefaultTarget: "Claude", Targets: []Target{{Name: "ChatGPT", Type: "mac_app", App: "ChatGPT"}}}, errContains: "defaultTarget"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.ApplyDefaultsAndValidate()
			if err == nil || !strings.Contains(err.Error(), tt.errContains) {
				t.Fatalf("error = %v, want substring %q", err, tt.errContains)
			}
		})
	}
}

func TestConfigHelpers(t *testing.T) {
	config := Config{}

	if !config.ShouldShowWindowOnStartup() || config.ShouldAutoHideAfterSend() {
		t.Fatalf("default helper values unexpected")
	}
	if got := config.GetHotkeyMode(); got != "toggle" {
		t.Fatalf("GetHotkeyMode default = %q", got)
	}
	if got := config.GetWindowWidth(900); got != 900 {
		t.Fatalf("GetWindowWidth default = %d", got)
	}
	if got := config.GetWindowHeight(640); got != 640 {
		t.Fatalf("GetWindowHeight default = %d", got)
	}
	if got := config.GetLauncherWindowWidth(760); got != 760 {
		t.Fatalf("GetLauncherWindowWidth default = %d", got)
	}
	if got := config.GetLauncherWindowHeight(280); got != 280 {
		t.Fatalf("GetLauncherWindowHeight default = %d", got)
	}
	if _, _, ok := config.GetWindowPosition(); ok {
		t.Fatalf("GetWindowPosition should be unavailable without both coords")
	}
}

func TestFindTargetByName(t *testing.T) {
	config := Config{
		Targets: []Target{
			{Name: "ChatGPT", Type: "mac_app", App: "ChatGPT"},
			{Name: "Claude", Type: "mac_app", App: "Claude"},
		},
	}

	target, err := config.FindTargetByName("Claude")
	if err != nil {
		t.Fatalf("FindTargetByName returned error: %v", err)
	}
	if target.Name != "Claude" {
		t.Fatalf("target.Name = %q, want %q", target.Name, "Claude")
	}
	if _, err := config.FindTargetByName("Gemini"); err == nil {
		t.Fatalf("expected error for missing target")
	}
}

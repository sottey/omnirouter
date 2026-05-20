package main

import (
	"strings"
	"testing"
)

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
	config := Config{
		Router: &Router{},
		Targets: []Target{
			{
				Name: "Auto",
				Type: "auto",
			},
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

	if config.Router.Provider != "openai" {
		t.Fatalf("Router.Provider default = %q, want %q", config.Router.Provider, "openai")
	}
	if config.Router.Model != "gpt-4o-mini" {
		t.Fatalf("Router.Model default = %q, want %q", config.Router.Model, "gpt-4o-mini")
	}
	if config.Router.APIKeyEnv != "OPENAI_API_KEY" {
		t.Fatalf("Router.APIKeyEnv default = %q, want %q", config.Router.APIKeyEnv, "OPENAI_API_KEY")
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

	if config.DefaultTarget != "ChatGPT" {
		t.Fatalf("DefaultTarget normalization = %q, want %q", config.DefaultTarget, "ChatGPT")
	}
	if config.HotkeyMode != "launcher" {
		t.Fatalf("HotkeyMode normalization = %q, want %q", config.HotkeyMode, "launcher")
	}
	if config.Router.Provider != "openai" {
		t.Fatalf("Router.Provider normalization = %q, want %q", config.Router.Provider, "openai")
	}
	target := config.Targets[0]
	if target.Name != "ChatGPT" || target.Type != "mac_app" || target.App != "ChatGPT" || target.SendMode != "paste_enter" {
		t.Fatalf("target normalization unexpected: %#v", target)
	}
}

func TestApplyDefaultsAndValidate_Errors(t *testing.T) {
	tests := []struct {
		name        string
		config      Config
		errContains string
	}{
		{
			name:        "missing targets",
			config:      Config{},
			errContains: "at least one target",
		},
		{
			name: "missing target name",
			config: Config{
				Targets: []Target{{Type: "mac_app", App: "ChatGPT"}},
			},
			errContains: "missing name",
		},
		{
			name: "missing target type",
			config: Config{
				Targets: []Target{{Name: "ChatGPT", App: "ChatGPT"}},
			},
			errContains: "missing type",
		},
		{
			name: "mac app missing app",
			config: Config{
				Targets: []Target{{Name: "ChatGPT", Type: "mac_app"}},
			},
			errContains: "requires app",
		},
		{
			name: "auto target without router",
			config: Config{
				Targets: []Target{{Name: "Auto", Type: "auto"}},
			},
			errContains: "requires router config",
		},
		{
			name: "invalid hotkey mode",
			config: Config{
				HotkeyMode: "bad",
				Targets: []Target{
					{Name: "ChatGPT", Type: "mac_app", App: "ChatGPT"},
				},
			},
			errContains: "hotkeyMode must be one of",
		},
		{
			name: "invalid router provider",
			config: Config{
				Router: &Router{Provider: "anthropic"},
				Targets: []Target{
					{Name: "ChatGPT", Type: "mac_app", App: "ChatGPT"},
				},
			},
			errContains: "router.provider must be openai",
		},
		{
			name: "default target not found",
			config: Config{
				DefaultTarget: "Claude",
				Targets: []Target{
					{Name: "ChatGPT", Type: "mac_app", App: "ChatGPT"},
				},
			},
			errContains: "defaultTarget",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.ApplyDefaultsAndValidate()
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.errContains)
			}
			if !strings.Contains(err.Error(), tt.errContains) {
				t.Fatalf("error = %q, want substring %q", err.Error(), tt.errContains)
			}
		})
	}
}

func TestConfigHelpers(t *testing.T) {
	config := Config{}

	if !config.ShouldShowWindowOnStartup() {
		t.Fatalf("ShouldShowWindowOnStartup default should be true")
	}
	if config.ShouldAutoHideAfterSend() {
		t.Fatalf("ShouldAutoHideAfterSend default should be false")
	}
	if got := config.GetHotkeyMode(); got != "toggle" {
		t.Fatalf("GetHotkeyMode default = %q, want %q", got, "toggle")
	}
	if got := config.GetWindowWidth(900); got != 900 {
		t.Fatalf("GetWindowWidth default = %d, want %d", got, 900)
	}
	if got := config.GetWindowHeight(640); got != 640 {
		t.Fatalf("GetWindowHeight default = %d, want %d", got, 640)
	}
	if got := config.GetLauncherWindowWidth(760); got != 760 {
		t.Fatalf("GetLauncherWindowWidth default = %d, want %d", got, 760)
	}
	if got := config.GetLauncherWindowHeight(280); got != 280 {
		t.Fatalf("GetLauncherWindowHeight default = %d, want %d", got, 280)
	}
	if _, _, ok := config.GetWindowPosition(); ok {
		t.Fatalf("GetWindowPosition should be unavailable without both coords")
	}

	show := false
	hide := true
	hotkey := "launcher"
	width := 1000
	height := 700
	launcherWidth := 500
	launcherHeight := 300
	x := 15
	y := 30
	config.ShowWindowOnStartup = &show
	config.AutoHideAfterSend = &hide
	config.HotkeyMode = hotkey
	config.WindowWidth = &width
	config.WindowHeight = &height
	config.LauncherWindowWidth = &launcherWidth
	config.LauncherWindowHeight = &launcherHeight
	config.WindowX = &x
	config.WindowY = &y

	if config.ShouldShowWindowOnStartup() {
		t.Fatalf("ShouldShowWindowOnStartup should reflect false value")
	}
	if !config.ShouldAutoHideAfterSend() {
		t.Fatalf("ShouldAutoHideAfterSend should reflect true value")
	}
	if got := config.GetHotkeyMode(); got != "launcher" {
		t.Fatalf("GetHotkeyMode = %q, want %q", got, "launcher")
	}
	if got := config.GetWindowWidth(900); got != 1000 {
		t.Fatalf("GetWindowWidth = %d, want %d", got, 1000)
	}
	if got := config.GetWindowHeight(640); got != 700 {
		t.Fatalf("GetWindowHeight = %d, want %d", got, 700)
	}
	if got := config.GetLauncherWindowWidth(760); got != 500 {
		t.Fatalf("GetLauncherWindowWidth = %d, want %d", got, 500)
	}
	if got := config.GetLauncherWindowHeight(280); got != 300 {
		t.Fatalf("GetLauncherWindowHeight = %d, want %d", got, 300)
	}
	gotX, gotY, ok := config.GetWindowPosition()
	if !ok || gotX != 15 || gotY != 30 {
		t.Fatalf("GetWindowPosition = (%d,%d,%v), want (15,30,true)", gotX, gotY, ok)
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

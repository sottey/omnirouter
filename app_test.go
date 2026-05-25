package main

import (
	"path/filepath"
	"strings"
	"testing"

	omnirouterconfig "omnirouter/internal/config"
	"omnirouter/internal/core"
)

func TestRunDesktopApp_ConfigError(t *testing.T) {
	err := runDesktopApp(filepath.Join(t.TempDir(), "missing.json"))
	if err == nil || !strings.Contains(err.Error(), "read config") {
		t.Fatalf("error = %v", err)
	}
}

func TestAppConfigDelegation(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	cfg := &omnirouterconfig.Config{
		Targets: []omnirouterconfig.Target{
			{Name: "OpenAI API", Type: "api", Provider: "openai", Model: "gpt-4o-mini", APIKeyEnv: "OPENAI_API_KEY"},
		},
	}
	if err := cfg.ApplyDefaultsAndValidate(); err != nil {
		t.Fatalf("ApplyDefaultsAndValidate returned error: %v", err)
	}
	if err := omnirouterconfig.WriteConfig(configPath, cfg); err != nil {
		t.Fatalf("WriteConfig returned error: %v", err)
	}

	app := NewApp(cfg, configPath)
	targets, err := app.GetTargets()
	if err != nil || len(targets) != 1 {
		t.Fatalf("targets=%v err=%v", targets, err)
	}

	gotConfig, err := app.GetConfig()
	if err != nil || len(gotConfig.Targets) != 1 {
		t.Fatalf("config=%#v err=%v", gotConfig, err)
	}

	nextCfg := Config{
		DefaultTarget: "OpenAI API",
		Targets: []Target{
			{Name: "OpenAI API", Type: "api", Provider: "openai", Model: "gpt-4o-mini", APIKeyEnv: "OPENAI_API_KEY"},
		},
	}
	if err := app.SaveConfig(nextCfg); err != nil {
		t.Fatalf("SaveConfig returned error: %v", err)
	}
	if err := app.ReloadConfig(); err != nil {
		t.Fatalf("ReloadConfig returned error: %v", err)
	}

	gotConfig, err = app.GetConfig()
	if err != nil || gotConfig.DefaultTarget != "OpenAI API" {
		t.Fatalf("config=%#v err=%v", gotConfig, err)
	}
}

func TestAppNilConfigErrorsAndNoopMethods(t *testing.T) {
	app := NewApp(nil, "config.json")
	if _, err := app.GetTargets(); err == nil || !strings.Contains(err.Error(), "config not loaded") {
		t.Fatalf("error = %v", err)
	}
	if _, err := app.GetConfig(); err == nil || !strings.Contains(err.Error(), "config not loaded") {
		t.Fatalf("error = %v", err)
	}
	if _, err := app.SendPrompt("ChatGPT", "hello"); err == nil || !strings.Contains(err.Error(), "config not loaded") {
		t.Fatalf("error = %v", err)
	}
	if err := app.TestTarget("ChatGPT"); err == nil || !strings.Contains(err.Error(), "config not loaded") {
		t.Fatalf("error = %v", err)
	}

	app.ToggleMainWindow()
	app.ShowMainWindow()
	app.HideMainWindow()
	app.ShowLauncherWindow()
	app.OpenSettingsWindow()
	if app.ConsumeOpenSettingsRequest() {
		t.Fatalf("ConsumeOpenSettingsRequest should be false")
	}
	app.HandleGlobalHotkey()
	app.applyStoredWindowPosition()
}

func TestAppFlagsAndMenuWrappers(t *testing.T) {
	app := &App{quitting: true}
	if got := app.beforeClose(nil); got {
		t.Fatalf("beforeClose should be false when quitting")
	}

	app.openSettings = true
	if !app.ConsumeOpenSettingsRequest() {
		t.Fatalf("ConsumeOpenSettingsRequest should be true")
	}
	if app.ConsumeOpenSettingsRequest() {
		t.Fatalf("second ConsumeOpenSettingsRequest should be false")
	}

	activeApp = nil
	omniRouterShowWindow()
	omniRouterOpenSettings()
	omniRouterToggleWindow()
	omniRouterQuit()
	omniRouterReloadConfig()

	activeApp = &App{service: core.NewService(filepath.Join(t.TempDir(), "missing.json"), nil)}
	omniRouterShowWindow()
	omniRouterOpenSettings()
	omniRouterToggleWindow()
	omniRouterQuit()
	omniRouterReloadConfig()
	activeApp = nil
}

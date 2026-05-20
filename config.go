package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	if err := config.ApplyDefaultsAndValidate(); err != nil {
		return nil, err
	}

	return &config, nil
}

func WriteConfig(path string, config *Config) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	data = append(data, '\n')

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	return nil
}

func (c *Config) ApplyDefaultsAndValidate() error {
	c.DefaultTarget = strings.TrimSpace(c.DefaultTarget)

	if c.ShowWindowOnStartup == nil {
		showWindowOnStartup := true
		c.ShowWindowOnStartup = &showWindowOnStartup
	}

	if c.AutoHideAfterSend == nil {
		autoHideAfterSend := false
		c.AutoHideAfterSend = &autoHideAfterSend
	}

	c.HotkeyMode = strings.TrimSpace(strings.ToLower(c.HotkeyMode))
	if c.HotkeyMode == "" {
		c.HotkeyMode = "toggle"
	}
	if c.HotkeyMode != "toggle" && c.HotkeyMode != "launcher" {
		return fmt.Errorf("hotkeyMode must be one of: toggle, launcher")
	}

	if c.Router != nil {
		c.Router.Provider = strings.TrimSpace(strings.ToLower(c.Router.Provider))
		c.Router.Model = strings.TrimSpace(c.Router.Model)
		c.Router.APIKeyEnv = strings.TrimSpace(c.Router.APIKeyEnv)
		c.Router.SystemPrompt = strings.TrimSpace(c.Router.SystemPrompt)

		if c.Router.Provider == "" {
			c.Router.Provider = "openai"
		}

		if c.Router.Provider != "openai" {
			return fmt.Errorf("router.provider must be openai")
		}

		if c.Router.Model == "" {
			c.Router.Model = "gpt-4o-mini"
		}

		if c.Router.APIKeyEnv == "" {
			c.Router.APIKeyEnv = "OPENAI_API_KEY"
		}
	}

	if len(c.Targets) == 0 {
		return fmt.Errorf("config must contain at least one target")
	}

	for i := range c.Targets {
		target := &c.Targets[i]

		target.Name = strings.TrimSpace(target.Name)
		target.Type = strings.TrimSpace(target.Type)
		target.App = strings.TrimSpace(target.App)
		target.URL = strings.TrimSpace(target.URL)
		target.Description = strings.TrimSpace(target.Description)
		target.Shortcut = strings.TrimSpace(target.Shortcut)
		target.SendMode = strings.TrimSpace(strings.ToLower(target.SendMode))

		if target.Name == "" {
			return fmt.Errorf("target at index %d is missing name", i)
		}

		if target.Type == "" {
			return fmt.Errorf("target %q is missing type", target.Name)
		}

		if target.SendMode == "" {
			target.SendMode = sendModePasteEnter
		}
		if target.SendMode != sendModePasteEnter && target.SendMode != sendModePasteOnly {
			return fmt.Errorf("target %q has unsupported sendMode %q", target.Name, target.SendMode)
		}

		if target.StartupDelayMs == 0 {
			target.StartupDelayMs = 900
		}

		if target.Type == "mac_app" && target.App == "" {
			return fmt.Errorf("target %q requires app for type mac_app", target.Name)
		}

		if target.Type == "auto" && c.Router == nil {
			return fmt.Errorf("target %q requires router config for type auto", target.Name)
		}
	}

	if c.DefaultTarget != "" {
		if _, err := c.FindTargetByName(c.DefaultTarget); err != nil {
			return fmt.Errorf("defaultTarget %q not found in targets", c.DefaultTarget)
		}
	}

	return nil
}

func (c *Config) ShouldShowWindowOnStartup() bool {
	if c.ShowWindowOnStartup == nil {
		return true
	}

	return *c.ShowWindowOnStartup
}

func (c *Config) ShouldAutoHideAfterSend() bool {
	if c.AutoHideAfterSend == nil {
		return false
	}

	return *c.AutoHideAfterSend
}

func (c *Config) GetHotkeyMode() string {
	if c.HotkeyMode == "" {
		return "toggle"
	}

	return c.HotkeyMode
}

func (c *Config) GetWindowWidth(defaultWidth int) int {
	if c.WindowWidth == nil || *c.WindowWidth <= 0 {
		return defaultWidth
	}

	return *c.WindowWidth
}

func (c *Config) GetWindowHeight(defaultHeight int) int {
	if c.WindowHeight == nil || *c.WindowHeight <= 0 {
		return defaultHeight
	}

	return *c.WindowHeight
}

func (c *Config) GetLauncherWindowWidth(defaultWidth int) int {
	if c.LauncherWindowWidth == nil || *c.LauncherWindowWidth <= 0 {
		return defaultWidth
	}

	return *c.LauncherWindowWidth
}

func (c *Config) GetLauncherWindowHeight(defaultHeight int) int {
	if c.LauncherWindowHeight == nil || *c.LauncherWindowHeight <= 0 {
		return defaultHeight
	}

	return *c.LauncherWindowHeight
}

func (c *Config) GetWindowPosition() (int, int, bool) {
	if c.WindowX == nil || c.WindowY == nil {
		return 0, 0, false
	}

	return *c.WindowX, *c.WindowY, true
}

func (c *Config) FindTargetByName(name string) (*Target, error) {
	for i := range c.Targets {
		if c.Targets[i].Name == name {
			return &c.Targets[i], nil
		}
	}

	return nil, fmt.Errorf("target %q not found", name)
}

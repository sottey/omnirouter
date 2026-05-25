package main

type Config struct {
	ShowWindowOnStartup  *bool    `json:"showWindowOnStartup"`
	DefaultTarget        string   `json:"defaultTarget,omitempty"`
	AutoHideAfterSend    *bool    `json:"autoHideAfterSend,omitempty"`
	HotkeyMode           string   `json:"hotkeyMode,omitempty"`
	LauncherWindowWidth  *int     `json:"launcherWindowWidth,omitempty"`
	LauncherWindowHeight *int     `json:"launcherWindowHeight,omitempty"`
	WindowWidth          *int     `json:"windowWidth,omitempty"`
	WindowHeight         *int     `json:"windowHeight,omitempty"`
	WindowX              *int     `json:"windowX,omitempty"`
	WindowY              *int     `json:"windowY,omitempty"`
	Router               *Router  `json:"router,omitempty"`
	Targets              []Target `json:"targets"`
}

type Router struct {
	Provider     string `json:"provider"`
	Model        string `json:"model"`
	APIKeyEnv    string `json:"apiKeyEnv"`
	SystemPrompt string `json:"systemPrompt,omitempty"`
}

type Target struct {
	Name           string `json:"name"`
	Type           string `json:"type"`
	App            string `json:"app,omitempty"`
	URL            string `json:"url,omitempty"`
	Description    string `json:"description,omitempty"`
	Shortcut       string `json:"shortcut,omitempty"`
	SendMode       string `json:"sendMode"`
	StartupDelayMs int    `json:"startupDelayMs"`
	Provider       string `json:"provider,omitempty"`
	Model          string `json:"model,omitempty"`
	APIKeyEnv      string `json:"apiKeyEnv,omitempty"`
	SystemPrompt   string `json:"systemPrompt,omitempty"`
}

type SendPromptResult struct {
	TargetName   string `json:"targetName"`
	ResponseText string `json:"responseText,omitempty"`
	IsAPI        bool   `json:"isApi"`
}

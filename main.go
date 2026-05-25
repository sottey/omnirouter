package main

import (
	"context"
	"embed"
	"fmt"
	stdruntime "runtime"
	"time"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	rt "github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed frontend/dist
var assets embed.FS

type App struct {
	ctx          context.Context
	config       *Config
	configPath   string
	launcherMode bool
	windowHidden bool
	quitting     bool
	openSettings bool
}

func NewApp(config *Config, configPath string) *App {
	return &App{config: config, configPath: configPath}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.windowHidden = !a.config.ShouldShowWindowOnStartup()
	activeApp = a
	setupMenuBar(ctx)
	setDockVisible(!a.windowHidden)
	a.applyStoredWindowPosition()
}

func (a *App) shutdown(ctx context.Context) {
	teardownMenuBar()
	if activeApp == a {
		activeApp = nil
	}
}

func (a *App) beforeClose(ctx context.Context) bool {
	if stdruntime.GOOS != "darwin" || a.quitting {
		return false
	}

	a.HideMainWindow()
	return true
}

func (a *App) GetTargets() ([]Target, error) {
	if a.config == nil {
		return nil, fmt.Errorf("config not loaded")
	}

	return a.config.Targets, nil
}

func (a *App) GetConfig() (Config, error) {
	if a.config == nil {
		return Config{}, fmt.Errorf("config not loaded")
	}

	return *a.config, nil
}

func (a *App) SaveConfig(config Config) error {
	if err := config.ApplyDefaultsAndValidate(); err != nil {
		return err
	}

	if err := WriteConfig(a.configPath, &config); err != nil {
		return err
	}

	a.config = &config
	return nil
}

func (a *App) SendPrompt(targetName string, prompt string) (SendPromptResult, error) {
	if a.config == nil {
		return SendPromptResult{}, fmt.Errorf("config not loaded")
	}

	target, chosenTargetName, err := ResolveTarget(a.config, targetName, prompt)
	if err != nil {
		return SendPromptResult{}, err
	}

	if target.Type == "api" {
		responseText, err := CallLLMAPI(target.Provider, target.Model, target.APIKeyEnv, target.SystemPrompt, prompt)
		if err != nil {
			return SendPromptResult{}, err
		}
		return SendPromptResult{
			TargetName:   chosenTargetName,
			ResponseText: responseText,
			IsAPI:        true,
		}, nil
	}

	if err := SendPromptToTarget(*target, prompt); err != nil {
		return SendPromptResult{}, err
	}

	return SendPromptResult{
		TargetName: chosenTargetName,
		IsAPI:      false,
	}, nil
}

func (a *App) TestTarget(targetName string) error {
	if a.config == nil {
		return fmt.Errorf("config not loaded")
	}

	target, err := a.config.FindTargetByName(targetName)
	if err != nil {
		return err
	}
	if target.Type == "auto" {
		return fmt.Errorf("test target must not be auto")
	}

	testPrompt := fmt.Sprintf("OmniRouter test message (%s)", time.Now().Format(time.RFC3339))
	return SendPromptToTarget(*target, testPrompt)
}

func (a *App) ShowMainWindow() {
	if a.ctx == nil {
		return
	}

	a.launcherMode = false
	setDockVisible(true)
	rt.WindowUnminimise(a.ctx)
	rt.WindowShow(a.ctx)
	a.windowHidden = false
}

func (a *App) OpenSettingsWindow() {
	if a.ctx == nil {
		return
	}

	a.ShowMainWindow()
	a.openSettings = true
}

func (a *App) ConsumeOpenSettingsRequest() bool {
	if !a.openSettings {
		return false
	}

	a.openSettings = false
	return true
}

func (a *App) HideMainWindow() {
	if a.ctx == nil {
		return
	}

	rt.WindowHide(a.ctx)
	setDockVisible(false)
	a.windowHidden = true
}

func (a *App) ShowLauncherWindow() {
	if a.ctx == nil {
		return
	}

	a.launcherMode = true
	setDockVisible(true)
	rt.WindowSetSize(a.ctx, a.config.GetLauncherWindowWidth(760), a.config.GetLauncherWindowHeight(280))
	rt.WindowCenter(a.ctx)
	rt.WindowUnminimise(a.ctx)
	rt.WindowShow(a.ctx)
	a.windowHidden = false
}

func (a *App) ToggleMainWindow() {
	if a.ctx == nil {
		return
	}

	if a.windowHidden || rt.WindowIsMinimised(a.ctx) {
		a.ShowMainWindow()
		return
	}

	a.HideMainWindow()
}

func (a *App) Quit() {
	if a.ctx == nil {
		return
	}

	a.quitting = true
	teardownMenuBar()
	rt.Quit(a.ctx)
}

func (a *App) SaveWindowState() error {
	if a.config == nil {
		return fmt.Errorf("config not loaded")
	}

	if a.ctx == nil || a.launcherMode {
		return nil
	}

	width, height := rt.WindowGetSize(a.ctx)
	x, y := rt.WindowGetPosition(a.ctx)

	if width <= 0 || height <= 0 {
		return fmt.Errorf("window size must be positive")
	}

	latestConfig, err := LoadConfig(a.configPath)
	if err != nil {
		return err
	}

	// Avoid redundant writes if window state has not changed
	if latestConfig.WindowWidth != nil && *latestConfig.WindowWidth == width &&
		latestConfig.WindowHeight != nil && *latestConfig.WindowHeight == height &&
		latestConfig.WindowX != nil && *latestConfig.WindowX == x &&
		latestConfig.WindowY != nil && *latestConfig.WindowY == y {
		return nil
	}

	latestConfig.WindowWidth = &width
	latestConfig.WindowHeight = &height
	latestConfig.WindowX = &x
	latestConfig.WindowY = &y

	if err := WriteConfig(a.configPath, latestConfig); err != nil {
		return err
	}

	a.config = latestConfig
	return nil
}

func (a *App) ReloadConfig() error {
	config, err := LoadConfig(a.configPath)
	if err != nil {
		return err
	}

	a.config = config
	return nil
}

func (a *App) HandleGlobalHotkey() {
	if a.ctx == nil {
		return
	}

	if a.windowHidden || rt.WindowIsMinimised(a.ctx) {
		if a.config.GetHotkeyMode() == "launcher" {
			a.ShowLauncherWindow()
			return
		}

		a.ShowMainWindow()
		return
	}

	a.HideMainWindow()
}

func (a *App) applyStoredWindowPosition() {
	if a.ctx == nil || a.config == nil {
		return
	}

	x, y, ok := a.config.GetWindowPosition()
	if !ok {
		return
	}

	rt.WindowSetPosition(a.ctx, x, y)
}

func main() {
	configPath := "config.json"
	config, err := LoadConfig(configPath)
	if err != nil {
		panic(err)
	}

	app := NewApp(config, configPath)

	err = wails.Run(&options.App{
		Title:       "OmniRouter",
		Width:       config.GetWindowWidth(900),
		Height:      config.GetWindowHeight(640),
		StartHidden: !config.ShouldShowWindowOnStartup(),
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnStartup:     app.startup,
		OnShutdown:    app.shutdown,
		OnBeforeClose: app.beforeClose,
		Bind: []interface{}{
			app,
		},
	})
	if err != nil {
		panic(err)
	}
}

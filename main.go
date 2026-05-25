package main

import (
	"context"
	"embed"
	"fmt"
	omnirouterconfig "omnirouter/internal/config"
	"omnirouter/internal/core"
	stdruntime "runtime"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	rt "github.com/wailsapp/wails/v2/pkg/runtime"
)

//go:embed frontend/dist
var assets embed.FS

type App struct {
	ctx          context.Context
	configPath   string
	service      *core.Service
	launcherMode bool
	windowHidden bool
	quitting     bool
	openSettings bool
}

func NewApp(config *Config, configPath string) *App {
	return &App{
		configPath: configPath,
		service:    core.NewService(configPath, config),
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	config, err := a.service.GetConfig()
	if err == nil {
		a.windowHidden = !config.ShouldShowWindowOnStartup()
	}
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
	config, err := a.service.GetConfig()
	if err != nil {
		return nil, err
	}

	return config.Targets, nil
}

func (a *App) GetConfig() (Config, error) {
	return a.service.GetConfig()
}

func (a *App) SaveConfig(config Config) error {
	return a.service.SaveConfig(config)
}

func (a *App) SendPrompt(targetName string, prompt string) (SendPromptResult, error) {
	return a.service.SendPrompt(targetName, prompt)
}

func (a *App) TestTarget(targetName string) error {
	return a.service.TestTarget(targetName)
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

	config, err := a.service.GetConfig()
	if err != nil {
		return
	}

	a.launcherMode = true
	setDockVisible(true)
	rt.WindowSetSize(a.ctx, config.GetLauncherWindowWidth(760), config.GetLauncherWindowHeight(280))
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
	if a.ctx == nil || a.launcherMode {
		return nil
	}

	width, height := rt.WindowGetSize(a.ctx)
	x, y := rt.WindowGetPosition(a.ctx)

	if width <= 0 || height <= 0 {
		return fmt.Errorf("window size must be positive")
	}

	latestConfig, err := omnirouterconfig.LoadConfig(a.configPath)
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

	if err := omnirouterconfig.WriteConfig(a.configPath, latestConfig); err != nil {
		return err
	}

	a.service = core.NewService(a.configPath, latestConfig)
	return nil
}

func (a *App) ReloadConfig() error {
	return a.service.ReloadConfig()
}

func (a *App) HandleGlobalHotkey() {
	if a.ctx == nil {
		return
	}

	config, err := a.service.GetConfig()
	if err != nil {
		return
	}

	if a.windowHidden || rt.WindowIsMinimised(a.ctx) {
		if config.GetHotkeyMode() == "launcher" {
			a.ShowLauncherWindow()
			return
		}

		a.ShowMainWindow()
		return
	}

	a.HideMainWindow()
}

func (a *App) applyStoredWindowPosition() {
	if a.ctx == nil {
		return
	}

	config, err := a.service.GetConfig()
	if err != nil {
		return
	}

	x, y, ok := config.GetWindowPosition()
	if !ok {
		return
	}

	rt.WindowSetPosition(a.ctx, x, y)
}

func runDesktopApp(configPath string) error {
	config, err := omnirouterconfig.LoadConfig(configPath)
	if err != nil {
		return err
	}

	app := NewApp(config, configPath)

	return wails.Run(&options.App{
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
}

func main() {
	if err := runDesktopApp("config.json"); err != nil {
		panic(err)
	}
}

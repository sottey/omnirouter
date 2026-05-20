//go:build darwin

package main

/*
#cgo CFLAGS: -x objective-c -fobjc-arc -fblocks
#cgo LDFLAGS: -framework Cocoa -framework Carbon -framework UniformTypeIdentifiers
void OmniRouterSetAppIcon(const unsigned char *bytes, int length);
void OmniRouterApplyAppIcon(void);
void OmniRouterApplyDockTileIcon(void);
void OmniRouterClearDockTileIcon(void);
void OmniRouterSetStatusBarIcon(const unsigned char *bytes, int length);
void OmniRouterSetRegularActivationPolicy(void);
void OmniRouterSetupStatusItem(void);
void OmniRouterRegisterHotKey(void);
void OmniRouterUnregisterHotKey(void);
void OmniRouterSetAccessoryActivationPolicy(void);
*/
import "C"

import (
	"context"
	_ "embed"
	"unsafe"
)

//go:embed frontend/images/omnirouter_128x128.png
var omniRouterIconPNG []byte

//go:embed frontend/images/omnirouter_status_36x36.png
var omniRouterStatusIconPNG []byte

var activeApp *App

//export omniRouterShowWindow
func omniRouterShowWindow() {
	if activeApp == nil {
		return
	}

	activeApp.ShowMainWindow()
}

//export omniRouterOpenSettings
func omniRouterOpenSettings() {
	if activeApp == nil {
		return
	}

	activeApp.OpenSettingsWindow()
}

//export omniRouterToggleWindow
func omniRouterToggleWindow() {
	if activeApp == nil {
		return
	}

	activeApp.HandleGlobalHotkey()
}

//export omniRouterQuit
func omniRouterQuit() {
	if activeApp == nil {
		return
	}

	activeApp.Quit()
}

//export omniRouterReloadConfig
func omniRouterReloadConfig() {
	if activeApp == nil {
		return
	}

	_ = activeApp.ReloadConfig()
}

func setupMenuBar(ctx context.Context) {
	C.OmniRouterSetAccessoryActivationPolicy()
	if len(omniRouterIconPNG) > 0 {
		C.OmniRouterSetAppIcon((*C.uchar)(unsafe.Pointer(&omniRouterIconPNG[0])), C.int(len(omniRouterIconPNG)))
	}
	if len(omniRouterStatusIconPNG) > 0 {
		C.OmniRouterSetStatusBarIcon((*C.uchar)(unsafe.Pointer(&omniRouterStatusIconPNG[0])), C.int(len(omniRouterStatusIconPNG)))
	}
	C.OmniRouterSetupStatusItem()
	C.OmniRouterRegisterHotKey()
}

func setDockVisible(visible bool) {
	if visible {
		C.OmniRouterApplyAppIcon()
		C.OmniRouterSetRegularActivationPolicy()
		C.OmniRouterApplyDockTileIcon()
		return
	}

	C.OmniRouterClearDockTileIcon()
	C.OmniRouterSetAccessoryActivationPolicy()
}

func teardownMenuBar() {
	C.OmniRouterUnregisterHotKey()
}

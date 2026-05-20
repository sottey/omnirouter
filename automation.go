package main

import (
	"bytes"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

const (
	sendModePasteEnter = "paste_enter"
	sendModePasteOnly  = "paste_only"
)

func SendPromptToTarget(target Target, prompt string) error {
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return fmt.Errorf("prompt is required")
	}

	if runtime.GOOS != "darwin" {
		return fmt.Errorf("this MVP currently supports macOS only")
	}

	switch target.Type {
	case "mac_app":
		return sendToMacApp(target, prompt)
	default:
		return fmt.Errorf("unsupported target type %q", target.Type)
	}
}

func sendToMacApp(target Target, prompt string) error {
	if target.SendMode != sendModePasteEnter && target.SendMode != sendModePasteOnly {
		return fmt.Errorf("unsupported sendMode %q", target.SendMode)
	}

	// Backup clipboard
	oldClipboard, err := exec.Command("pbpaste").Output()
	var hasOldClipboard bool
	if err == nil {
		hasOldClipboard = true
	}

	if err := exec.Command("open", "-a", target.App).Run(); err != nil {
		return fmt.Errorf("open app %q: %w", target.App, err)
	}

	time.Sleep(time.Duration(target.StartupDelayMs) * time.Millisecond)

	copyCmd := exec.Command("pbcopy")
	copyCmd.Stdin = strings.NewReader(prompt)
	if err := copyCmd.Run(); err != nil {
		return fmt.Errorf("copy prompt to clipboard: %w", err)
	}

	pasteCmd := exec.Command(
		"osascript",
		"-e",
		`tell application "System Events" to keystroke "v" using command down`,
	)
	if err := pasteCmd.Run(); err != nil {
		return fmt.Errorf("paste prompt: %w", err)
	}

	if target.SendMode == sendModePasteEnter {
		enterCmd := exec.Command(
			"osascript",
			"-e",
			`tell application "System Events" to key code 36`,
		)
		if err := enterCmd.Run(); err != nil {
			return fmt.Errorf("submit prompt: %w", err)
		}
	}

	// Restore clipboard in the background after a short delay
	if hasOldClipboard {
		go func() {
			time.Sleep(500 * time.Millisecond)
			restoreCmd := exec.Command("pbcopy")
			restoreCmd.Stdin = bytes.NewReader(oldClipboard)
			_ = restoreCmd.Run()
		}()
	}

	return nil
}

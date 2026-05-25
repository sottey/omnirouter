package core

import (
	"bytes"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	omnirouterconfig "omnirouter/internal/config"
)

var execCommand = exec.Command
var sleep = time.Sleep

func SendPromptToTarget(target omnirouterconfig.Target, prompt string) error {
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

func sendToMacApp(target omnirouterconfig.Target, prompt string) error {
	if target.SendMode != omnirouterconfig.SendModePasteEnter && target.SendMode != omnirouterconfig.SendModePasteOnly {
		return fmt.Errorf("unsupported sendMode %q", target.SendMode)
	}

	oldClipboard, err := execCommand("pbpaste").Output()
	var hasOldClipboard bool
	if err == nil {
		hasOldClipboard = true
	}

	if err := execCommand("open", "-a", target.App).Run(); err != nil {
		return fmt.Errorf("open app %q: %w", target.App, err)
	}

	sleep(time.Duration(target.StartupDelayMs) * time.Millisecond)

	copyCmd := execCommand("pbcopy")
	copyCmd.Stdin = strings.NewReader(prompt)
	if err := copyCmd.Run(); err != nil {
		return fmt.Errorf("copy prompt to clipboard: %w", err)
	}

	pasteCmd := execCommand(
		"osascript",
		"-e",
		`tell application "System Events" to keystroke "v" using command down`,
	)
	if err := pasteCmd.Run(); err != nil {
		return fmt.Errorf("paste prompt: %w", err)
	}

	if target.SendMode == omnirouterconfig.SendModePasteEnter {
		enterCmd := execCommand(
			"osascript",
			"-e",
			`tell application "System Events" to key code 36`,
		)
		if err := enterCmd.Run(); err != nil {
			return fmt.Errorf("submit prompt: %w", err)
		}
	}

	if hasOldClipboard {
		go func() {
			sleep(500 * time.Millisecond)
			restoreCmd := execCommand("pbcopy")
			restoreCmd.Stdin = bytes.NewReader(oldClipboard)
			_ = restoreCmd.Run()
		}()
	}

	return nil
}

package main

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"
)

func newSendCmd(deps commandDeps) *cobra.Command {
	var configPath string
	var targetName string
	var prompt string

	cmd := &cobra.Command{
		Use:   "send",
		Short: "Send a prompt to a configured OmniRouter target",
		RunE: func(cmd *cobra.Command, args []string) error {
			targetName = strings.TrimSpace(targetName)
			prompt = strings.TrimSpace(prompt)

			if targetName == "" {
				return fmt.Errorf("target is required")
			}

			if prompt == "" {
				stdinPrompt, err := readPromptFromStdin(deps.stdin, deps.stdinIsTerminal)
				if err != nil {
					return err
				}
				prompt = stdinPrompt
			}

			if prompt == "" {
				return fmt.Errorf("prompt is required")
			}

			sender, err := deps.newSender(configPath)
			if err != nil {
				return err
			}

			result, err := sender.SendPrompt(targetName, prompt)
			if err != nil {
				return err
			}

			if result.TargetName != "" && result.TargetName != targetName {
				_, _ = fmt.Fprintf(deps.stderr, "Auto selected target: %s\n", result.TargetName)
			}

			if result.IsAPI {
				_, _ = fmt.Fprintln(deps.stdout, result.ResponseText)
				return nil
			}

			if result.TargetName != "" {
				_, _ = fmt.Fprintf(deps.stdout, "Sent to %s\n", result.TargetName)
				return nil
			}

			_, _ = fmt.Fprintln(deps.stdout, "Prompt sent")
			return nil
		},
	}

	cmd.Flags().StringVar(&configPath, "config", "config.json", "path to config file")
	cmd.Flags().StringVar(&targetName, "target", "", "target name from config")
	cmd.Flags().StringVar(&prompt, "prompt", "", "prompt text; if omitted, stdin is used")

	return cmd
}

func readPromptFromStdin(stdin io.Reader, stdinIsTerminal func() (bool, error)) (string, error) {
	isTerminal, err := stdinIsTerminal()
	if err != nil {
		return "", err
	}

	if isTerminal {
		return "", fmt.Errorf("prompt is required")
	}

	data, err := io.ReadAll(stdin)
	if err != nil {
		return "", fmt.Errorf("read stdin: %w", err)
	}

	return strings.TrimSpace(string(data)), nil
}

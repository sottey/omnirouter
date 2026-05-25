package main

import (
	"fmt"
	"io"
	"os"

	omnirouterconfig "omnirouter/internal/config"
	"omnirouter/internal/core"

	"github.com/spf13/cobra"
)

type promptSender interface {
	SendPrompt(targetName string, prompt string) (omnirouterconfig.SendPromptResult, error)
}

type commandDeps struct {
	stdin           io.Reader
	stdout          io.Writer
	stderr          io.Writer
	stdinIsTerminal func() (bool, error)
	newSender       func(configPath string) (promptSender, error)
}

func newRootCmd(deps commandDeps) *cobra.Command {
	cmd := &cobra.Command{
		Use:           "omnirouter-cli",
		Short:         "Send prompts through OmniRouter from the command line",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	cmd.SetIn(deps.stdin)
	cmd.SetOut(deps.stdout)
	cmd.SetErr(deps.stderr)
	cmd.AddCommand(newSendCmd(deps))

	return cmd
}

func defaultCommandDeps() commandDeps {
	return commandDeps{
		stdin:  os.Stdin,
		stdout: os.Stdout,
		stderr: os.Stderr,
		stdinIsTerminal: func() (bool, error) {
			info, err := os.Stdin.Stat()
			if err != nil {
				return false, fmt.Errorf("stat stdin: %w", err)
			}

			return info.Mode()&os.ModeCharDevice != 0, nil
		},
		newSender: newServiceSender,
	}
}

func newServiceSender(configPath string) (promptSender, error) {
	cfg, err := omnirouterconfig.LoadConfig(configPath)
	if err != nil {
		return nil, err
	}

	return core.NewService(configPath, cfg), nil
}

package main

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	omnirouterconfig "omnirouter/internal/config"
)

type fakeSender struct {
	result       omnirouterconfig.SendPromptResult
	err          error
	calledTarget string
	calledPrompt string
}

func (f *fakeSender) SendPrompt(targetName string, prompt string) (omnirouterconfig.SendPromptResult, error) {
	f.calledTarget = targetName
	f.calledPrompt = prompt
	return f.result, f.err
}

func TestRootCmd_NoArgsShowsHelp(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	cmd := newRootCmd(commandDeps{
		stdin:  strings.NewReader(""),
		stdout: stdout,
		stderr: stderr,
		stdinIsTerminal: func() (bool, error) {
			return true, nil
		},
		newSender: func(configPath string) (promptSender, error) {
			t.Fatalf("newSender should not be called")
			return nil, nil
		},
	})
	cmd.SetArgs([]string{})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	if got := stdout.String(); !strings.Contains(got, "Usage:") || !strings.Contains(got, "send") {
		t.Fatalf("stdout = %q", got)
	}
	if got := stderr.String(); got != "" {
		t.Fatalf("stderr = %q", got)
	}
}

func TestSendCmd_ExplicitPrompt(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	sender := &fakeSender{
		result: omnirouterconfig.SendPromptResult{TargetName: "ChatGPT"},
	}

	cmd := newRootCmd(commandDeps{
		stdin:  strings.NewReader(""),
		stdout: stdout,
		stderr: stderr,
		stdinIsTerminal: func() (bool, error) {
			return true, nil
		},
		newSender: func(configPath string) (promptSender, error) {
			if configPath != "config.json" {
				t.Fatalf("configPath = %q", configPath)
			}
			return sender, nil
		},
	})
	cmd.SetArgs([]string{"send", "--target", "ChatGPT", "--prompt", "hello"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	if sender.calledTarget != "ChatGPT" || sender.calledPrompt != "hello" {
		t.Fatalf("SendPrompt called with target=%q prompt=%q", sender.calledTarget, sender.calledPrompt)
	}
	if got := stdout.String(); got != "Sent to ChatGPT\n" {
		t.Fatalf("stdout = %q", got)
	}
	if got := stderr.String(); got != "" {
		t.Fatalf("stderr = %q", got)
	}
}

func TestSendCmd_StdinPrompt(t *testing.T) {
	stdout := &bytes.Buffer{}
	sender := &fakeSender{
		result: omnirouterconfig.SendPromptResult{TargetName: "ChatGPT"},
	}

	cmd := newRootCmd(commandDeps{
		stdin:  strings.NewReader("  prompt from stdin  "),
		stdout: stdout,
		stderr: &bytes.Buffer{},
		stdinIsTerminal: func() (bool, error) {
			return false, nil
		},
		newSender: func(configPath string) (promptSender, error) {
			return sender, nil
		},
	})
	cmd.SetArgs([]string{"send", "--target", "ChatGPT"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	if sender.calledPrompt != "prompt from stdin" {
		t.Fatalf("prompt = %q", sender.calledPrompt)
	}
}

func TestSendCmd_APITargetOutput(t *testing.T) {
	stdout := &bytes.Buffer{}
	sender := &fakeSender{
		result: omnirouterconfig.SendPromptResult{
			TargetName:   "OpenAI API",
			ResponseText: "response text",
			IsAPI:        true,
		},
	}

	cmd := newRootCmd(commandDeps{
		stdin:  strings.NewReader(""),
		stdout: stdout,
		stderr: &bytes.Buffer{},
		stdinIsTerminal: func() (bool, error) {
			return true, nil
		},
		newSender: func(configPath string) (promptSender, error) {
			return sender, nil
		},
	})
	cmd.SetArgs([]string{"send", "--target", "OpenAI API", "--prompt", "hello"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	if got := stdout.String(); got != "response text\n" {
		t.Fatalf("stdout = %q", got)
	}
}

func TestSendCmd_AutoTargetNotice(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	sender := &fakeSender{
		result: omnirouterconfig.SendPromptResult{TargetName: "Claude"},
	}

	cmd := newRootCmd(commandDeps{
		stdin:  strings.NewReader(""),
		stdout: stdout,
		stderr: stderr,
		stdinIsTerminal: func() (bool, error) {
			return true, nil
		},
		newSender: func(configPath string) (promptSender, error) {
			return sender, nil
		},
	})
	cmd.SetArgs([]string{"send", "--target", "Auto", "--prompt", "hello"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	if got := stderr.String(); got != "Auto selected target: Claude\n" {
		t.Fatalf("stderr = %q", got)
	}
	if got := stdout.String(); got != "Sent to Claude\n" {
		t.Fatalf("stdout = %q", got)
	}
}

func TestSendCmd_MissingTarget(t *testing.T) {
	cmd := newRootCmd(commandDeps{
		stdin:  strings.NewReader(""),
		stdout: &bytes.Buffer{},
		stderr: &bytes.Buffer{},
		stdinIsTerminal: func() (bool, error) {
			return true, nil
		},
		newSender: func(configPath string) (promptSender, error) {
			t.Fatalf("newSender should not be called")
			return nil, nil
		},
	})
	cmd.SetArgs([]string{"send", "--prompt", "hello"})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "target is required") {
		t.Fatalf("error = %v", err)
	}
}

func TestSendCmd_MissingPrompt(t *testing.T) {
	cmd := newRootCmd(commandDeps{
		stdin:  strings.NewReader(""),
		stdout: &bytes.Buffer{},
		stderr: &bytes.Buffer{},
		stdinIsTerminal: func() (bool, error) {
			return true, nil
		},
		newSender: func(configPath string) (promptSender, error) {
			t.Fatalf("newSender should not be called")
			return nil, nil
		},
	})
	cmd.SetArgs([]string{"send", "--target", "ChatGPT"})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "prompt is required") {
		t.Fatalf("error = %v", err)
	}
}

func TestSendCmd_NewSenderError(t *testing.T) {
	cmd := newRootCmd(commandDeps{
		stdin:  strings.NewReader(""),
		stdout: &bytes.Buffer{},
		stderr: &bytes.Buffer{},
		stdinIsTerminal: func() (bool, error) {
			return true, nil
		},
		newSender: func(configPath string) (promptSender, error) {
			return nil, fmt.Errorf("load failure")
		},
	})
	cmd.SetArgs([]string{"send", "--target", "ChatGPT", "--prompt", "hello"})

	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "load failure") {
		t.Fatalf("error = %v", err)
	}
}

func TestReadPromptFromStdin_TerminalError(t *testing.T) {
	_, err := readPromptFromStdin(strings.NewReader(""), func() (bool, error) {
		return false, fmt.Errorf("stat failed")
	})
	if err == nil || !strings.Contains(err.Error(), "stat failed") {
		t.Fatalf("error = %v", err)
	}
}

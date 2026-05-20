package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
)

type routeChoice struct {
	Target string `json:"target"`
}

type openAIResponsesRequest struct {
	Model        string                 `json:"model"`
	Instructions string                 `json:"instructions"`
	Input        string                 `json:"input"`
	Text         map[string]interface{} `json:"text"`
}

type openAIResponsesResponse struct {
	Status            string                 `json:"status"`
	Error             *openAIResponseError   `json:"error"`
	IncompleteDetails map[string]interface{} `json:"incomplete_details"`
	Output            []openAIOutputItem     `json:"output"`
}

type openAIResponseError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code"`
}

type openAIOutputItem struct {
	Type    string                `json:"type"`
	Role    string                `json:"role"`
	Status  string                `json:"status"`
	Content []openAIOutputContent `json:"content"`
}

type openAIOutputContent struct {
	Type    string `json:"type"`
	Text    string `json:"text"`
	Refusal string `json:"refusal"`
}

func ResolveTarget(config *Config, selectedTargetName string, prompt string) (*Target, string, error) {
	target, err := config.FindTargetByName(selectedTargetName)
	if err != nil {
		return nil, "", err
	}

	if target.Type != "auto" {
		return target, target.Name, nil
	}

	chosenTargetName, err := RoutePrompt(config, prompt)
	if err != nil {
		return nil, "", err
	}

	chosenTarget, err := config.FindTargetByName(chosenTargetName)
	if err != nil {
		return nil, "", fmt.Errorf("router selected unknown target %q", chosenTargetName)
	}

	if chosenTarget.Type == "auto" {
		return nil, "", fmt.Errorf("router selected auto target %q", chosenTargetName)
	}

	return chosenTarget, chosenTarget.Name, nil
}

func RoutePrompt(config *Config, prompt string) (string, error) {
	if config == nil || config.Router == nil {
		return "", fmt.Errorf("router config is required")
	}

	apiKey := strings.TrimSpace(os.Getenv(config.Router.APIKeyEnv))
	if apiKey == "" {
		return "", fmt.Errorf("router API key env var %q is not set", config.Router.APIKeyEnv)
	}

	availableTargets := make([]Target, 0, len(config.Targets))
	targetNames := make([]string, 0, len(config.Targets))
	for _, target := range config.Targets {
		if target.Type == "auto" {
			continue
		}

		availableTargets = append(availableTargets, target)
		targetNames = append(targetNames, target.Name)
	}

	if len(availableTargets) == 0 {
		return "", fmt.Errorf("router requires at least one non-auto target")
	}

	instructions := config.Router.SystemPrompt
	if instructions == "" {
		instructions = buildDefaultRouterInstructions(availableTargets)
	}

	requestBody := openAIResponsesRequest{
		Model:        config.Router.Model,
		Instructions: instructions,
		Input:        buildRouterInput(prompt, availableTargets),
		Text: map[string]interface{}{
			"format": map[string]interface{}{
				"type":   "json_schema",
				"name":   "route_choice",
				"strict": true,
				"schema": map[string]interface{}{
					"type":                 "object",
					"additionalProperties": false,
					"properties": map[string]interface{}{
						"target": map[string]interface{}{
							"type": "string",
							"enum": targetNames,
						},
					},
					"required": []string{"target"},
				},
			},
		},
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("marshal router request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, "https://api.openai.com/v1/responses", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create router request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("route prompt with OpenAI: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var errBody map[string]interface{}
		_ = json.NewDecoder(resp.Body).Decode(&errBody)
		return "", fmt.Errorf("router request failed with status %d", resp.StatusCode)
	}

	var response openAIResponsesResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("decode router response: %w", err)
	}

	if response.Error != nil && response.Error.Message != "" {
		return "", fmt.Errorf("router request failed: %s", response.Error.Message)
	}

	if response.Status != "" && response.Status != "completed" {
		return "", fmt.Errorf("router response status %q", response.Status)
	}

	outputText := extractRouterOutputText(response.Output)
	if strings.TrimSpace(outputText) == "" {
		if refusal := extractRouterRefusalText(response.Output); strings.TrimSpace(refusal) != "" {
			return "", fmt.Errorf("router refused request: %s", strings.TrimSpace(refusal))
		}
		return "", fmt.Errorf("router returned empty output")
	}

	var choice routeChoice
	if err := json.Unmarshal([]byte(outputText), &choice); err != nil {
		return "", fmt.Errorf("decode router choice: %w", err)
	}

	choice.Target = strings.TrimSpace(choice.Target)
	if choice.Target == "" {
		return "", fmt.Errorf("router returned empty target")
	}

	if _, err := config.FindTargetByName(choice.Target); err != nil {
		return "", fmt.Errorf("router returned invalid target %q", choice.Target)
	}

	return choice.Target, nil
}

func extractRouterOutputText(items []openAIOutputItem) string {
	for _, item := range items {
		if item.Type != "message" {
			continue
		}

		for _, content := range item.Content {
			if content.Type == "output_text" && strings.TrimSpace(content.Text) != "" {
				return content.Text
			}
		}
	}

	return ""
}

func extractRouterRefusalText(items []openAIOutputItem) string {
	for _, item := range items {
		if item.Type != "message" {
			continue
		}

		for _, content := range item.Content {
			if content.Type == "refusal" && strings.TrimSpace(content.Refusal) != "" {
				return content.Refusal
			}
		}
	}

	return ""
}

func buildDefaultRouterInstructions(targets []Target) string {
	var builder strings.Builder
	builder.WriteString("Choose the single best target for the user's prompt.\n")
	builder.WriteString("You must return valid JSON with exactly one field named target.\n")
	builder.WriteString("The target value must exactly match one of the provided target names.\n")
	builder.WriteString("Do not explain your choice.\n")
	builder.WriteString("Consider each target's description when present.\n")
	builder.WriteString("Available targets:\n")
	for _, target := range targets {
		builder.WriteString("- ")
		builder.WriteString(target.Name)
		if target.Description != "" {
			builder.WriteString(": ")
			builder.WriteString(target.Description)
		}
		builder.WriteString("\n")
	}
	return strings.TrimSpace(builder.String())
}

func buildRouterInput(prompt string, targets []Target) string {
	var builder strings.Builder
	builder.WriteString("User prompt:\n")
	builder.WriteString(prompt)
	builder.WriteString("\n\nTargets:\n")
	for _, target := range targets {
		builder.WriteString("- ")
		builder.WriteString(target.Name)
		if target.Description != "" {
			builder.WriteString(": ")
			builder.WriteString(target.Description)
		}
		builder.WriteString("\n")
	}
	return strings.TrimSpace(builder.String())
}

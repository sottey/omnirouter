package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	omnirouterconfig "omnirouter/internal/config"
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

func ResolveTarget(config *omnirouterconfig.Config, selectedTargetName string, prompt string) (*omnirouterconfig.Target, string, error) {
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

func RoutePrompt(config *omnirouterconfig.Config, prompt string) (string, error) {
	if config == nil || config.Router == nil {
		return "", fmt.Errorf("router config is required")
	}

	apiKey := strings.TrimSpace(os.Getenv(config.Router.APIKeyEnv))
	if apiKey == "" {
		return "", fmt.Errorf("router API key env var %q is not set", config.Router.APIKeyEnv)
	}

	availableTargets := make([]omnirouterconfig.Target, 0, len(config.Targets))
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

	var chosenTarget string
	var err error

	switch config.Router.Provider {
	case "openai":
		chosenTarget, err = routePromptOpenAI(config, instructions, prompt, targetNames, availableTargets, apiKey)
	case "gemini":
		chosenTarget, err = routePromptGemini(config, instructions, prompt, targetNames, availableTargets, apiKey)
	case "anthropic":
		chosenTarget, err = routePromptAnthropic(config, instructions, prompt, targetNames, availableTargets, apiKey)
	default:
		return "", fmt.Errorf("unsupported router provider %q", config.Router.Provider)
	}

	if err != nil {
		return "", err
	}

	if _, err := config.FindTargetByName(chosenTarget); err != nil {
		return "", fmt.Errorf("router returned invalid target %q", chosenTarget)
	}

	return chosenTarget, nil
}

func routePromptOpenAI(config *omnirouterconfig.Config, instructions string, prompt string, targetNames []string, targets []omnirouterconfig.Target, apiKey string) (string, error) {
	requestBody := openAIResponsesRequest{
		Model:        config.Router.Model,
		Instructions: instructions,
		Input:        buildRouterInput(prompt, targets),
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

	return choice.Target, nil
}

func routePromptGemini(config *omnirouterconfig.Config, instructions string, prompt string, targetNames []string, targets []omnirouterconfig.Target, apiKey string) (string, error) {
	type part struct {
		Text string `json:"text"`
	}
	type content struct {
		Role  string `json:"role,omitempty"`
		Parts []part `json:"parts"`
	}
	type schemaProp struct {
		Type string   `json:"type"`
		Enum []string `json:"enum,omitempty"`
	}
	type schemaObj struct {
		Type       string                `json:"type"`
		Properties map[string]schemaProp `json:"properties"`
		Required   []string              `json:"required"`
	}
	type generationConfig struct {
		ResponseMimeType string     `json:"responseMimeType"`
		ResponseSchema   *schemaObj `json:"responseSchema,omitempty"`
	}
	type geminiRequest struct {
		Contents          []content         `json:"contents"`
		SystemInstruction *content          `json:"systemInstruction,omitempty"`
		GenerationConfig  *generationConfig `json:"generationConfig,omitempty"`
	}

	input := buildRouterInput(prompt, targets)

	reqBody := geminiRequest{
		Contents: []content{
			{
				Parts: []part{{Text: input}},
			},
		},
		SystemInstruction: &content{
			Parts: []part{{Text: instructions}},
		},
		GenerationConfig: &generationConfig{
			ResponseMimeType: "application/json",
			ResponseSchema: &schemaObj{
				Type: "OBJECT",
				Properties: map[string]schemaProp{
					"target": {
						Type: "STRING",
						Enum: targetNames,
					},
				},
				Required: []string{"target"},
			},
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal gemini request: %w", err)
	}

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", config.Router.Model, apiKey)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create gemini request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("route prompt with Gemini: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var errBody map[string]interface{}
		_ = json.NewDecoder(resp.Body).Decode(&errBody)
		return "", fmt.Errorf("gemini router request failed with status %d", resp.StatusCode)
	}

	type geminiPart struct {
		Text string `json:"text"`
	}
	type geminiContent struct {
		Parts []geminiPart `json:"parts"`
	}
	type geminiCandidate struct {
		Content      geminiContent `json:"content"`
		FinishReason string        `json:"finishReason"`
	}
	type geminiResponse struct {
		Candidates []geminiCandidate `json:"candidates"`
	}

	var response geminiResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("decode gemini response: %w", err)
	}

	if len(response.Candidates) == 0 || len(response.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("gemini returned empty response")
	}

	responseText := response.Candidates[0].Content.Parts[0].Text
	if strings.TrimSpace(responseText) == "" {
		return "", fmt.Errorf("gemini returned empty output text")
	}

	var choice routeChoice
	if err := json.Unmarshal([]byte(responseText), &choice); err != nil {
		return "", fmt.Errorf("decode gemini router choice: %w (raw response: %q)", err, responseText)
	}

	choice.Target = strings.TrimSpace(choice.Target)
	if choice.Target == "" {
		return "", fmt.Errorf("gemini router returned empty target")
	}

	return choice.Target, nil
}

func routePromptAnthropic(config *omnirouterconfig.Config, instructions string, prompt string, targetNames []string, targets []omnirouterconfig.Target, apiKey string) (string, error) {
	type anthropicMessage struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	type anthropicSchemaProp struct {
		Type string   `json:"type"`
		Enum []string `json:"enum,omitempty"`
	}
	type anthropicSchemaObj struct {
		Type       string                         `json:"type"`
		Properties map[string]anthropicSchemaProp `json:"properties"`
		Required   []string                       `json:"required"`
	}
	type anthropicTool struct {
		Name        string             `json:"name"`
		Description string             `json:"description"`
		InputSchema anthropicSchemaObj `json:"input_schema"`
	}
	type anthropicToolChoice struct {
		Type string `json:"type"`
		Name string `json:"name"`
	}
	type anthropicRequest struct {
		Model      string               `json:"model"`
		MaxTokens  int                  `json:"max_tokens"`
		System     string               `json:"system,omitempty"`
		Messages   []anthropicMessage   `json:"messages"`
		Tools      []anthropicTool      `json:"tools"`
		ToolChoice *anthropicToolChoice `json:"tool_choice,omitempty"`
	}

	input := buildRouterInput(prompt, targets)

	reqBody := anthropicRequest{
		Model:     config.Router.Model,
		MaxTokens: 1024,
		System:    instructions,
		Messages: []anthropicMessage{
			{Role: "user", Content: input},
		},
		Tools: []anthropicTool{
			{
				Name:        "route_choice",
				Description: "Select the best model target for the user prompt.",
				InputSchema: anthropicSchemaObj{
					Type: "object",
					Properties: map[string]anthropicSchemaProp{
						"target": {
							Type: "string",
							Enum: targetNames,
						},
					},
					Required: []string{"target"},
				},
			},
		},
		ToolChoice: &anthropicToolChoice{
			Type: "tool",
			Name: "route_choice",
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal anthropic request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, "https://api.anthropic.com/v1/messages", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create anthropic request: %w", err)
	}
	req.Header.Set("X-API-Key", apiKey)
	req.Header.Set("Anthropic-Version", "2023-06-01")
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("route prompt with Anthropic: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var errBody map[string]interface{}
		_ = json.NewDecoder(resp.Body).Decode(&errBody)
		return "", fmt.Errorf("anthropic router request failed with status %d", resp.StatusCode)
	}

	type anthropicResponseContent struct {
		Type  string                 `json:"type"`
		Name  string                 `json:"name"`
		Input map[string]interface{} `json:"input"`
	}
	type anthropicResponse struct {
		Content []anthropicResponseContent `json:"content"`
	}

	var response anthropicResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("decode anthropic response: %w", err)
	}

	var targetVal string
	for _, contentItem := range response.Content {
		if contentItem.Type == "tool_use" && contentItem.Name == "route_choice" && contentItem.Input != nil {
			if target, ok := contentItem.Input["target"].(string); ok {
				targetVal = target
				break
			}
		}
	}

	if targetVal == "" {
		return "", fmt.Errorf("anthropic response did not contain target tool call")
	}

	return targetVal, nil
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

func buildDefaultRouterInstructions(targets []omnirouterconfig.Target) string {
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

func buildRouterInput(prompt string, targets []omnirouterconfig.Target) string {
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

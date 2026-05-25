package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
)

func CallLLMAPI(provider, model, apiKeyEnv, systemPrompt, prompt string) (string, error) {
	apiKey := strings.TrimSpace(os.Getenv(apiKeyEnv))
	if apiKey == "" {
		return "", fmt.Errorf("API key env var %q is not set", apiKeyEnv)
	}

	switch provider {
	case "openai":
		return callOpenAI(model, apiKey, systemPrompt, prompt)
	case "gemini":
		return callGemini(model, apiKey, systemPrompt, prompt)
	case "anthropic":
		return callAnthropic(model, apiKey, systemPrompt, prompt)
	default:
		return "", fmt.Errorf("unsupported API provider %q", provider)
	}
}

func callOpenAI(model, apiKey, systemPrompt, prompt string) (string, error) {
	type message struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	type request struct {
		Model    string    `json:"model"`
		Messages []message `json:"messages"`
	}

	messages := make([]message, 0, 2)
	if systemPrompt != "" {
		messages = append(messages, message{Role: "system", Content: systemPrompt})
	}
	messages = append(messages, message{Role: "user", Content: prompt})

	reqBody := request{
		Model:    model,
		Messages: messages,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal openai request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, "https://api.openai.com/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create openai request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("execute openai request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var errBody map[string]interface{}
		_ = json.NewDecoder(resp.Body).Decode(&errBody)
		return "", fmt.Errorf("openai API request failed with status %d", resp.StatusCode)
	}

	type choice struct {
		Message message `json:"message"`
	}
	type response struct {
		Choices []choice `json:"choices"`
	}

	var respBody response
	if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
		return "", fmt.Errorf("decode openai response: %w", err)
	}

	if len(respBody.Choices) == 0 {
		return "", fmt.Errorf("openai returned no choices")
	}

	return respBody.Choices[0].Message.Content, nil
}

func callGemini(model, apiKey, systemPrompt, prompt string) (string, error) {
	type part struct {
		Text string `json:"text"`
	}
	type content struct {
		Parts []part `json:"parts"`
	}
	type request struct {
		Contents          []content `json:"contents"`
		SystemInstruction *content  `json:"systemInstruction,omitempty"`
	}

	reqBody := request{
		Contents: []content{
			{Parts: []part{{Text: prompt}}},
		},
	}
	if systemPrompt != "" {
		reqBody.SystemInstruction = &content{
			Parts: []part{{Text: systemPrompt}},
		}
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal gemini request: %w", err)
	}

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", model, apiKey)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create gemini request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("execute gemini request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var errBody map[string]interface{}
		_ = json.NewDecoder(resp.Body).Decode(&errBody)
		return "", fmt.Errorf("gemini API request failed with status %d", resp.StatusCode)
	}

	type geminiPart struct {
		Text string `json:"text"`
	}
	type geminiContent struct {
		Parts []geminiPart `json:"parts"`
	}
	type geminiCandidate struct {
		Content geminiContent `json:"content"`
	}
	type response struct {
		Candidates []geminiCandidate `json:"candidates"`
	}

	var respBody response
	if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
		return "", fmt.Errorf("decode gemini response: %w", err)
	}

	if len(respBody.Candidates) == 0 || len(respBody.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("gemini returned no content")
	}

	return respBody.Candidates[0].Content.Parts[0].Text, nil
}

func callAnthropic(model, apiKey, systemPrompt, prompt string) (string, error) {
	type message struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	type request struct {
		Model     string    `json:"model"`
		MaxTokens int       `json:"max_tokens"`
		System    string    `json:"system,omitempty"`
		Messages  []message `json:"messages"`
	}

	reqBody := request{
		Model:     model,
		MaxTokens: 4096,
		System:    systemPrompt,
		Messages: []message{
			{Role: "user", Content: prompt},
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
		return "", fmt.Errorf("execute anthropic request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var errBody map[string]interface{}
		_ = json.NewDecoder(resp.Body).Decode(&errBody)
		return "", fmt.Errorf("anthropic API request failed with status %d", resp.StatusCode)
	}

	type anthropicPart struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	type response struct {
		Content []anthropicPart `json:"content"`
	}

	var respBody response
	if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
		return "", fmt.Errorf("decode anthropic response: %w", err)
	}

	var responseText string
	for _, part := range respBody.Content {
		if part.Type == "text" {
			responseText += part.Text
		}
	}

	if responseText == "" {
		return "", fmt.Errorf("anthropic returned no text content")
	}

	return responseText, nil
}

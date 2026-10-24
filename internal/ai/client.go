package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"logpulse/internal/db"
)

type AIRequest struct {
	Provider  string `json:"provider"`
	Model     string `json:"model"`
	Prompt    string `json:"prompt"`
	UserNotes string `json:"userNotes"`
	Telemetry string `json:"telemetry"`
}

type AIResponse struct {
	Text         string  `json:"text"`
	InputTokens  int     `json:"inputTokens"`
	OutputTokens int     `json:"outputTokens"`
	CostUSD      float64 `json:"costUsd"`
}

type MultiProviderClient struct {
	store          *db.Store
	httpClient     *http.Client
	ollamaEndpoint string
}

func NewMultiProviderClient(store *db.Store) *MultiProviderClient {
	return &MultiProviderClient{
		store: store,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		ollamaEndpoint: "http://localhost:11434",
	}
}

func (c *MultiProviderClient) SetOllamaEndpoint(endpoint string) {
	c.ollamaEndpoint = strings.TrimRight(endpoint, "/")
}

func (c *MultiProviderClient) Query(ctx context.Context, req AIRequest) (*AIResponse, error) {
	// Budget enforcement
	if c.store != nil {
		budgetStr, _ := c.store.GetSetting("monthly_budget_usd")
		if budgetStr != "" {
			if maxBudget, err := strconv.ParseFloat(budgetStr, 64); err == nil && maxBudget > 0 {
				spend, err := c.store.GetMonthlyAISpendUSD()
				if err == nil && spend >= maxBudget {
					return nil, fmt.Errorf("monthly budget limit exceeded: current spend $%.2f exceeds limit $%.2f", spend, maxBudget)
				}
			}
		}
	}

	fullPrompt := req.Prompt
	if req.UserNotes != "" {
		fullPrompt = fmt.Sprintf("USER CONTEXT:\n%s\n\nLOG TO INVESTIGATE:\n%s", req.UserNotes, fullPrompt)
	}
	if req.Telemetry != "" {
		fullPrompt = fmt.Sprintf("%s\n\nSYSTEM TELEMETRY:\n%s", fullPrompt, req.Telemetry)
	}

	inTokens := EstimateTokens(fullPrompt)
	providerLower := strings.ToLower(req.Provider)

	var responseText string
	var err error

	switch providerLower {
	case "ollama":
		responseText, err = c.queryOllama(ctx, req.Model, fullPrompt)
	case "openai":
		responseText, err = c.queryOpenAI(ctx, req.Model, fullPrompt)
	case "anthropic":
		responseText, err = c.queryAnthropic(ctx, req.Model, fullPrompt)
	case "gemini":
		responseText, err = c.queryGemini(ctx, req.Model, fullPrompt)
	default:
		// Default fallback to Ollama
		responseText, err = c.queryOllama(ctx, req.Model, fullPrompt)
	}

	if err != nil {
		return nil, err
	}

	outTokens := EstimateTokens(responseText)
	cost := CalculateCostUSD(req.Provider, req.Model, inTokens, outTokens)

	if c.store != nil {
		summary := req.Prompt
		if len(summary) > 60 {
			summary = summary[:60] + "..."
		}
		_ = c.store.RecordAIAccounting(db.AIAccountingEntry{
			Timestamp:    time.Now(),
			Provider:     req.Provider,
			Model:        req.Model,
			InputTokens:  inTokens,
			OutputTokens: outTokens,
			CostUSD:      cost,
			QuerySummary: summary,
		})
	}

	return &AIResponse{
		Text:         responseText,
		InputTokens:  inTokens,
		OutputTokens: outTokens,
		CostUSD:      cost,
	}, nil
}

func (c *MultiProviderClient) queryOllama(ctx context.Context, model, prompt string) (string, error) {
	if model == "" {
		model = "llama3"
	}
	url := fmt.Sprintf("%s/api/generate", c.ollamaEndpoint)
	payload := map[string]interface{}{
		"model":  model,
		"prompt": prompt,
		"stream": false,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	res, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("ollama connection error: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		respBytes, _ := io.ReadAll(res.Body)
		return "", fmt.Errorf("ollama error (status %d): %s", res.StatusCode, string(respBytes))
	}

	var parsed struct {
		Response string `json:"response"`
	}
	if err := json.NewDecoder(res.Body).Decode(&parsed); err != nil {
		return "", err
	}

	return parsed.Response, nil
}

func (c *MultiProviderClient) queryOpenAI(ctx context.Context, model, prompt string) (string, error) {
	var apiKey string
	if c.store != nil {
		apiKey, _ = c.store.GetSetting("api_key_openai")
	}
	if apiKey == "" {
		return "", fmt.Errorf("OpenAI API key not configured")
	}
	if model == "" {
		model = "gpt-4o"
	}

	payload := map[string]interface{}{
		"model": model,
		"messages": []map[string]string{
			{"role": "system", "content": "You are LogPulse AI, an expert system engineer diagnosing log files and stack traces."},
			{"role": "user", "content": prompt},
		},
	}

	body, _ := json.Marshal(payload)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	res, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		respBytes, _ := io.ReadAll(res.Body)
		return "", fmt.Errorf("OpenAI API error (%d): %s", res.StatusCode, string(respBytes))
	}

	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(res.Body).Decode(&parsed); err != nil {
		return "", err
	}
	if len(parsed.Choices) > 0 {
		return parsed.Choices[0].Message.Content, nil
	}
	return "", fmt.Errorf("empty OpenAI response")
}

func (c *MultiProviderClient) queryAnthropic(ctx context.Context, model, prompt string) (string, error) {
	var apiKey string
	if c.store != nil {
		apiKey, _ = c.store.GetSetting("api_key_anthropic")
	}
	if apiKey == "" {
		return "", fmt.Errorf("Anthropic API key not configured")
	}
	if model == "" {
		model = "claude-3-5-sonnet-20241022"
	}

	payload := map[string]interface{}{
		"model":      model,
		"max_tokens": 2048,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	}

	body, _ := json.Marshal(payload)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("x-api-key", apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")
	httpReq.Header.Set("Content-Type", "application/json")

	res, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		respBytes, _ := io.ReadAll(res.Body)
		return "", fmt.Errorf("Anthropic API error (%d): %s", res.StatusCode, string(respBytes))
	}

	var parsed struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.NewDecoder(res.Body).Decode(&parsed); err != nil {
		return "", err
	}
	if len(parsed.Content) > 0 {
		return parsed.Content[0].Text, nil
	}
	return "", fmt.Errorf("empty Anthropic response")
}

func (c *MultiProviderClient) queryGemini(ctx context.Context, model, prompt string) (string, error) {
	var apiKey string
	if c.store != nil {
		apiKey, _ = c.store.GetSetting("api_key_gemini")
	}
	if apiKey == "" {
		return "", fmt.Errorf("Gemini API key not configured")
	}
	if model == "" {
		model = "gemini-1.5-flash"
	}

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", model, apiKey)
	payload := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]string{
					{"text": prompt},
				},
			},
		},
	}

	body, _ := json.Marshal(payload)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	res, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		respBytes, _ := io.ReadAll(res.Body)
		return "", fmt.Errorf("Gemini API error (%d): %s", res.StatusCode, string(respBytes))
	}

	var parsed struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := json.NewDecoder(res.Body).Decode(&parsed); err != nil {
		return "", err
	}
	if len(parsed.Candidates) > 0 && len(parsed.Candidates[0].Content.Parts) > 0 {
		return parsed.Candidates[0].Content.Parts[0].Text, nil
	}
	return "", fmt.Errorf("empty Gemini response")
}

package ai_test

import (
	"testing"

	"logpulse/internal/ai"
)

func TestEstimateTokens(t *testing.T) {
	text := "This is a standard log error line with 10 words."
	tokens := ai.EstimateTokens(text)
	if tokens < 8 || tokens > 20 {
		t.Errorf("unexpected token count: %d", tokens)
	}
}

func TestCalculateCostUSD(t *testing.T) {
	costOpenAI := ai.CalculateCostUSD("openai", "gpt-4o", 1000, 500)
	if costOpenAI <= 0 {
		t.Errorf("expected positive cost, got %f", costOpenAI)
	}

	costClaude := ai.CalculateCostUSD("anthropic", "claude-3-5-sonnet", 1000, 500)
	if costClaude <= 0 {
		t.Errorf("expected positive cost, got %f", costClaude)
	}

	costOllama := ai.CalculateCostUSD("ollama", "llama3", 1000, 500)
	if costOllama != 0 {
		t.Errorf("ollama cost should be 0, got %f", costOllama)
	}
}

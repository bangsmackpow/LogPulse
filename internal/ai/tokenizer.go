package ai

import (
	"strings"
)

func EstimateTokens(text string) int {
	if len(text) == 0 {
		return 0
	}
	words := len(strings.Fields(text))
	charBased := (len(text) + 3) / 4
	wordBased := int(float64(words) * 1.3)

	if charBased > wordBased {
		return charBased
	}
	return wordBased
}

func CalculateCostUSD(provider, model string, inTokens, outTokens int) float64 {
	providerLower := strings.ToLower(provider)
	modelLower := strings.ToLower(model)

	if providerLower == "ollama" || providerLower == "local" {
		return 0.0
	}

	var inPricePerMillion, outPricePerMillion float64

	switch {
	case strings.Contains(modelLower, "gpt-4o-mini"):
		inPricePerMillion = 0.15
		outPricePerMillion = 0.60
	case strings.Contains(modelLower, "gpt-4o"):
		inPricePerMillion = 2.50
		outPricePerMillion = 10.00
	case strings.Contains(modelLower, "claude-3-5-sonnet") || strings.Contains(modelLower, "sonnet"):
		inPricePerMillion = 3.00
		outPricePerMillion = 15.00
	case strings.Contains(modelLower, "claude-3-haiku") || strings.Contains(modelLower, "haiku"):
		inPricePerMillion = 0.25
		outPricePerMillion = 1.25
	case strings.Contains(modelLower, "gemini-1.5-flash") || strings.Contains(modelLower, "gemini-2.0-flash"):
		inPricePerMillion = 0.075
		outPricePerMillion = 0.30
	case strings.Contains(modelLower, "gemini-1.5-pro"):
		inPricePerMillion = 1.25
		outPricePerMillion = 5.00
	case strings.Contains(modelLower, "deepseek-reasoner") || strings.Contains(modelLower, "deepseek-r1"):
		inPricePerMillion = 0.55
		outPricePerMillion = 2.19
	case strings.Contains(modelLower, "deepseek-chat") || strings.Contains(modelLower, "deepseek-v3"):
		inPricePerMillion = 0.14
		outPricePerMillion = 0.28
	case strings.Contains(modelLower, "llama-3.3") || strings.Contains(modelLower, "llama-3-"):
		inPricePerMillion = 0.59
		outPricePerMillion = 0.79
	case strings.Contains(modelLower, "mixtral"):
		inPricePerMillion = 0.24
		outPricePerMillion = 0.24
	default:
		// Default estimation for unknown modern LLM
		inPricePerMillion = 1.00
		outPricePerMillion = 4.00
	}

	inCost := (float64(inTokens) / 1_000_000.0) * inPricePerMillion
	outCost := (float64(outTokens) / 1_000_000.0) * outPricePerMillion
	return inCost + outCost
}

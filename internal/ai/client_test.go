package ai_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"logpulse/internal/ai"
	"logpulse/internal/db"
)

func TestMultiProviderClient_BudgetCeiling(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "ai_budget_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "test.db")
	store, err := db.NewStore(dbPath)
	if err != nil {
		t.Fatalf("failed to init store: %v", err)
	}
	defer store.Close()

	// Set monthly budget to 1.00 USD
	store.SetSetting("monthly_budget_usd", "1.00")

	// Pre-fill spend with 1.50 USD
	store.RecordAIAccounting(db.AIAccountingEntry{
		CostUSD: 1.50,
	})

	client := ai.NewMultiProviderClient(store)

	req := ai.AIRequest{
		Provider: "openai",
		Model:    "gpt-4o",
		Prompt:   "Analyze log",
	}

	_, err = client.Query(context.Background(), req)
	if err == nil {
		t.Fatal("expected error due to monthly budget limit exceeded")
	}
}

func TestMultiProviderClient_OllamaMockDispatch(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"response":"This log indicates database connection timeout"}`))
	}))
	defer ts.Close()

	tempDir, _ := os.MkdirTemp("", "ai_ollama_test_*")
	defer os.RemoveAll(tempDir)
	store, _ := db.NewStore(filepath.Join(tempDir, "test.db"))
	defer store.Close()

	client := ai.NewMultiProviderClient(store)
	client.SetOllamaEndpoint(ts.URL)

	req := ai.AIRequest{
		Provider: "ollama",
		Model:    "llama3",
		Prompt:   "Explain error: timeout",
	}

	resp, err := client.Query(context.Background(), req)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if resp.Text != "This log indicates database connection timeout" {
		t.Errorf("unexpected response: %q", resp.Text)
	}
}

func TestMultiProviderClient_OpenRouterDispatch(t *testing.T) {
	var capturedReferer, capturedTitle, capturedAuth string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedReferer = r.Header.Get("HTTP-Referer")
		capturedTitle = r.Header.Get("X-Title")
		capturedAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"choices":[{"message":{"content":"OpenRouter diagnosis: memory leak"}}]}`))
	}))
	defer ts.Close()

	tempDir, _ := os.MkdirTemp("", "ai_openrouter_test_*")
	defer os.RemoveAll(tempDir)
	store, _ := db.NewStore(filepath.Join(tempDir, "test.db"))
	defer store.Close()

	store.SetSetting("api_key_openrouter", "sk-or-test-key")

	client := ai.NewMultiProviderClient(store)
	client.SetOpenRouterEndpoint(ts.URL)

	req := ai.AIRequest{
		Provider: "openrouter",
		Model:    "anthropic/claude-3.5-sonnet",
		Prompt:   "Explain memory leak",
	}

	resp, err := client.Query(context.Background(), req)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if resp.Text != "OpenRouter diagnosis: memory leak" {
		t.Errorf("unexpected response: %q", resp.Text)
	}
	if capturedReferer != "https://github.com/bangsmackpow/LogPulse" {
		t.Errorf("expected HTTP-Referer header, got: %q", capturedReferer)
	}
	if capturedTitle != "LogPulse" {
		t.Errorf("expected X-Title header, got: %q", capturedTitle)
	}
	if capturedAuth != "Bearer sk-or-test-key" {
		t.Errorf("expected Authorization header, got: %q", capturedAuth)
	}
}

func TestMultiProviderClient_DeepSeekDispatch(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"choices":[{"message":{"content":"DeepSeek R1 reasoning: nil pointer dereference"}}]}`))
	}))
	defer ts.Close()

	tempDir, _ := os.MkdirTemp("", "ai_deepseek_test_*")
	defer os.RemoveAll(tempDir)
	store, _ := db.NewStore(filepath.Join(tempDir, "test.db"))
	defer store.Close()

	store.SetSetting("api_key_deepseek", "sk-deepseek-test")

	client := ai.NewMultiProviderClient(store)
	client.SetDeepSeekEndpoint(ts.URL)

	req := ai.AIRequest{
		Provider: "deepseek",
		Model:    "deepseek-reasoner",
		Prompt:   "Trace nil pointer",
	}

	resp, err := client.Query(context.Background(), req)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if resp.Text != "DeepSeek R1 reasoning: nil pointer dereference" {
		t.Errorf("unexpected response: %q", resp.Text)
	}
}

func TestMultiProviderClient_GroqDispatch(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"choices":[{"message":{"content":"Groq fast response: syntax error"}}]}`))
	}))
	defer ts.Close()

	tempDir, _ := os.MkdirTemp("", "ai_groq_test_*")
	defer os.RemoveAll(tempDir)
	store, _ := db.NewStore(filepath.Join(tempDir, "test.db"))
	defer store.Close()

	store.SetSetting("api_key_groq", "gsk-test")

	client := ai.NewMultiProviderClient(store)
	client.SetGroqEndpoint(ts.URL)

	req := ai.AIRequest{
		Provider: "groq",
		Model:    "llama-3.3-70b-versatile",
		Prompt:   "Trace syntax error",
	}

	resp, err := client.Query(context.Background(), req)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if resp.Text != "Groq fast response: syntax error" {
		t.Errorf("unexpected response: %q", resp.Text)
	}
}

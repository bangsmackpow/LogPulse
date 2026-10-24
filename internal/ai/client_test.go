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

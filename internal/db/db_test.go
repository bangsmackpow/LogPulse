package db_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"logpulse/internal/db"
)

func TestStore_DiscoveredLogs(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "logpulse_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "test.db")
	store, err := db.NewStore(dbPath)
	if err != nil {
		t.Fatalf("failed to initialize store: %v", err)
	}
	defer store.Close()

	entry := db.DiscoveredLog{
		FilePath:     "C:\\test\\app.log",
		AppName:      "TestApp",
		FileSize:     1024,
		LastModified: time.Now().Truncate(time.Second),
		Format:       "text",
		IsFavorite:   false,
		Tag:          "diagnostic",
	}

	if err := store.SaveDiscoveredLog(entry); err != nil {
		t.Fatalf("failed to save discovered log: %v", err)
	}

	logs, err := store.ListDiscoveredLogs()
	if err != nil {
		t.Fatalf("failed to list discovered logs: %v", err)
	}

	if len(logs) != 1 {
		t.Fatalf("expected 1 log, got %d", len(logs))
	}

	if logs[0].AppName != "TestApp" || logs[0].FilePath != entry.FilePath {
		t.Errorf("unexpected log record: %+v", logs[0])
	}
}

func TestStore_AIAccountingAndBudget(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "logpulse_ai_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "test_ai.db")
	store, err := db.NewStore(dbPath)
	if err != nil {
		t.Fatalf("failed to initialize store: %v", err)
	}
	defer store.Close()

	err = store.RecordAIAccounting(db.AIAccountingEntry{
		Timestamp:    time.Now(),
		Provider:     "openai",
		Model:        "gpt-4o",
		InputTokens:  1000,
		OutputTokens: 200,
		CostUSD:      0.0075,
		QuerySummary: "Crash investigation",
	})
	if err != nil {
		t.Fatalf("failed to record AI accounting: %v", err)
	}

	spend, err := store.GetMonthlyAISpendUSD()
	if err != nil {
		t.Fatalf("failed to get monthly AI spend: %v", err)
	}

	if spend < 0.0074 || spend > 0.0076 {
		t.Errorf("expected spend around 0.0075, got %f", spend)
	}
}

func TestStore_Settings(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "logpulse_settings_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "test_settings.db")
	store, err := db.NewStore(dbPath)
	if err != nil {
		t.Fatalf("failed to initialize store: %v", err)
	}
	defer store.Close()

	if err := store.SetSetting("theme", "dark"); err != nil {
		t.Fatalf("failed to set setting: %v", err)
	}

	val, err := store.GetSetting("theme")
	if err != nil {
		t.Fatalf("failed to get setting: %v", err)
	}
	if val != "dark" {
		t.Errorf("expected 'dark', got '%s'", val)
	}
}

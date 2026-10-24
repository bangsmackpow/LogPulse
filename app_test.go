package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"logpulse/internal/db"
)

func TestApp_LifecycleAndSettings(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "app_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "app_test.db")
	app, err := NewTestApp(dbPath)
	if err != nil {
		t.Fatalf("NewTestApp failed: %v", err)
	}
	defer app.Shutdown(context.Background())

	settings := map[string]string{
		"theme":              "dark",
		"monthly_budget_usd": "10.00",
		"provider":           "ollama",
	}

	if err := app.SaveSettings(settings); err != nil {
		t.Fatalf("SaveSettings failed: %v", err)
	}

	loaded, err := app.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings failed: %v", err)
	}

	if loaded["theme"] != "dark" || loaded["monthly_budget_usd"] != "10.00" {
		t.Errorf("unexpected settings: %+v", loaded)
	}
}

func TestApp_InvestigateWithAI(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "app_ai_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "app_ai_test.db")
	app, err := NewTestApp(dbPath)
	if err != nil {
		t.Fatalf("NewTestApp failed: %v", err)
	}
	defer app.Shutdown(context.Background())

	// Save pre-populated log in DB
	store := app.GetStore()
	store.SaveDiscoveredLog(db.DiscoveredLog{
		FilePath: "C:\\app.log",
		AppName:  "TestApp",
		Format:   "log",
	})

	logs, err := app.GetDiscoveredLogs()
	if err != nil {
		t.Fatalf("GetDiscoveredLogs failed: %v", err)
	}
	if len(logs) != 1 {
		t.Errorf("expected 1 log, got %d", len(logs))
	}

	// Test Scrubbing through IPC method
	scrubbed, matches := app.ScrubText("Error at 192.168.1.1 with user user@test.com")
	if len(matches) != 2 {
		t.Errorf("expected 2 matches, got %d", len(matches))
	}
	if scrubbed != "Error at [REDACTED_IP] with user [REDACTED_EMAIL]" {
		t.Errorf("unexpected scrubbed output: %s", scrubbed)
	}
}

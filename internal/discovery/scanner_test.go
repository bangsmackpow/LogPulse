package discovery_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"logpulse/internal/db"
	"logpulse/internal/discovery"
)

func TestInferAppName(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"C:\\Users\\alice\\AppData\\Local\\Slack\\app.log", "Slack"},
		{"C:\\ProgramData\\Docker\\service.log", "Docker"},
		{"/var/log/nginx/access.log", "nginx"},
		{"C:\\Games\\Cyberpunk\\crash.trace", "Cyberpunk"},
		{"C:\\Windows\\Logs\\CBS\\CBS.log", "CBS"},
	}

	for _, tt := range tests {
		got := discovery.InferAppName(tt.path)
		if got != tt.expected {
			t.Errorf("InferAppName(%q) = %q; want %q", tt.path, got, tt.expected)
		}
	}
}

func TestScanner_ScanDirectory(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "scanner_test_*")
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

	// Create test folder tree
	appDir := filepath.Join(tempDir, "MyApp")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		t.Fatalf("failed to create app dir: %v", err)
	}
	gitDir := filepath.Join(tempDir, "IgnoredApp", ".git")
	if err := os.MkdirAll(gitDir, 0755); err != nil {
		t.Fatalf("failed to create git dir: %v", err)
	}

	logFile := filepath.Join(appDir, "app.log")
	if err := os.WriteFile(logFile, []byte("log entry"), 0644); err != nil {
		t.Fatalf("failed to write log: %v", err)
	}
	ignoredLog := filepath.Join(gitDir, "ignored.log")
	if err := os.WriteFile(ignoredLog, []byte("ignored"), 0644); err != nil {
		t.Fatalf("failed to write ignored log: %v", err)
	}

	scanner := discovery.NewScanner(store)
	discovered, err := scanner.ScanDirectory(tempDir)
	if err != nil {
		t.Fatalf("ScanDirectory failed: %v", err)
	}

	if len(discovered) != 1 {
		t.Fatalf("expected 1 discovered log, got %d", len(discovered))
	}

	if discovered[0].AppName != "MyApp" {
		t.Errorf("expected AppName 'MyApp', got %q", discovered[0].AppName)
	}
}

func TestScanner_Tier2Background(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "scanner_tier2_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "test_tier2.db")
	store, err := db.NewStore(dbPath)
	if err != nil {
		t.Fatalf("failed to init store: %v", err)
	}
	defer store.Close()

	logDir := filepath.Join(tempDir, "TestService")
	os.MkdirAll(logDir, 0755)
	logFile := filepath.Join(logDir, "service.out")
	os.WriteFile(logFile, []byte("output line"), 0644)

	scanner := discovery.NewScanner(store)
	scanner.SetCustomRoots([]string{tempDir})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	foundChan := make(chan db.DiscoveredLog, 1)
	scanner.StartTier2Background(ctx, func(l db.DiscoveredLog) {
		foundChan <- l
	})

	select {
	case found := <-foundChan:
		if found.AppName != "TestService" {
			t.Errorf("expected 'TestService', got %q", found.AppName)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for background scanner to find file")
	}
}

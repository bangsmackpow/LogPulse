package tailer_test

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"logpulse/internal/tailer"
)

func TestParseLogLevel(t *testing.T) {
	tests := []struct {
		line     string
		expected string
	}{
		{"2026-10-24 10:00:00 [ERROR] Connection lost", "ERROR"},
		{"2026-10-24 10:00:01 [FATAL] Null pointer dereference", "ERROR"},
		{"[CRITICAL] Disk full", "ERROR"},
		{"2026-10-24 10:00:02 [WARN] High memory usage", "WARN"},
		{"WARNING: Deprecated API call", "WARN"},
		{"2026-10-24 10:00:03 [INFO] Service started", "INFO"},
		{"info: listening on port 8080", "INFO"},
		{"2026-10-24 10:00:04 [DEBUG] Cache hit for key=123", "DEBUG"},
		{"plain unformatted log text", "INFO"},
	}

	for _, tt := range tests {
		got := tailer.ParseLogLevel(tt.line)
		if got != tt.expected {
			t.Errorf("ParseLogLevel(%q) = %q; want %q", tt.line, got, tt.expected)
		}
	}
}

func TestTailSession_InitialTailAndStreaming(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "tailer_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	testFile := filepath.Join(tempDir, "app.log")
	initialContent := "Line 1\nLine 2\nLine 3\nLine 4\nLine 5\n"
	if err := os.WriteFile(testFile, []byte(initialContent), 0644); err != nil {
		t.Fatalf("failed to write initial file: %v", err)
	}

	var mu sync.Mutex
	var received []tailer.LogLine

	session, err := tailer.StartTail(testFile, 3, func(batch []tailer.LogLine) {
		mu.Lock()
		received = append(received, batch...)
		mu.Unlock()
	})
	if err != nil {
		t.Fatalf("StartTail failed: %v", err)
	}
	defer session.Stop()

	// Initial tail should have loaded the last 3 lines
	time.Sleep(100 * time.Millisecond)
	mu.Lock()
	if len(received) < 3 {
		t.Fatalf("expected at least 3 initial lines, got %d", len(received))
	}
	if received[0].Raw != "Line 3" || received[2].Raw != "Line 5" {
		t.Errorf("unexpected initial lines: %v, %v", received[0].Raw, received[2].Raw)
	}
	mu.Unlock()

	// Append new line to file
	f, err := os.OpenFile(testFile, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatalf("failed to open for append: %v", err)
	}
	f.WriteString("Line 6: [ERROR] Something failed\n")
	f.Close()

	// Wait for streaming batch
	time.Sleep(200 * time.Millisecond)
	mu.Lock()
	lastLine := received[len(received)-1]
	mu.Unlock()

	if lastLine.Raw != "Line 6: [ERROR] Something failed" {
		t.Errorf("expected new streamed line, got: %s", lastLine.Raw)
	}
	if lastLine.Level != "ERROR" {
		t.Errorf("expected ERROR level, got: %s", lastLine.Level)
	}
}

func TestTailSession_TruncationRotation(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "tailer_trunc_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	testFile := filepath.Join(tempDir, "service.log")
	if err := os.WriteFile(testFile, []byte("Old line 1\nOld line 2\nOld line 3\n"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	var mu sync.Mutex
	var received []tailer.LogLine

	session, err := tailer.StartTail(testFile, 10, func(batch []tailer.LogLine) {
		mu.Lock()
		received = append(received, batch...)
		mu.Unlock()
	})
	if err != nil {
		t.Fatalf("StartTail failed: %v", err)
	}
	defer session.Stop()

	time.Sleep(100 * time.Millisecond)

	// Truncate file and write fresh content (log rotation simulation)
	if err := os.WriteFile(testFile, []byte("Rotated Line 1\n"), 0644); err != nil {
		t.Fatalf("failed to truncate file: %v", err)
	}

	time.Sleep(300 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	foundRotated := false
	for _, l := range received {
		if l.Raw == "Rotated Line 1" {
			foundRotated = true
			break
		}
	}

	if !foundRotated {
		t.Errorf("expected to find 'Rotated Line 1' after truncation, received: %v", received)
	}
}

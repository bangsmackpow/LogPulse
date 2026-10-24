package tailer_test

import (
	"testing"
	"time"

	"logpulse/internal/tailer"
)

func TestMultiplexer_ChronologicalMerge(t *testing.T) {
	now := time.Now()

	streamA := []tailer.LogLine{
		{FileID: "fileA.log", LineNumber: 1, Timestamp: now.Add(1 * time.Second), Raw: "A1"},
		{FileID: "fileA.log", LineNumber: 2, Timestamp: now.Add(3 * time.Second), Raw: "A2"},
	}

	streamB := []tailer.LogLine{
		{FileID: "fileB.log", LineNumber: 1, Timestamp: now.Add(2 * time.Second), Raw: "B1"},
		{FileID: "fileB.log", LineNumber: 2, Timestamp: now.Add(4 * time.Second), Raw: "B2"},
	}

	merged := tailer.MergeStreams([][]tailer.LogLine{streamA, streamB})

	if len(merged) != 4 {
		t.Fatalf("expected 4 merged lines, got %d", len(merged))
	}

	expectedOrder := []string{"A1", "B1", "A2", "B2"}
	for i, exp := range expectedOrder {
		if merged[i].Raw != exp {
			t.Errorf("at index %d: expected %s, got %s", i, exp, merged[i].Raw)
		}
	}
}

func TestMultiplexer_FilterBySessionMarker(t *testing.T) {
	baseTime := time.Date(2026, 10, 24, 12, 0, 0, 0, time.UTC)

	lines := []tailer.LogLine{
		{LineNumber: 1, Timestamp: baseTime.Add(-10 * time.Second), Raw: "Before session"},
		{LineNumber: 2, Timestamp: baseTime.Add(-1 * time.Second), Raw: "Right before incident"},
		{LineNumber: 3, Timestamp: baseTime.Add(2 * time.Second), Raw: "Incident occurred"},
		{LineNumber: 4, Timestamp: baseTime.Add(5 * time.Second), Raw: "Crash trace"},
	}

	// Filter after baseTime
	after := tailer.FilterAfterTimestamp(lines, baseTime)
	if len(after) != 2 {
		t.Fatalf("expected 2 lines after marker, got %d", len(after))
	}
	if after[0].Raw != "Incident occurred" {
		t.Errorf("unexpected first line: %s", after[0].Raw)
	}
}

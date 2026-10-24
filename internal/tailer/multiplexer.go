package tailer

import (
	"sort"
	"time"
)

// MergeStreams performs a stable chronological K-way merge of multiple log line slices
func MergeStreams(streams [][]LogLine) []LogLine {
	var totalLen int
	for _, s := range streams {
		totalLen += len(s)
	}

	merged := make([]LogLine, 0, totalLen)
	for _, s := range streams {
		merged = append(merged, s...)
	}

	// Sort stably by timestamp
	sort.SliceStable(merged, func(i, j int) bool {
		return merged[i].Timestamp.Before(merged[j].Timestamp)
	})

	return merged
}

// FilterAfterTimestamp filters out all log lines that occurred strictly before the specified timestamp
func FilterAfterTimestamp(lines []LogLine, mark time.Time) []LogLine {
	var filtered []LogLine
	for _, l := range lines {
		if !l.Timestamp.Before(mark) {
			filtered = append(filtered, l)
		}
	}
	return filtered
}

//go:build !windows

package events

import (
	"fmt"
)

type FallbackEventReader struct {
	offline offlineEventReader
}

func NewEventReader() EventReader {
	return &FallbackEventReader{}
}

func (f *FallbackEventReader) ReadRecent(channel string, limit int) ([]EventRecord, error) {
	return nil, fmt.Errorf("live Windows Event Log querying is only supported on Windows")
}

func (f *FallbackEventReader) ReadFile(evtxPath string, limit int) ([]EventRecord, error) {
	return f.offline.ReadFile(evtxPath, limit)
}

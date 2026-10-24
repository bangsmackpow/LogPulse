package events_test

import (
	"os"
	"path/filepath"
	"testing"

	"logpulse/internal/events"
)

func TestEventReader_Creation(t *testing.T) {
	reader := events.NewEventReader()
	if reader == nil {
		t.Fatal("expected non-nil EventReader")
	}
}

func TestEventReader_NonExistentFile(t *testing.T) {
	reader := events.NewEventReader()
	_, err := reader.ReadFile("C:\\nonexistent\\path\\file.evtx", 10)
	if err == nil {
		t.Fatal("expected error when reading non-existent evtx file")
	}
}

func TestEventReader_ParseXMLRecord(t *testing.T) {
	sampleXML := `<Event xmlns="http://schemas.microsoft.com/win/2004/08/events/event">
	<System>
		<Provider Name="Application Error" Guid="{a0e9b465-b939-57e7-b5b3-e83fb3437c5d}"/>
		<EventID>1000</EventID>
		<Version>0</Version>
		<Level>2</Level>
		<Task>100</Task>
		<Opcode>0</Opcode>
		<Keywords>0x80000000000000</Keywords>
		<TimeCreated SystemTime="2026-10-24T12:34:56.789Z"/>
		<Channel>Application</Channel>
		<Computer>DESKTOP-ABC</Computer>
	</System>
	<EventData>
		<Data>Faulting application name: myapp.exe</Data>
		<Data>Faulting module name: ntdll.dll</Data>
	</EventData>
</Event>`

	record, err := events.ParseEventXML([]byte(sampleXML))
	if err != nil {
		t.Fatalf("failed to parse event XML: %v", err)
	}

	if record.ID != 1000 {
		t.Errorf("expected EventID 1000, got %d", record.ID)
	}
	if record.Level != "Error" {
		t.Errorf("expected Level 'Error', got %q", record.Level)
	}
	if record.Provider != "Application Error" {
		t.Errorf("expected Provider 'Application Error', got %q", record.Provider)
	}
	if record.Channel != "Application" {
		t.Errorf("expected Channel 'Application', got %q", record.Channel)
	}
	if len(record.Message) == 0 {
		t.Errorf("expected non-empty message")
	}
}

func TestEventReader_OfflineExportFile(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "evtx_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	fakeEvtx := filepath.Join(tempDir, "sample.xml")
	sampleXML := `<Event xmlns="http://schemas.microsoft.com/win/2004/08/events/event">
	<System>
		<Provider Name="TestSource"/>
		<EventID>42</EventID>
		<Level>3</Level>
		<TimeCreated SystemTime="2026-10-24T12:00:00.000Z"/>
		<Channel>System</Channel>
	</System>
	<EventData><Data>Test warning event</Data></EventData>
</Event>`
	if err := os.WriteFile(fakeEvtx, []byte(sampleXML), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	reader := events.NewEventReader()
	records, err := reader.ReadFile(fakeEvtx, 10)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	if records[0].Level != "Warning" {
		t.Errorf("expected Warning level, got %s", records[0].Level)
	}
}

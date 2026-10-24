package events

import (
	"encoding/xml"
	"fmt"
	"os"
	"strings"
	"time"
)

type EventRecord struct {
	ID        int64     `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"`
	Provider  string    `json:"provider"`
	Channel   string    `json:"channel"`
	Message   string    `json:"message"`
}

type EventReader interface {
	ReadRecent(channel string, limit int) ([]EventRecord, error)
	ReadFile(evtxPath string, limit int) ([]EventRecord, error)
}

// Win32 Event XML Schema structures
type xmlEvent struct {
	XMLName xml.Name  `xml:"Event"`
	System  xmlSystem `xml:"System"`
	EventData xmlEventData `xml:"EventData"`
	RenderingInfo xmlRenderingInfo `xml:"RenderingInfo"`
}

type xmlSystem struct {
	Provider    xmlProvider `xml:"Provider"`
	EventID     int64       `xml:"EventID"`
	Level       int         `xml:"Level"`
	TimeCreated xmlTime     `xml:"TimeCreated"`
	Channel     string      `xml:"Channel"`
	Computer    string      `xml:"Computer"`
}

type xmlProvider struct {
	Name string `xml:"Name,attr"`
}

type xmlTime struct {
	SystemTime string `xml:"SystemTime,attr"`
}

type xmlEventData struct {
	Data []string `xml:"Data"`
}

type xmlRenderingInfo struct {
	Message string `xml:"Message"`
}

func MapEventLevel(levelNum int) string {
	switch levelNum {
	case 1:
		return "Critical"
	case 2:
		return "Error"
	case 3:
		return "Warning"
	case 4:
		return "Information"
	case 5:
		return "Verbose"
	default:
		return "Information"
	}
}

func ParseEventXML(data []byte) (*EventRecord, error) {
	var ev xmlEvent
	if err := xml.Unmarshal(data, &ev); err != nil {
		return nil, fmt.Errorf("failed to parse event xml: %w", err)
	}

	ts, err := time.Parse(time.RFC3339Nano, ev.System.TimeCreated.SystemTime)
	if err != nil {
		ts = time.Now()
	}

	msg := ev.RenderingInfo.Message
	if strings.TrimSpace(msg) == "" && len(ev.EventData.Data) > 0 {
		msg = strings.Join(ev.EventData.Data, " | ")
	}

	return &EventRecord{
		ID:        ev.System.EventID,
		Timestamp: ts,
		Level:     MapEventLevel(ev.System.Level),
		Provider:  ev.System.Provider.Name,
		Channel:   ev.System.Channel,
		Message:   msg,
	}, nil
}

// Offline fallback reader for reading exported/standalone XML or text event dumps
type offlineEventReader struct{}

func (r *offlineEventReader) ReadFile(path string, limit int) ([]EventRecord, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Try parsing as single XML or split XMLs
	rec, err := ParseEventXML(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse evtx file content: %w", err)
	}

	return []EventRecord{*rec}, nil
}

package tailer

import (
	"bufio"
	"io"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

type LogLine struct {
	FileID     string    `json:"fileId"`
	LineNumber int64     `json:"lineNumber"`
	Timestamp  time.Time `json:"timestamp"`
	Raw        string    `json:"raw"`
	Level      string    `json:"level"`
}

var (
	reError = regexp.MustCompile(`(?i)\b(error|fatal|critical|exception|panic)\b`)
	reWarn  = regexp.MustCompile(`(?i)\b(warn|warning)\b`)
	reDebug = regexp.MustCompile(`(?i)\b(debug|trace|dbg)\b`)
)

func ParseLogLevel(raw string) string {
	if reError.MatchString(raw) {
		return "ERROR"
	}
	if reWarn.MatchString(raw) {
		return "WARN"
	}
	if reDebug.MatchString(raw) {
		return "DEBUG"
	}
	return "INFO"
}

// SanitizeLogLine replaces invalid UTF-8 byte sequences with unicode replacement character
func SanitizeLogLine(raw string) string {
	return strings.ToValidUTF8(raw, "\uFFFD")
}

var timestampFormats = []string{
	time.RFC3339Nano,
	time.RFC3339,
	"2006-01-02 15:04:05.000",
	"2006-01-02 15:04:05",
	"2006-01-02T15:04:05",
	"Jan _2 15:04:05",
}

// ExtractTimestamp attempts to parse standard timestamp formats at the beginning of a log line
func ExtractTimestamp(raw string) (time.Time, bool) {
	trimmed := strings.TrimSpace(raw)
	if len(trimmed) < 15 {
		return time.Time{}, false
	}

	for _, format := range timestampFormats {
		prefixLen := len(format)
		if len(trimmed) >= prefixLen {
			candidate := trimmed[:prefixLen]
			if t, err := time.Parse(format, candidate); err == nil {
				return t, true
			}
		}
	}
	return time.Time{}, false
}

type TailSession struct {
	filePath   string
	maxTail    int
	onBatch    func([]LogLine)
	batcher    *Batcher
	watcher    *fsnotify.Watcher
	file       *os.File
	offset     int64
	lineNum    int64
	stopChan   chan struct{}
	stopped    sync.Once
	mu         sync.Mutex
}

func StartTail(filePath string, maxTailLines int, onBatch func([]LogLine)) (*TailSession, error) {
	if maxTailLines <= 0 {
		maxTailLines = 1000
	}

	file, err := openSharedRead(filePath)
	if err != nil {
		return nil, err
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		file.Close()
		return nil, err
	}

	if err := watcher.Add(filePath); err != nil {
		// If watching the file directly fails, watch the parent directory
		dir := filePath
		if fi, sErr := os.Stat(filePath); sErr == nil && !fi.IsDir() {
			dir = filePath[:strings.LastIndex(filePath, string(os.PathSeparator))]
		}
		_ = watcher.Add(dir)
	}

	session := &TailSession{
		filePath: filePath,
		maxTail:  maxTailLines,
		onBatch:  onBatch,
		file:     file,
		watcher:  watcher,
		stopChan: make(chan struct{}),
	}

	session.batcher = NewBatcher(100, 50*time.Millisecond, onBatch)

	// Read initial tail
	session.readInitialTail()

	go session.watchLoop()
	return session, nil
}

func (s *TailSession) readInitialTail() {
	s.mu.Lock()
	defer s.mu.Unlock()

	stat, err := s.file.Stat()
	if err != nil {
		return
	}

	size := stat.Size()
	var lines []string

	// If file is smaller than 2MB, read all and take last N lines
	if size < 2*1024*1024 {
		_, _ = s.file.Seek(0, io.SeekStart)
		scanner := bufio.NewScanner(s.file)
		for scanner.Scan() {
			lines = append(lines, scanner.Text())
		}
		s.offset, _ = s.file.Seek(0, io.SeekCurrent)
	} else {
		// Seek to near end (e.g. last 512KB)
		readOffset := size - (512 * 1024)
		if readOffset < 0 {
			readOffset = 0
		}
		_, _ = s.file.Seek(readOffset, io.SeekStart)
		reader := bufio.NewReader(s.file)
		// Discard first partial line
		_, _ = reader.ReadString('\n')
		for {
			line, err := reader.ReadString('\n')
			if len(line) > 0 {
				lines = append(lines, strings.TrimRight(line, "\r\n"))
			}
			if err != nil {
				break
			}
		}
		s.offset, _ = s.file.Seek(0, io.SeekCurrent)
	}

	if len(lines) > s.maxTail {
		lines = lines[len(lines)-s.maxTail:]
	}

	for _, text := range lines {
		s.lineNum++
		cleanText := SanitizeLogLine(text)
		ts, hasTs := ExtractTimestamp(cleanText)
		if !hasTs {
			ts = time.Now()
		}
		s.batcher.Add(LogLine{
			FileID:     s.filePath,
			LineNumber: s.lineNum,
			Timestamp:  ts,
			Raw:        cleanText,
			Level:      ParseLogLevel(cleanText),
		})
	}
	s.batcher.Flush()
}

func (s *TailSession) watchLoop() {
	pollTicker := time.NewTicker(200 * time.Millisecond)
	defer pollTicker.Stop()

	for {
		select {
		case <-s.stopChan:
			return
		case event, ok := <-s.watcher.Events:
			if !ok {
				return
			}
			if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) {
				s.readNewLines()
			}
		case <-pollTicker.C:
			// Fallback polling for rotation / truncation
			s.readNewLines()
		}
	}
}

func (s *TailSession) readNewLines() {
	s.mu.Lock()
	defer s.mu.Unlock()

	stat, err := os.Stat(s.filePath)
	if err != nil {
		return
	}

	currentSize := stat.Size()
	// Check for file truncation / rotation
	if currentSize < s.offset {
		// File was truncated or rotated! Reset offset
		s.offset = 0
		_ = s.file.Close()
		newFile, err := openSharedRead(s.filePath)
		if err != nil {
			return
		}
		s.file = newFile
	}

	if currentSize == s.offset {
		return
	}

	_, err = s.file.Seek(s.offset, io.SeekStart)
	if err != nil {
		return
	}

	reader := bufio.NewReader(s.file)
	for {
		line, err := reader.ReadString('\n')
		if len(line) > 0 {
			cleanLine := SanitizeLogLine(strings.TrimRight(line, "\r\n"))
			s.lineNum++
			ts, hasTs := ExtractTimestamp(cleanLine)
			if !hasTs {
				ts = time.Now()
			}
			s.batcher.Add(LogLine{
				FileID:     s.filePath,
				LineNumber: s.lineNum,
				Timestamp:  ts,
				Raw:        cleanLine,
				Level:      ParseLogLevel(cleanLine),
			})
		}
		if err != nil {
			break
		}
	}

	newOffset, err := s.file.Seek(0, io.SeekCurrent)
	if err == nil {
		s.offset = newOffset
	}
}

func (s *TailSession) Stop() error {
	s.stopped.Do(func() {
		close(s.stopChan)
		s.batcher.Stop()
		if s.watcher != nil {
			_ = s.watcher.Close()
		}
		if s.file != nil {
			_ = s.file.Close()
		}
	})
	return nil
}

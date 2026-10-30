package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"

	wailsRuntime "github.com/wailsapp/wails/v2/pkg/runtime"

	"logpulse/internal/ai"
	"logpulse/internal/db"
	"logpulse/internal/discovery"
	"logpulse/internal/events"
	"logpulse/internal/tailer"
)

type ScrubResult struct {
	Text    string          `json:"text"`
	Matches []ai.ScrubMatch `json:"matches"`
}

type App struct {
	ctx         context.Context
	store       *db.Store
	scanner     *discovery.Scanner
	eventReader events.EventReader
	aiClient    *ai.MultiProviderClient
	activeTails map[string]*tailer.TailSession
	cancelTier2 context.CancelFunc
	mu          sync.Mutex
}

func NewApp() *App {
	userConfig, err := os.UserConfigDir()
	if err != nil {
		userConfig = "."
	}
	appDir := filepath.Join(userConfig, "LogPulse")
	_ = os.MkdirAll(appDir, 0755)
	dbPath := filepath.Join(appDir, "logpulse.db")

	store, err := db.NewStore(dbPath)
	if err != nil {
		fmt.Printf("Warning: failed to initialize SQLite store at %s: %v\n", dbPath, err)
	}

	scanner := discovery.NewScanner(store)
	eventReader := events.NewEventReader()
	aiClient := ai.NewMultiProviderClient(store)

	return &App{
		store:       store,
		scanner:     scanner,
		eventReader: eventReader,
		aiClient:    aiClient,
		activeTails: make(map[string]*tailer.TailSession),
	}
}

func NewTestApp(dbPath string) (*App, error) {
	store, err := db.NewStore(dbPath)
	if err != nil {
		return nil, err
	}
	return &App{
		store:       store,
		scanner:     discovery.NewScanner(store),
		eventReader: events.NewEventReader(),
		aiClient:    ai.NewMultiProviderClient(store),
		activeTails: make(map[string]*tailer.TailSession),
	}, nil
}

func (a *App) GetStore() *db.Store {
	return a.store
}

func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx

	// Run initial Tier 1 instant scan
	go func() {
		logs, err := a.scanner.ScanTier1()
		if err == nil && len(logs) > 0 {
			wailsRuntime.EventsEmit(a.ctx, "logs:tier1_complete", logs)
		}

		// Start background Tier 2 scan
		t2Ctx, cancel := context.WithCancel(context.Background())
		a.mu.Lock()
		a.cancelTier2 = cancel
		a.mu.Unlock()

		a.scanner.StartTier2Background(t2Ctx, func(l db.DiscoveredLog) {
			wailsRuntime.EventsEmit(a.ctx, "log:discovered", l)
		})
	}()
}

func (a *App) Shutdown(ctx context.Context) {
	a.mu.Lock()
	if a.cancelTier2 != nil {
		a.cancelTier2()
	}
	for _, session := range a.activeTails {
		_ = session.Stop()
	}
	a.activeTails = make(map[string]*tailer.TailSession)
	a.mu.Unlock()

	if a.store != nil {
		_ = a.store.Close()
	}
}

func (a *App) GetDiscoveredLogs() ([]db.DiscoveredLog, error) {
	if a.store == nil {
		return []db.DiscoveredLog{}, nil
	}
	return a.store.ListDiscoveredLogs()
}

func (a *App) RefreshLogs() ([]db.DiscoveredLog, error) {
	return a.scanner.ScanTier1()
}

func (a *App) StartTail(path string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if _, exists := a.activeTails[path]; exists {
		return nil // already tailing
	}

	// Handle Windows .evtx event logs
	if filepath.Ext(path) == ".evtx" || filepath.Ext(path) == ".xml" {
		records, err := a.eventReader.ReadFile(path, 500)
		if err == nil && len(records) > 0 {
			var lines []tailer.LogLine
			for i, r := range records {
				lvl := "INFO"
				if r.Level == "Error" || r.Level == "Critical" {
					lvl = "ERROR"
				} else if r.Level == "Warning" {
					lvl = "WARN"
				}
				lines = append(lines, tailer.LogLine{
					FileID:     path,
					LineNumber: int64(i + 1),
					Timestamp:  r.Timestamp,
					Raw:        fmt.Sprintf("[%s] EventID %d (%s): %s", r.Level, r.ID, r.Provider, r.Message),
					Level:      lvl,
				})
			}
			if a.ctx != nil {
				wailsRuntime.EventsEmit(a.ctx, "log:batch", lines)
			}
			return nil
		}
	}

	session, err := tailer.StartTail(path, 1000, func(batch []tailer.LogLine) {
		if a.ctx != nil {
			wailsRuntime.EventsEmit(a.ctx, "log:batch", batch)
		}
	})
	if err != nil {
		return err
	}

	a.activeTails[path] = session
	return nil
}

func (a *App) StopTail(path string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if session, exists := a.activeTails[path]; exists {
		_ = session.Stop()
		delete(a.activeTails, path)
	}
	return nil
}

func (a *App) ScrubText(text string) (ScrubResult, error) {
	scrubbed, matches := ai.ScrubPII(text)
	return ScrubResult{
		Text:    scrubbed,
		Matches: matches,
	}, nil
}

func (a *App) EstimateCost(provider, model, prompt string) map[string]interface{} {
	tokens := ai.EstimateTokens(prompt)
	cost := ai.CalculateCostUSD(provider, model, tokens, 500)
	return map[string]interface{}{
		"tokens":  tokens,
		"costUsd": cost,
	}
}

func (a *App) InvestigateWithAI(req ai.AIRequest) (*ai.AIResponse, error) {
	// Auto-scrub prompt for privacy
	scrubbedPrompt, _ := ai.ScrubPII(req.Prompt)
	req.Prompt = scrubbedPrompt

	ctx := context.Background()
	if a.ctx != nil {
		ctx = a.ctx
	}
	return a.aiClient.Query(ctx, req)
}

func (a *App) OpenFileInEditor(filePath string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", "", filePath)
	case "darwin":
		cmd = exec.Command("open", filePath)
	default:
		cmd = exec.Command("xdg-open", filePath)
	}
	return cmd.Start()
}

func (a *App) RevealInExplorer(filePath string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("explorer", fmt.Sprintf("/select,%s", filePath))
	case "darwin":
		cmd = exec.Command("open", "-R", filePath)
	default:
		cmd = exec.Command("xdg-open", filepath.Dir(filePath))
	}
	return cmd.Start()
}

func (a *App) GetSettings() (map[string]string, error) {
	settings := make(map[string]string)
	if a.store == nil {
		return settings, nil
	}

	keys := []string{
		"theme",
		"provider",
		"model",
		"api_key_openai",
		"api_key_anthropic",
		"api_key_gemini",
		"api_key_openrouter",
		"api_key_deepseek",
		"api_key_groq",
		"ollama_endpoint",
		"monthly_budget_usd",
	}

	for _, k := range keys {
		val, err := a.store.GetSetting(k)
		if err == nil && val != "" {
			settings[k] = val
		}
	}
	return settings, nil
}

func (a *App) SaveSettings(settings map[string]string) error {
	if a.store == nil {
		return fmt.Errorf("store not initialized")
	}

	for k, v := range settings {
		if err := a.store.SetSetting(k, v); err != nil {
			return err
		}
	}
	return nil
}

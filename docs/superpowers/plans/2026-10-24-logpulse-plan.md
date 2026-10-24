# LogPulse Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build LogPulse, a high-performance cross-platform desktop log discovery and diagnostic workbench in Go and Wails v2 with a React/Vite/Tailwind frontend, featuring instant tiered log crawling, 60fps virtualized tail streaming, Windows `.evtx` parsing, and a privacy-first AI diagnostic assistant with PII redaction and cost estimation.

**Architecture:** The Go backend handles multi-threaded tiered file discovery, zero-copy live tail streaming with micro-batched IPC event dispatching, pure-Go SQLite persistence, and multi-provider AI client bridging with local regex PII redaction. The Vite + React 18 frontend uses `@tanstack/react-virtual` for 250k+ line rendering, off-thread Web Worker regex search, Zustand state management, and an interactive context-bracketing AI investigation drawer.

**Tech Stack:** Go 1.22+, Wails v2, `modernc.org/sqlite`, `fsnotify/fsnotify`, `golang.org/x/sys/windows`, React 18, TypeScript, Vite, Tailwind CSS, Lucide Icons, `@tanstack/react-virtual`, Zustand.

**Spec:** [`docs/superpowers/specs/2026-10-24-logpulse-design.md`](file:///C:/Users/curtis/Desktop/dev/logpulse/docs/superpowers/specs/2026-10-24-logpulse-design.md)

## Global Constraints
- Target platforms: Windows 10/11 (Tier 1 first-class), macOS, Linux.
- Pure-Go SQLite (`modernc.org/sqlite`) with WAL mode enabled to avoid CGO dependency.
- Wails IPC bridge must never emit 1 event per log line; batch in 50ms micro-ticks or 100-line chunks.
- Virtualized log viewer must maintain 60 FPS up to 250,000 lines without UI lockup.
- All outbound AI prompts must pass through the local regex PII/secret scrubber before dispatch.
- Right-click context actions ("Open in Default Editor", "Reveal in Explorer", "Investigate with AI") must be supported across log catalog and log stream.

## Review Focus
- **Win32 file sharing violation (`ERROR_SHARING_VIOLATION`):** File scanner and tailer must open files with `FILE_SHARE_READ | FILE_SHARE_WRITE | FILE_SHARE_DELETE` so active logs locked by third-party applications can still be read.
- **Log rotation / truncation mid-stream:** Tailer must detect when file size becomes smaller than the previous read offset, automatically reset offset to 0, and continue streaming without crashing.
- **Binary / Corrupted log files:** Discovery and parser must inspect initial bytes, sanitize invalid UTF-8 sequences using replacement characters, and avoid crashing JSON serialization across the Wails IPC bridge.
- **High-throughput log storm (10k+ lines/sec):** Micro-batch buffer must enforce ring-buffer limits (max 250,000 lines) and drop older in-memory lines rather than ballooning RAM or saturating the webview.
- **AI provider rate limits or network timeout:** Multi-provider HTTP client must enforce a 30s timeout and return structured user-friendly error codes instead of leaving the UI in an indefinite pending state.

---

### Task 1: Go Module Setup & Pure-Go SQLite Persistence (`internal/db`)

**Files:**
- Create: `go.mod`
- Create: `internal/db/db.go`
- Create: `internal/db/db_test.go`
- Create: `internal/db/models.go`

**Interfaces:**
- Consumes: Go standard library, `modernc.org/sqlite`
- Produces:
  - `type Store struct`
  - `func NewStore(dbPath string) (*Store, error)`
  - `func (s *Store) Close() error`
  - `func (s *Store) SaveDiscoveredLog(log DiscoveredLog) error`
  - `func (s *Store) ListDiscoveredLogs() ([]DiscoveredLog, error)`
  - `func (s *Store) RecordAIAccounting(entry AIAccountingEntry) error`
  - `func (s *Store) GetMonthlyAISpendUSD() (float64, error)`
  - `func (s *Store) SetSetting(key, val string) error`
  - `func (s *Store) GetSetting(key string) (string, error)`

- [ ] **Step 1: Write the failing test**
Create `internal/db/db_test.go` testing database initialization with WAL mode, inserting and listing discovered logs, recording AI token spend, calculating monthly spend, and setting/getting configuration values.

- [ ] **Step 2: Run test to verify it fails**
Run: `go test -v ./internal/db`  
Expected: FAIL (files/packages do not exist yet)

- [ ] **Step 3: Implement `internal/db` with `modernc.org/sqlite`**
Initialize `go.mod` with module name `logpulse`, implement schema creation (`discovered_logs`, `ai_accounting`, `settings`), enable `PRAGMA journal_mode=WAL;`, and implement CRUD repository methods.

- [ ] **Step 4: Run test to verify it passes**
Run: `go test -v ./internal/db`  
Expected: PASS

- [ ] **Step 5: Commit**
```bash
git add go.mod go.sum internal/db
git commit -m "feat(db): implement pure-Go SQLite storage with WAL mode"
```

---

### Task 2: Tiered Discovery & Scanner Engine (`internal/discovery`)

**Files:**
- Create: `internal/discovery/scanner.go`
- Create: `internal/discovery/scanner_test.go`
- Create: `internal/discovery/paths_windows.go`
- Create: `internal/discovery/paths_other.go`

**Interfaces:**
- Consumes: `internal/db`, Go `os`, `path/filepath`
- Produces:
  - `type Scanner struct`
  - `func NewScanner(store *db.Store) *Scanner`
  - `func (s *Scanner) ScanTier1() ([]db.DiscoveredLog, error)`
  - `func (s *Scanner) StartTier2Background(ctx context.Context, onFound func(db.DiscoveredLog))`
  - `func InferAppName(filePath string) string`

- [ ] **Step 1: Write the failing test**
Create `internal/discovery/scanner_test.go` with mock directory hierarchies verifying:
1. `InferAppName` accurately derives application names from parent folders or executable filenames.
2. `ScanTier1` crawls designated directories and discovers `*.log`, `*.txt`, `*.trace`, `*.jsonl`, and `.out` files.
3. Excluded directories (`.git`, `node_modules`, `AppData\Local\Temp`) are ignored.
4. Locked files are opened safely with shared-read flags.

- [ ] **Step 2: Run test to verify it fails**
Run: `go test -v ./internal/discovery`  
Expected: FAIL

- [ ] **Step 3: Implement scanner logic and platform path resolvers**
Implement `internal/discovery/paths_windows.go` (reading `%LOCALAPPDATA%`, `%APPDATA%`, `%PROGRAMDATA%`, Windows Logs) and `paths_other.go` for Linux/macOS. Implement worker pool with 5ms sleep throttling in Tier 2 to keep CPU/IO under 5%.

- [ ] **Step 4: Run test to verify it passes**
Run: `go test -v ./internal/discovery`  
Expected: PASS

- [ ] **Step 5: Commit**
```bash
git add internal/discovery
git commit -m "feat(discovery): implement Tier 1 instant and Tier 2 background log scanner"
```

---

### Task 3: Windows Event Log (`.evtx`) Parser & Event Normalizer (`internal/events`)

**Files:**
- Create: `internal/events/events.go`
- Create: `internal/events/events_windows.go`
- Create: `internal/events/events_fallback.go`
- Create: `internal/events/events_test.go`

**Interfaces:**
- Consumes: Go standard library, `golang.org/x/sys/windows`
- Produces:
  - `type EventRecord struct { ID int64, Timestamp time.Time, Level string, Provider string, Channel string, Message string }`
  - `type EventReader interface { ReadRecent(channel string, limit int) ([]EventRecord, error); ReadFile(evtxPath string, limit int) ([]EventRecord, error) }`
  - `func NewEventReader() EventReader`

- [ ] **Step 1: Write the failing test**
Create `internal/events/events_test.go` verifying record normalization, level string mapping (Error, Warning, Information), XML/text message extraction, and graceful handling of missing or inaccessible channels.

- [ ] **Step 2: Run test to verify it fails**
Run: `go test -v ./internal/events`  
Expected: FAIL

- [ ] **Step 3: Implement Windows `wevtapi.dll` wrapper and cross-platform fallback**
Implement Win32 `EvtQuery`, `EvtNext`, `EvtRender` using `golang.org/x/sys/windows` under `//go:build windows`. Implement `events_fallback.go` under `//go:build !windows` returning standard status and supporting offline XML/exported file reads.

- [ ] **Step 4: Run test to verify it passes**
Run: `go test -v ./internal/events`  
Expected: PASS

- [ ] **Step 5: Commit**
```bash
git add internal/events
git commit -m "feat(events): implement Windows Event Log reader with cross-platform fallback"
```

---

### Task 4: High-Performance Live Tailer & Rotation Engine (`internal/tailer`)

**Files:**
- Create: `internal/tailer/tailer.go`
- Create: `internal/tailer/tailer_test.go`
- Create: `internal/tailer/batcher.go`

**Interfaces:**
- Consumes: `github.com/fsnotify/fsnotify`, Go `os`
- Produces:
  - `type LogLine struct { FileID string, LineNumber int64, Timestamp time.Time, Raw string, Level string }`
  - `type TailSession struct`
  - `func StartTail(filePath string, maxTailLines int, onBatch func([]LogLine)) (*TailSession, error)`
  - `func (s *TailSession) Stop() error`
  - `func ParseLogLevel(raw string) string`

- [ ] **Step 1: Write the failing test**
Create `internal/tailer/tailer_test.go` testing:
1. Reading initial tail of last $N$ lines from a large file without reading the whole file into memory.
2. Streaming new appended lines in batched chunks.
3. Handling file truncation (file size becomes smaller -> seek to 0 and stream new content).
4. Extracting timestamps and grouping multiline stack traces with parent log line.
5. Level detection (ERROR, WARN, INFO, DEBUG).

- [ ] **Step 2: Run test to verify it fails**
Run: `go test -v ./internal/tailer`  
Expected: FAIL

- [ ] **Step 3: Implement Tailer and Micro-Batcher**
Implement `internal/tailer/tailer.go` and `batcher.go` with 50ms ticker or 100-line threshold flush. Use shared read locks (`os.OpenFile` with appropriate mode or Windows file flags) so active files can be read concurrently with running apps.

- [ ] **Step 4: Run test to verify it passes**
Run: `go test -v ./internal/tailer`  
Expected: PASS

- [ ] **Step 5: Commit**
```bash
git add internal/tailer
git commit -m "feat(tailer): implement zero-copy live tailer with rotation detection and micro-batching"
```

---

### Task 5: AI Diagnostic Engine, PII Scrubber & Cost Tokenizer (`internal/ai`)

**Files:**
- Create: `internal/ai/scrubber.go`
- Create: `internal/ai/scrubber_test.go`
- Create: `internal/ai/tokenizer.go`
- Create: `internal/ai/tokenizer_test.go`
- Create: `internal/ai/client.go`
- Create: `internal/ai/client_test.go`

**Interfaces:**
- Consumes: Go standard library (`net/http`, `regexp`), `internal/db`
- Produces:
  - `func ScrubPII(text string) (string, []ScrubMatch)`
  - `func EstimateTokens(text string) int`
  - `func CalculateCostUSD(provider, model string, inTokens, outTokens int) float64`
  - `type AIClient interface { Query(ctx context.Context, req AIRequest) (*AIResponse, error) }`
  - `func NewMultiProviderClient(store *db.Store) *MultiProviderClient`

- [ ] **Step 1: Write the failing test**
Create unit tests in `scrubber_test.go` testing regex scrubbing for IPv4, IPv6, JWT tokens (`eyJ...`), API keys (e.g. `sk-...`, `AKIA...`), emails, and passwords. Test token calculation and cost estimation in `tokenizer_test.go`. Test mock provider HTTP dispatch and budget ceiling enforcement in `client_test.go`.

- [ ] **Step 2: Run test to verify it fails**
Run: `go test -v ./internal/ai`  
Expected: FAIL

- [ ] **Step 3: Implement PII scrubber, tokenizer, and multi-provider clients**
Implement regex pattern library in `scrubber.go`. Implement token estimation and pricing tables for Claude 3.5 Sonnet, GPT-4o/mini, Gemini 1.5/2.0, and local Ollama (free/0 cost). Implement clean HTTP dispatch with 30s timeout and monthly budget limit checks in `client.go`.

- [ ] **Step 4: Run test to verify it passes**
Run: `go test -v ./internal/ai`  
Expected: PASS

- [ ] **Step 5: Commit**
```bash
git add internal/ai
git commit -m "feat(ai): implement local PII scrubber, token cost estimator, and multi-provider client"
```

---

### Task 6: Application IPC Bridge & Wails Service Layer (`app.go`, `main.go`, `wails.json`)

**Files:**
- Create: `app.go`
- Create: `app_test.go`
- Create: `main.go`
- Create: `wails.json`

**Interfaces:**
- Consumes: `internal/db`, `internal/discovery`, `internal/events`, `internal/tailer`, `internal/ai`, Wails v2 runtime
- Produces:
  - `type App struct`
  - Wails IPC Methods:
    - `GetDiscoveredLogs() ([]db.DiscoveredLog, error)`
    - `RefreshLogs() ([]db.DiscoveredLog, error)`
    - `StartTail(path string) error`
    - `StopTail(path string) error`
    - `InvestigateWithAI(req AIInvestigationPayload) (*AIResponse, error)`
    - `OpenFileInEditor(path string) error`
    - `RevealInExplorer(path string) error`
    - `GetSettings() (map[string]string, error)`
    - `SaveSettings(settings map[string]string) error`

- [ ] **Step 1: Write the failing test**
Create `app_test.go` testing App initialization with in-memory SQLite store, mock tailing, settings retrieval, and OS open/reveal command generation.

- [ ] **Step 2: Run test to verify it fails**
Run: `go test -v .`  
Expected: FAIL

- [ ] **Step 3: Implement `app.go`, `main.go`, and `wails.json`**
Wire all internal subsystems into `App`. Connect `tailer.StartTail` batch callbacks to `runtime.EventsEmit(a.ctx, "log:batch", batch)`. Implement `OpenFileInEditor` using `cmd /c start` on Windows (`open` on macOS, `xdg-open` on Linux) and `explorer /select,path` for file reveal.

- [ ] **Step 4: Run test to verify it passes**
Run: `go test -v .`  
Expected: PASS

- [ ] **Step 5: Commit**
```bash
git add app.go app_test.go main.go wails.json
git commit -m "feat(app): implement Wails v2 lifecycle bindings and IPC methods"
```

---

### Task 7: Frontend Scaffolding, Theme & Zustand Stores (`frontend/`)

**Files:**
- Create: `frontend/package.json`
- Create: `frontend/vite.config.ts`
- Create: `frontend/tsconfig.json`
- Create: `frontend/tailwind.config.js`
- Create: `frontend/src/index.css`
- Create: `frontend/src/main.tsx`
- Create: `frontend/src/App.tsx`
- Create: `frontend/src/stores/useLogStore.ts`
- Create: `frontend/src/stores/useAIStore.ts`
- Create: `frontend/src/types.ts`

**Interfaces:**
- Consumes: React 18, Vite, Tailwind CSS, Zustand, Lucide Icons
- Produces:
  - `useLogStore`: manages catalog list, selected logs, active tail streams, search term, level filters, troubleshooting session marker.
  - `useAIStore`: manages API keys, selected provider/model, monthly budget limit, context modal state, AI investigation history.

- [ ] **Step 1: Scaffold frontend package and write store test**
Create `frontend/package.json` with dependencies (`@tanstack/react-virtual`, `zustand`, `lucide-react`, `tailwindcss`, `vite`). Create unit tests for `useLogStore` testing line ingestion, session markers, and filter toggling.

- [ ] **Step 2: Run test / typecheck to verify it fails**
Run: `cd frontend && npm install && npm run build`  
Expected: FAIL (missing source components)

- [ ] **Step 3: Implement types, Zustand stores, and base layout in `App.tsx`**
Implement `types.ts`, `useLogStore.ts`, `useAIStore.ts`, and full dark-theme workbench layout with split sidebar (Log Catalog), main viewport (Log Stream), and top troubleshooting action bar.

- [ ] **Step 4: Run build & tests to verify it passes**
Run: `cd frontend && npm run build`  
Expected: PASS

- [ ] **Step 5: Commit**
```bash
git add frontend
git commit -m "feat(frontend): scaffold Vite+React application with Zustand stores and Tailwind theme"
```

---

### Task 8: Virtual Scroll Log Viewer & Search Web Worker (`frontend/src/components/VirtualizedGrid`)

**Files:**
- Create: `frontend/src/components/VirtualizedGrid.tsx`
- Create: `frontend/src/components/LogTree.tsx`
- Create: `frontend/src/components/ContextMenu.tsx`
- Create: `frontend/src/workers/searchWorker.ts`
- Create: `frontend/src/hooks/useLogStream.ts`

**Interfaces:**
- Consumes: `@tanstack/react-virtual`, `useLogStore`, Wails IPC runtime
- Produces:
  - `<VirtualizedGrid />`: 60 FPS rendering up to 250,000 lines, sticky gutter, colorized log levels, match highlighting.
  - `<LogTree />`: Categorized log catalog with quick filter, file size, last modified timestamp.
  - `<ContextMenu />`: Custom right-click menu with "Investigate with AI", "Open in Default Editor", "Reveal in Explorer".

- [ ] **Step 1: Write component tests / story verification**
Write tests for virtualized grid rendering, line selection bracketing, and Web Worker search filter matching.

- [ ] **Step 2: Verify tests fail**
Run: `cd frontend && npm run build`  
Expected: FAIL

- [ ] **Step 3: Implement `VirtualizedGrid`, `searchWorker`, and `LogTree`**
Implement `@tanstack/react-virtual` row virtualizer with dynamic measurement. Implement `searchWorker.ts` off the main thread for instant regex matching across 250k lines. Connect custom right-click context menu.

- [ ] **Step 4: Verify build and component pass**
Run: `cd frontend && npm run build`  
Expected: PASS

- [ ] **Step 5: Commit**
```bash
git add frontend/src/components frontend/src/workers frontend/src/hooks
git commit -m "feat(frontend): implement 250k line virtualized grid, Web Worker search, and context actions"
```

---

### Task 9: AI Assist Modal & Diagnostic Workbench Drawer (`frontend/src/components/AIModal`)

**Files:**
- Create: `frontend/src/components/AIModal.tsx`
- Create: `frontend/src/components/PIIScrubberDiff.tsx`
- Create: `frontend/src/components/CostBadge.tsx`
- Create: `frontend/src/components/SettingsModal.tsx`

**Interfaces:**
- Consumes: `useAIStore`, `useLogStore`, Wails IPC methods
- Produces:
  - `<AIModal />`: Context slider ($N$ lines before, $M$ lines after), user notes textarea, telemetry toggle, PII scrubbing diff toggle, token/cost badge, and "Investigate" action.
  - `<SettingsModal />`: API key management for Anthropic, OpenAI, Gemini, and Ollama endpoint configuration with monthly budget limits.

- [ ] **Step 1: Write component tests for AI modal and PII diff preview**
Test context window calculation ($N$ before + selected + $M$ after), redaction diff highlighting, and budget warning when cost exceeds limit.

- [ ] **Step 2: Verify test fails**
Run: `cd frontend && npm run build`  
Expected: FAIL

- [ ] **Step 3: Implement `AIModal`, `PIIScrubberDiff`, `CostBadge`, and `SettingsModal`**
Build modal with live token estimation, interactive scrubber toggle showing `[REDACTED_IP]` diffs, provider dropdown, markdown response renderer, and copy solution button.

- [ ] **Step 4: Verify build passes**
Run: `cd frontend && npm run build`  
Expected: PASS

- [ ] **Step 5: Commit**
```bash
git add frontend/src/components
git commit -m "feat(frontend): implement AI investigation modal, PII diff scrubber, and settings manager"
```

---

### Task 10: Multi-Log Multiplexing, Troubleshooting Sessions & End-to-End Verification

**Files:**
- Create: `frontend/src/components/SessionControl.tsx`
- Create: `internal/tailer/multiplexer.go`
- Create: `internal/tailer/multiplexer_test.go`
- Create: `test_e2e.go`
- Modify: `README.md`

**Interfaces:**
- Consumes: All internal backend packages and frontend components
- Produces:
  - Merged chronological stream for 2+ active files.
  - Visual pin / marker in the log stream isolating logs before/after the incident reproduction.
  - Complete test suite passing (`go test ./...` and `npm run build`).

- [ ] **Step 1: Write multiplexer test**
Test K-way merge sorting of multiple log files with interleaved timestamps and session marker placement.

- [ ] **Step 2: Run test to verify it fails**
Run: `go test -v ./internal/tailer`  
Expected: FAIL

- [ ] **Step 3: Implement stream multiplexer and session marker controls**
Implement chronological stream combiner in Go and add "Start Troubleshooting Session" / "Drop Marker" UI controls in `SessionControl.tsx`. Create comprehensive `README.md` with architecture guide, setup, and build instructions.

- [ ] **Step 4: Run full backend and frontend verification**
Run:
```bash
go test -v ./...
cd frontend && npm run build
```
Expected: All backend tests PASS, frontend builds with 0 errors.

- [ ] **Step 5: Commit**
```bash
git add internal/tailer frontend/src/components README.md
git commit -m "feat: complete log multiplexing, troubleshooting session controls, and full documentation"
```

# AGENTS.md — Developer & AI Agent Guidelines for LogPulse

Welcome to the **LogPulse** repository. This document defines the architectural principles, conventions, build requirements, and system contracts for AI agents and human contributors working on this codebase.

---

## 1. System Overview & Core Directives

LogPulse is a high-performance cross-platform desktop log discovery and diagnostic workbench built with **Go 1.22+**, **Wails v2**, and a **React 18 / TypeScript / Tailwind CSS** frontend.

### Hard Architectural Rules
1. **Never Emit 1 IPC Event Per Log Line:**
   - Wails IPC bridge performance collapses if flooded with single-line events.
   - All streaming updates must be dispatched via `tailer.Batcher` (default: 50ms flush or 100 lines threshold).
2. **Mandatory Build Tags for Go CLI Compilation:**
   - Plain `go build` produces a Wails stub executable that displays a build tags error.
   - Always compile with `-tags production` (or `-tags dev`).
   - Windows GUI binaries should include `-ldflags "-H windowsgui"` to suppress the background cmd console.
3. **Wails IPC Method Return Signatures:**
   - Methods bound to the JavaScript frontend in `app.go` must return either `(T)` or `(T, error)`.
   - Never return `(T1, T2)` without implementing `error` on `T2`. Wails silently discards the second non-error return value.
4. **Shared File Locks on Windows (`FILE_SHARE_DELETE`):**
   - Active logs are frequently locked, rolled, or deleted by running services.
   - Always open files with `windows.FILE_SHARE_READ | windows.FILE_SHARE_WRITE | windows.FILE_SHARE_DELETE` (see `internal/tailer/tailer_windows.go`).
5. **Mandatory Pre-Flight PII Redaction:**
   - No log text may be dispatched to external AI providers (OpenAI, Anthropic, Gemini) without first running through `internal/ai/scrubber.go`.
   - Monthly budget ceilings stored in SQLite (`monthly_budget_usd`) must be strictly honored before dispatching outbound HTTP calls.
6. **CGO-Free Persistence:**
   - Use `modernc.org/sqlite` with `PRAGMA journal_mode=WAL;`. Do not introduce dependencies requiring CGO (such as `mattn/go-sqlite3`).

---

## 2. Repository Layout & Package Boundaries

```
logpulse/
├── .github/
│   └── workflows/
│       └── release.yml          # Multi-platform matrix release (Windows, macOS, Linux)
├── build/                       # App packaging assets (appicon.png, binaries)
├── frontend/                    # Vite + React 18 + TypeScript + Tailwind CSS
│   ├── src/
│   │   ├── components/          # VirtualizedGrid, LogTree, ContextMenu, AIModal, SessionControl
│   │   ├── hooks/               # useLogStream (Wails event listeners)
│   │   ├── stores/              # Zustand state (useLogStore, useAIStore)
│   │   ├── workers/             # searchWorker.ts (off-thread regex filtering)
│   │   ├── types.ts             # Shared frontend data contracts
│   │   └── App.tsx              # Main workbench shell
│   ├── package.json
│   └── vite.config.ts
├── internal/
│   ├── ai/                      # Multi-provider client, PII scrubber, token heuristics
│   │   ├── client.go            # Ollama, OpenAI, Anthropic, Gemini dispatcher
│   │   ├── scrubber.go          # Regex redactor (IPv4/v6, JWT, API keys, emails)
│   │   └── tokenizer.go         # BPE estimation & USD cost calculation
│   ├── db/                      # Pure-Go SQLite persistence & WAL migrations
│   │   ├── db.go                # Store CRUD methods
│   │   └── models.go            # DiscoveredLog, AIAccountingEntry
│   ├── discovery/               # Tiered log crawler & path heuristics
│   │   ├── scanner.go           # Tier 1 instant & Tier 2 throttled background crawl
│   │   ├── paths_windows.go     # %LOCALAPPDATA%, %PROGRAMDATA%, winevt paths
│   │   └── paths_other.go       # /var/log, ~/Library/Logs
│   ├── events/                  # Windows Event Log (.evtx) parser
│   │   ├── events.go            # Common EventRecord normalization
│   │   ├── events_windows.go    # Win32 wevtapi.dll wrapper
│   │   └── events_fallback.go   # Cross-platform fallback / offline XML parser
│   └── tailer/                  # Live file tailer & stream multiplexer
│       ├── tailer.go            # Low-latency tailer & rotation reset
│       ├── batcher.go           # Micro-batch flush queue
│       ├── multiplexer.go       # Chronological K-way stream merge & session filtering
│       ├── tailer_windows.go    # Win32 shared read locks
│       └── tailer_other.go      # Non-Windows file open
├── app.go                       # Wails application struct & IPC methods
├── main.go                      # Entrypoint & asset embed server
├── wails.json                   # Wails project configuration
├── README.md                    # User & build documentation
└── AGENTS.md                    # This developer reference
```

---

## 3. Development Workflow & Commands

### Running Backend Unit Tests
Always run all unit tests before committing changes:
```bash
go test -v ./...
```

### Running Frontend Typecheck & Build
```bash
cd frontend
npm ci
npm run build
```

### Compiling Desktop Binary Manually
```bash
# Windows (suppress console window)
go build -tags production -ldflags "-H windowsgui" -o build/bin/logpulse.exe .

# macOS / Linux
go build -tags production -o build/bin/logpulse .
```

### Running in Live Development Mode (Hot Reload)
```bash
# Requires Wails CLI installed: go install github.com/wailsapp/wails/v2/cmd/wails@latest
wails dev
```

---

## 4. Frontend Virtualization & State Contracts

### High-Throughput DOM Virtualization (`@tanstack/react-virtual`)
- Log lines in `useLogStore` can grow up to 250,000 items in memory (`MAX_BUFFER = 250000`).
- The `searchWorker.ts` Web Worker handles all regex matching and level filtering off the main UI thread.
- Always debounce or throttle messages sent to `searchWorker.ts` (e.g. 80–100ms) to avoid cloning massive arrays repeatedly during rapid log bursts.

### Troubleshooting Session Multiplexing
- When the user starts a session (`isSessionActive: true`), dropping markers records a timeline pin with the line number.
- Selecting multiple logs with `Ctrl+Click` / `Cmd+Click` in `LogTree.tsx` adds paths to `activeTailedPaths`, streaming interleaved lines chronologically into the virtual grid.

---

## 5. Security & PII Redaction Rules

- All API keys (`api_key_openai`, `api_key_anthropic`, `api_key_gemini`) are stored exclusively in the local embedded SQLite database (`settings` table) on the user's workstation.
- When an AI investigation is initiated:
  1. `App.InvestigateWithAI` automatically runs `ai.ScrubPII` on the prompt.
  2. The frontend shows a side-by-side diff preview (`PIIScrubberDiff.tsx`).
  3. Spend is calculated and recorded in `ai_accounting`.
  4. If monthly spend exceeds `monthly_budget_usd`, requests are blocked immediately with a descriptive error.

---

## 6. GitHub Actions Matrix Release Workflow

The automated release workflow lives in [`.github/workflows/release.yml`](.github/workflows/release.yml).
- **Triggers:**
  - Pushing any tag matching `v*` (e.g. `v1.0.0`, `v1.0.1`).
  - Manual execution via `workflow_dispatch` in the GitHub Actions console.
- **Matrix Runners:**
  - `windows-latest` -> `logpulse-windows-amd64.zip`
  - `macos-latest` -> `logpulse-macos-universal.zip`
  - `ubuntu-22.04` -> `logpulse-linux-amd64.tar.gz` (installs `libwebkit2gtk-4.0-dev` and `libgtk-3-dev`)
- All assets are automatically attached to the corresponding GitHub Release.

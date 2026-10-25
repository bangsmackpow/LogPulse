# LogPulse

**LogPulse** is a high-performance, cross-platform desktop log discovery and diagnostic workbench built using Go and Wails v2 with a modern React, Vite, Tailwind CSS, and TanStack Virtual frontend.

It automatically catalogues log files across local storage, tails active files in real time with zero-copy buffer streaming, parses native Windows Event Logs (`.evtx`), and provides an AI Assist workspace with custom provider support (Ollama, OpenAI, Anthropic Claude, Google Gemini), context bracketing, pre-flight PII redaction, and deterministic token/cost tracking.

---

## Key Features

### 1. Intelligent Log Discovery & Scanner Service
- **Tier 1 (Instant Scan):** Instant crawl of well-known diagnostics directories on startup:
  - Windows: `%LOCALAPPDATA%`, `%APPDATA%`, `%PROGRAMDATA%`, `C:\Windows\Logs`, and `C:\Windows\System32\winevt\Logs\*.evtx`
  - Linux / macOS: `/var/log`, `~/Library/Logs`, and `~/.local/state`
- **Tier 2 (Background Recursive Crawl):** Low-priority worker pool walking local drives for `*.log`, `*.txt`, `*.trace`, `*.jsonl`, and `.out` files with automatic folder ignore (`node_modules`, `.git`, `temp`, `winsxs`) and I/O throttling.
- **Embedded Persistence:** Pure-Go SQLite (`modernc.org/sqlite`) with WAL mode enabled, indexing discovered logs, friendly application names, and format classifications without external database dependencies or CGO compilation hurdles.

### 2. High-Performance Tail & Grep Engine
- **Micro-Batch Live Tailing:** Emits streaming log updates via Wails IPC in batched micro-ticks (50ms interval or 100-line batches) to maintain 60 FPS and prevent IPC saturation.
- **File Rotation & Truncation Handling:** Seamlessly detects file truncation mid-tail (when size drops below offset) and rewinds offset to 0 without breaking watchers or dropping lines.
- **Virtualized Grid:** DOM virtualization using `@tanstack/react-virtual` capable of handling up to 250,000 in-memory lines with sticky line-number gutter and colorized log levels (`ERROR`, `WARN`, `INFO`, `DEBUG`).
- **Off-Thread Web Worker Search:** Regex filtering executed in a dedicated Web Worker to ensure zero UI frame drops during real-time queries.
- **Troubleshooting Sessions & Multi-Log Multiplexing:** Chronological K-way merge sorting across multiple active log streams and an interactive "Mark Session" tool to drop timeline pins and isolate logs before/after reproducing an incident.

### 3. AI Assist & Diagnostic Workbench
- **Highlight & Ask AI:** Right-click any log line or highlight a block to trigger "Investigate with AI".
- **Context Bracketing:** Context sliders to include $N$ lines before and $M$ lines after the selection.
- **Pre-Flight Local PII / Secret Scrubber:** Automatic local regex redactor for IPv4/IPv6 addresses, JWT tokens (`eyJ...`), API keys (`sk-...`, `AKIA...`, `AIza...`), emails, and passwords with an interactive diff preview showing masked secrets before dispatch.
- **Multi-Provider Client Interface:**
  - Local Ollama (`http://localhost:11434` / free)
  - OpenAI (GPT-4o, GPT-4o-mini)
  - Anthropic (Claude 3.5 Sonnet)
  - Google Gemini (Gemini 1.5/2.0)
- **Token & Cost Guardrails:** Real-time token heuristics and USD cost calculation displayed before dispatch, with hard session and monthly budget limits.

### 4. Native OS Integration
- Right-click context actions:
  - **Open in Default Editor:** Spawns system text editor / IDE (`cmd /c start`, `open`, or `xdg-open`).
  - **Reveal in File Explorer:** Opens parent directory with target file selected (`explorer /select,`, `open -R`).

---

## Architecture & Tech Stack

```
logpulse/
├── build/                     # Wails packaging assets
├── frontend/                  # React + TypeScript + Vite + Tailwind
│   ├── src/
│   │   ├── components/        # VirtualizedGrid, LogTree, ContextMenu, AIModal, SessionControl
│   │   ├── hooks/             # useLogStream
│   │   ├── stores/            # Zustand session & settings state
│   │   ├── workers/           # searchWorker.ts (off-thread regex search)
│   │   └── App.tsx
│   ├── package.json
│   └── vite.config.ts
├── internal/
│   ├── ai/                    # Multi-provider client, PII scrubber, tokenizer
│   ├── db/                    # Pure-Go SQLite migrations and repository queries
│   ├── discovery/             # Tiered disk scanner & path resolvers
│   ├── events/                # Windows .evtx parser and event wrappers
│   └── tailer/                # Live tailer, rotation handler, and stream multiplexer
├── app.go                     # Wails bound application logic / IPC bridge
├── main.go                    # Application entrypoint
├── wails.json                 # Wails project configuration
└── README.md
```

---

## Development & Testing

### Prerequisites
- **Go 1.22+**
- **Node.js 18+ & npm**
- **Wails CLI v2** (optional for dev running: `go install github.com/wailsapp/wails/v2/cmd/wails@latest`)

### Running Backend Unit Tests
```bash
go test -v ./...
```

### Running Frontend Typecheck & Build
```bash
cd frontend
npm install
npm run build
```

### Running in Development Mode
```bash
wails dev
```
Or run the Vite dev server directly:
```bash
cd frontend
npm run dev
```

### Building Desktop Binary

Using Wails CLI:
```bash
wails build
```

Or using standard Go CLI with required Wails production build tags:
```bash
# Windows (GUI mode without console window)
go build -tags production -ldflags "-H windowsgui" -o build/bin/logpulse.exe .

# macOS / Linux
go build -tags production -o build/bin/logpulse .
```

---

## License
MIT License

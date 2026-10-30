# LogPulse

**LogPulse** is a high-performance, cross-platform desktop log discovery and diagnostic workbench built using Go and Wails v2 with a modern React, Vite, Tailwind CSS, and TanStack Virtual frontend.

It automatically catalogues log files across local storage, tails active files in real time with zero-copy buffer streaming, parses native Windows Event Logs (`.evtx`), and provides an AI Assist workspace with custom provider support (OpenRouter.ai, DeepSeek, Groq, Ollama, OpenAI, Anthropic Claude, Google Gemini), context bracketing, pre-flight PII redaction, and deterministic token/cost tracking.

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
- **Chronological Inversion Toggle ("Newest at top"):** One-click checkbox switches between standard tailing (newest at bottom) and reverse chronological ordering (newest at top). Pinned auto-scroll respects the active orientation, and right-click AI diagnostics accurately retain underlying chronological indexes.
- **Quick Known Logs Selector & Multi-Stream Fusion:** Dropdown adjacent to the search bar provides instant 1-click access to Windows Event Logs (`System`, `Application`, `Security`) and detected application logs (Discord, Chrome, VS Code, Slack, Docker). When an application produces multiple logs, LogPulse fuses them into a single chronological stream with visual source pill badges (`main.log`, `renderer.log`).
- **File Rotation & Truncation Handling:** Seamlessly detects file truncation mid-tail (when size drops below offset) and rewinds offset to 0 without breaking watchers or dropping lines.
- **Virtualized Grid:** DOM virtualization using `@tanstack/react-virtual` capable of handling up to 250,000 in-memory lines with sticky line-number gutter and colorized log levels (`ERROR`, `WARN`, `INFO`, `DEBUG`).
- **Off-Thread Web Worker Search:** Regex filtering executed in a dedicated Web Worker to ensure zero UI frame drops during real-time queries.
- **Troubleshooting Sessions & Multi-Log Multiplexing:** Chronological K-way merge sorting across multiple active log streams and an interactive "Mark Session" tool to drop timeline pins and isolate logs before/after reproducing an incident.

### 3. AI Assist & Diagnostic Workbench
- **Highlight & Ask AI:** Right-click any log line or highlight a block to trigger "Investigate with AI".
- **Context Bracketing:** Context sliders to include $N$ lines before and $M$ lines after the selection.
- **Pre-Flight Local PII / Secret Scrubber:** Automatic local regex redactor for IPv4/IPv6 addresses, JWT tokens (`eyJ...`), API keys (`sk-...`, `AKIA...`, `AIza...`), emails, and passwords with an interactive diff preview showing masked secrets before dispatch. Scrubbing runs across prompt text, user notes, and telemetry fields.
- **Multi-Provider Client Interface:**
  - **OpenRouter.ai:** Unified access to hundreds of AI models (`claude-3.5-sonnet`, `deepseek-r1`, `gpt-4o`, `llama-3.3-70b`, `mistral-large`) with a single API key and arbitrary model ID overrides.
  - **DeepSeek:** Native API support for DeepSeek R1 reasoning and DeepSeek V3 chat.
  - **Groq:** Ultra-low latency inference for Llama 3.3 and Mixtral.
  - **Local Ollama:** Free, offline local inference (`http://localhost:11434`).
  - **OpenAI:** GPT-4o, GPT-4o-mini, and custom checkpoints.
  - **Anthropic Claude:** Claude 3.5 Sonnet and Haiku.
  - **Google Gemini:** Gemini 1.5 Pro, Flash, and 2.0.
- **Token & Cost Guardrails:** Real-time token heuristics and USD cost calculation displayed before dispatch, with hard session and monthly budget limits.

### 4. Native OS Integration
- Right-click context actions:
  - **Open in Default Editor:** Spawns system text editor / IDE (`cmd /c start`, `open`, or `xdg-open`).
  - **Reveal in File Explorer:** Opens parent directory with target file selected (`explorer /select,`, `open -R`).

---

## Architecture & Tech Stack

```
logpulse/
├── .github/
│   └── workflows/
│       └── release.yml        # Multi-platform matrix build
├── build/                     # Wails app icons & packaging assets
├── frontend/                  # React + TypeScript + Vite + Tailwind
│   ├── src/
│   │   ├── components/        # VirtualizedGrid, QuickLogSelector, LogTree, ContextMenu, AIModal, SessionControl
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
├── AGENTS.md                  # Developer & AI Agent Guidelines
└── README.md
```

---

## Multi-Platform Releases (GitHub Actions)

LogPulse features an automated matrix release workflow via [`.github/workflows/release.yml`](.github/workflows/release.yml). All release binaries can be downloaded directly from the [GitHub Releases page](https://github.com/bangsmackpow/LogPulse/releases):

| Target Platform | Runner | Build Dependencies | Packaged Release Asset |
| :--- | :--- | :--- | :--- |
| **Windows x64 (Installer)** | `windows-latest` | Go 1.23, Node 20, Wails CLI, Inno Setup | `logpulse-windows-amd64-setup.exe` (with Desktop & Start Menu shortcuts) |
| **Windows x64 (Portable)** | `windows-latest` | Go 1.23, Node 20, Wails CLI | `logpulse-windows-amd64.zip` |
| **macOS Universal** | `macos-latest` | Go 1.23, Node 20, Wails CLI | `logpulse-macos-universal.zip` (`.app` bundle) |
| **Linux x64** | `ubuntu-22.04` | `libgtk-3-dev`, `libwebkit2gtk-4.0-dev` | `logpulse-linux-amd64.tar.gz` |

To trigger an automated release across all platforms, push a semantic version tag:
```bash
git tag v1.0.2
git push origin v1.0.2
```
Or run the workflow manually under the **Actions** tab on GitHub via `workflow_dispatch`.

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

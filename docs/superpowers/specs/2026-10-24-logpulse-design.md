# LogPulse Design Document

**Date:** 2026-10-24  
**Status:** In Review  
**Target Platforms:** Windows 10/11 (Tier 1 First-Class), macOS (Universal), Linux (x86_64 AppImage/deb)

---

## 1. Intent & The "Why"
When diagnosing or troubleshooting desktop applications—especially legacy, third-party, or unfamiliar internal tools—engineers and support specialists frequently encounter silent failures. The user interface either fails silently, throws generic alerts ("An error occurred"), or leaves no obvious clue in the UI. The actual failure reasons (unhandled exceptions, missing DLLs, network timeouts, configuration parse failures) are logged to local files or Windows Event Log channels.

Locating these logs manually is tedious: logs are scattered across `%LOCALAPPDATA%`, `%APPDATA%`, `%PROGRAMDATA%`, Windows Event Log, and application directories. Furthermore, once located, log files can be massive, noisy, and contain cryptic errors.

**LogPulse** solves this by:
1. **Instantly cataloging and indexing** logs across standard and user-defined locations.
2. **Streaming live log updates** using a high-throughput, low-latency zero-copy virtualized tailer that can handle 250,000+ lines without UI stutter.
3. **Providing an AI-powered diagnostic bridge** with context bracketing, local PII/secret scrubbing, cost estimation, and multi-provider support (Anthropic, OpenAI, Gemini, and local Ollama) to translate unfamiliar logs into immediate, actionable solutions.

---

## 2. Goals, Non-Goals, & Anti-Goals

### Goals
- **Instant Discovery:** Crawl well-known Windows diagnostics directories in under 2 seconds on launch, followed by an asynchronous background recursive scan for common log file patterns (`*.log`, `*.txt`, `*.trace`, `*.jsonl`, `.out`).
- **High-Performance Tail Engine:** Batch Wails IPC streaming events (e.g. 50ms micro-ticks or 100-line batches) to maintain 60 FPS in React using `@tanstack/react-virtual`.
- **Windows Event Log Support:** Native reading of `.evtx` channels/files via `wevtapi.dll` using Go windows syscalls.
- **Troubleshooting Sessions:** Merged multi-log chronological view with a "Mark Session" capability to isolate logs generated during a specific reproduction step.
- **Air-Gapped / Privacy-First AI Assist:** Local regex-based PII and secret redaction (IPs, JWTs, API keys, emails) *before* any outbound HTTP request, with preview and token/cost estimation.
- **CGO-Free Embedded Persistence:** Pure-Go SQLite (`modernc.org/sqlite`) with WAL mode to ensure cross-compilation and easy distribution.

### Non-Goals
- **Enterprise Distributed APM:** LogPulse is not a replacement for Datadog, Elastic/Logstash, or Splunk; it is a desktop diagnostics workbench for local machines and reproduction environments.
- **Log Modification / Editing:** LogPulse is strictly a discovery, tail, search, and diagnostic viewer. It does not mutate or rewrite log files on disk.
- **Cloud-Hosted Log Aggregator:** No external backend servers; all indexing, settings, and credentials live locally on the user's workstation.

### Anti-Goals
- **IPC Flooding / UI Freezes:** Never dispatch one IPC event per log line. Flooding the Wails bridge degrades the entire webview process.
- **Leaking Secrets to External APIs:** Never send raw logs to AI providers without passing through the local scrubber and user pre-flight verification modal.
- **Unbounded Memory Consumption:** The virtualized view and backend ring buffers must enforce memory caps to prevent OOM when tailing multi-gigabyte files.

---

## 3. Constraints & Technical Stack

- **Backend:** Go 1.22+
  - **Desktop Framework:** Wails v2 (stable desktop lifecycle, webview bindings, and asset bundling)
  - **Database:** `modernc.org/sqlite` (pure Go, CGO-free, WAL enabled)
  - **File Watcher:** `github.com/fsnotify/fsnotify` + periodic polling fallback for rotated/truncated files
  - **Windows Event Log:** Windows API bindings via `golang.org/x/sys/windows` accessing `wevtapi.dll`
- **Frontend:**
  - **Runtime & Tooling:** Node 18+, Vite, React 18+, TypeScript
  - **Styling & UI:** Tailwind CSS, Lucide Icons, Radix UI primitives where applicable
  - **Virtualization:** `@tanstack/react-virtual`
  - **State Management:** Zustand
- **AI Integrations:**
  - Standard HTTP REST clients for Anthropic Claude, OpenAI (GPT-4o/mini), Google Gemini API, and local OpenAI-compatible endpoints (Ollama / LocalAI / LM Studio).
  - Local BPE / heuristic token estimation and USD pricing calculations.

---

## 4. Architecture & Module Breakdown

### 4.1. Architecture Diagram
```
+-------------------------------------------------------------------------+
|                              LogPulse Frontend                          |
|  (Vite + React 18 + TypeScript + Tailwind CSS + TanStack Virtual + Zustand)|
|                                                                         |
|  [Log Catalog Sidebar]   [Virtual Log Stream Gutter]   [AI Assist Drawer]|
+------------------------------------+------------------------------------+
                                     | Wails v2 IPC (Events & Methods)
+------------------------------------v------------------------------------+
|                               Go Backend (app.go)                       |
+------------------+------------------+-----------------+-----------------+
| Discovery Engine |  Tailer Engine   | Win Event Log   | AI & Tokenizer  |
| (Tier 1 & Tier 2)| (fsnotify + ring)|  (wevtapi.dll)  | (Scrubber + API)|
+------------------+------------------+-----------------+-----------------+
                                     |
+------------------------------------v------------------------------------+
|               Embedded SQLite Store (modernc.org/sqlite)                |
|        - Discovered Paths   - Search History   - AI Token Accounting    |
+-------------------------------------------------------------------------+
```

### 4.2. Internal Packages

1. **`internal/discovery` (Scanner Service)**
   - **Tier 1 (Instant):** Checks predefined environment paths:
     - `%LOCALAPPDATA%\**\*.log`
     - `%APPDATA%\**\*.log`
     - `%PROGRAMDATA%\**\*.log`
     - `C:\Windows\Logs\**\*`
     - `C:\Windows\System32\winevt\Logs\*.evtx`
     - Cross-platform equivalents (`/var/log`, `~/Library/Logs`).
   - **Tier 2 (Recursive Worker Pool):** Low-priority background crawler with configurable depth and directory exclusion (`node_modules`, `.git`, `Windows\WinSxS`, etc.).
   - Emits batch discovery events to the frontend and updates the SQLite catalog.

2. **`internal/tailer` (Live Log Streamer)**
   - Uses `fsnotify` with offset tracking.
   - Handles file rotation and truncation: if file size decreases, reset offset to zero; if file moves/renames, re-attach watcher.
   - Micro-batch dispatcher: flushes buffered lines every 50ms or every 100 lines, whichever comes first.
   - Ring-buffer memory management per open log file.

3. **`internal/events` (Windows Event Log Reader)**
   - Wraps `wevtapi.dll` using `golang.org/x/sys/windows`.
   - Supports querying active channels (System, Application, Security) or opening standalone `.evtx` files.
   - Normalizes event records into common `LogEntry` structure (timestamp, level, source, message, eventID).

4. **`internal/ai` (Multi-Provider AI & Pre-Flight Scrubber)**
   - **PII / Secret Scrubber:** Fast compiled regex matcher for:
     - IPv4 / IPv6 addresses
     - Bearer tokens / JWTs (`eyJ...`)
     - API Keys (AWS, OpenAI, Generic high-entropy hex/base64 patterns)
     - Email addresses
     - Connection strings / Passwords
   - **Tokenizer & Cost Estimator:** Estimates tokens per prompt + context window and computes cost based on current provider rates.
   - **Client Dispatcher:** Clean HTTP client for Anthropic, OpenAI, Gemini, and Ollama.
   - **Budget Safeguard:** Checks session and monthly accumulated spend against user-defined budget caps before dispatching.

5. **`internal/db` (Embedded SQLite)**
   - Pure Go SQLite without CGO requirements.
   - Schema tables:
     - `discovered_logs`: id, path, app_name, size, modified_at, format, is_favorite, tag
     - `ai_accounting`: id, timestamp, provider, model, input_tokens, output_tokens, cost_usd, query_summary
     - `settings`: key, value
   - WAL mode enabled for concurrent read/write stability.

6. **`frontend/src` (Web UI Workbench)**
   - **`components/LogTree`**: Hierarchical or searchable catalog list grouped by application or folder, with filter tags.
   - **`components/VirtualizedGrid`**: Windowed log viewer using `@tanstack/react-virtual` capable of handling 250,000+ lines.
     - Sticky line numbers, color-coded log levels (ERROR red, WARN amber, INFO blue, DEBUG gray).
     - Live regex filter and search matches highlight.
   - **`components/TroubleshootingSession`**: "Mark Session" button to drop visual timeline pins and filter logs before/after the incident.
   - **`components/AIModal`**: Context slider ($N$ lines before, $M$ lines after), user notes textarea, telemetry toggle, PII scrubbing diff toggle, token/cost estimation badge, and interactive "Send to AI" action.
   - **`stores/`**: Zustand state for active logs, filters, AI configuration, and current session markers.

---

### 4.3. Large-File Ingestion & External Editor Integration
- **Immediate Tail Focus:** When opening an existing or large log file, LogPulse immediately reads the recent tail (last 10,000–50,000 lines up to the 250,000 in-memory virtual buffer limit) to ensure zero launch delay and instant troubleshooting responsiveness.
- **Right-Click Context Menu:**
  - In both the log catalog tree and within the virtual log stream, right-clicking offers:
    - **"Open File in Default Editor"** (spawns OS default editor/IDE, e.g. VS Code, Notepad++, or text editor).
    - **"Reveal in File Explorer / Finder"** (opens the containing directory).
    - **"Investigate with AI"** (for selected lines).
    - **"Copy Line / Selection"** and **"Copy File Path"**.

## 5. Security & Safety

- **Local Storage of Credentials:** API keys are stored in encrypted or secure local config/database entries, never broadcast via telemetry.
- **Redaction by Default:** The PII scrubber runs automatically with a clear preview showing what has been masked (e.g. `[REDACTED_IP]`, `[REDACTED_TOKEN]`).
- **Deterministic Cost Guardrails:** Users can configure a hard monthly limit (e.g., $5.00) and per-query limit. If exceeded, dispatch is blocked with a clear override prompt.

---

## 6. Architecture Decisions & Builder Calls

The following decisions were reviewed during the builder check and settled as sensible defaults for performance, maintainability, and cross-platform compatibility:

1. **Timestamp Parsing & Multiline Log Merging (Chronological Stream):**
   - Heuristic timestamp extractor supporting standard formats (ISO8601, RFC3339, common syslog `MMM d HH:mm:ss`, and standard Windows Event Log timestamps).
   - If a line has no timestamp (e.g. stack traces, continuation lines, `Caused by:`), it is automatically grouped with the preceding timestamped entry.
   - For unstructured logs without detectable timestamps, entries fall back to file write/arrival time.

2. **Search and Regex Filtering Execution:**
   - **Hybrid Search Architecture:**
     - In-memory buffered lines (up to 250,000) are filtered in real-time on the frontend using a dedicated Web Worker off the main UI thread to ensure zero UI frame drops.
     - For searching through unbuffered historical disk files, a backend Go worker scans the file using memory-mapped buffers and returns match coordinates.

3. **Cross-Platform Parity for Windows Event Logs (`.evtx`):**
   - Windows 10/11 is the primary first-class target. Active channels and live event subscriptions use native `wevtapi.dll` via Windows syscalls.
   - On macOS and Linux, the live Windows Event Log channels are gracefully disabled, and system log discovery crawls standard paths (`/var/log` on Linux, `~/Library/Logs` on macOS). A pure-Go offline `.evtx` parser is provided for opening exported `.evtx` files across all platforms.

4. **Default Context Slider Range:**
   - Default: 15 lines before, 15 lines after (configurable up to 100 lines).

5. **Discovery Crawl Throttling:**
   - Tier 2 background crawl sleeps 5ms between folders and yields to keep disk I/O under 5% and CPU minimal.

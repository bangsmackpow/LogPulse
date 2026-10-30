# Design Document: Log Display Ordering, Quick Known Logs Selector, and Expanded AI Providers

- **Date:** 2026-10-29
- **Status:** Final / Approved for Planning
- **Target:** LogPulse Desktop Workbench (Go + Wails v2 + React 18 / TypeScript / Tailwind CSS)

---

## 1. Intent & Problem Statement

Users troubleshooting live issues need faster pathways from launching LogPulse to diagnosing the root cause. This involves:
1. **Reverse Chronological Tailing:** Traditional log tails append to the bottom. In fast-moving logs or incident triage, users frequently prefer reading from top-to-bottom with the newest entries immediately visible at the top.
2. **Instant Navigation via Quick Log Dropdown:** Rather than manually expanding deep filesystem trees to find Windows Event Logs or application-specific directories (such as Discord or Chrome), users need a 1-click preset selector adjacent to the search bar.
3. **Multi-File Stream Fusion (Option C):** When an application writes across multiple files (e.g., Discord's `main.log`, `renderer.log`, and `crashpad.log`), selecting the app immediately streams all of those files together into a unified, chronologically merged feed with source file tags.
4. **Universal AI Access via OpenRouter & Extended Providers:** Users shouldn't be limited to only three proprietary AI providers. Supporting **OpenRouter.ai** allows users with a single API key to query 100+ models (DeepSeek-R1, DeepSeek-V3, Claude 3.5 Sonnet, GPT-4o, Llama 3.3, Mistral), while also supporting direct keys for DeepSeek, Groq, and local Ollama.

---

## 2. Goals, Non-Goals, and Anti-Goals

### Goals
- **G1:** Add a persistent "Newest at top" toggle in the top global navigation bar next to level filter buttons (Option B).
- **G2:** Seamlessly support top-down streaming: when "Newest at top" is enabled, incoming tail updates prepend to the top of the virtual grid without breaking scroll stability or line index targeting for AI context menus.
- **G3:** Add a categorized dropdown near the search bar for Windows Event Logs (`System`, `Application`, `Security`) and common applications (`Discord`, `Google Chrome`, `Visual Studio Code`, `Slack`, `Docker Desktop`).
- **G4:** When selecting a multi-log application from the dropdown, activate all discovered log files for that app in LogPulse's stream multiplexer, stopping previous tails and rendering source badges for each file.
- **G5:** Add OpenRouter.ai as a first-class AI provider, alongside DeepSeek, Groq, OpenAI, Anthropic, Gemini, and Ollama.
- **G6:** Allow users to choose popular model presets or type any custom model ID string (e.g., `deepseek/deepseek-r1`, `anthropic/claude-3.5-sonnet`).

### Non-Goals
- **NG1:** Replacing the left sidebar log tree catalog (the dropdown is a fast jump shortcut).
- **NG2:** Custom proprietary binary parsers for every application (plain text files use timestamp heuristics; Windows `.evtx` files continue to use the native Win32 EVTX parser).
- **NG3:** Storing API keys anywhere outside the local embedded SQLite `settings` table.

### Anti-Goals
- **AG1:** Never break the 60 FPS rendering performance of `@tanstack/react-virtual` when "Newest at top" is active.
- **AG2:** Never mutate `LogLine.raw` when tagging multi-file sources; always use visual UI pill badges to protect search, regex, and AI PII scrubbing accuracy.

---

## 3. Architecture & Detailed Specifications

### 3.1 Feature 1: "Newest at Top" Log Display Order

#### Placement & UI
- Placed in `frontend/src/App.tsx` top navigation header, to the right of the level filters (`ERROR`, `WARN`, `INFO`, `DEBUG`).
- Styled as a compact pill checkbox:
  ```tsx
  <label className="flex items-center gap-1.5 px-2.5 py-1 rounded bg-canvas-900 border border-canvas-700 text-xs text-gray-300 cursor-pointer hover:border-canvas-600 transition-colors">
    <input 
      type="checkbox" 
      checked={isNewestOnTop} 
      onChange={(e) => setNewestOnTop(e.target.checked)} 
      className="accent-indigo-500 rounded cursor-pointer" 
    />
    <span className="font-mono text-[11px] select-none">Newest at top</span>
  </label>
  ```

#### State & Virtualization Contract
- State in `useLogStore`: `isNewestOnTop: boolean` (persisted in localStorage or settings, defaults to `false`).
- In `VirtualizedGrid.tsx`:
  - When `isNewestOnTop` is `false`:
    - Row `index` maps directly to `lines[index]`.
    - New incoming lines scroll to bottom (`scrollToBottom()`) if user was near the bottom.
  - When `isNewestOnTop` is `true`:
    - Row `index` maps to `lines[lines.length - 1 - index]`.
    - Line number gutter displays the line's actual `line.lineNumber`.
    - When new lines arrive, if the user is at the top (`scrollTop < 30`), scroll stays pinned to top (`scrollToOffset(0)`).
  - Context menu and AI investigation:
    - Context actions receive the underlying line object and its true chronological index in `lines`, guaranteeing context bracketing (before/after $N$ lines) correctly retrieves preceding and succeeding chronological context.

---

### 3.2 Feature 2: Quick Known Logs Dropdown & Multi-File Interleaving

#### Placement & UI
- Placed immediately to the right of the regex search input in `frontend/src/App.tsx`.
- Categorized dropdown showing available logs found on the user's system:
  ```
  ⚡ Quick Jump to Known Log...
  ├── ── Windows Event Logs ──
  │   ├── Windows System (System.evtx)
  │   ├── Windows Application (Application.evtx)
  │   └── Windows Security (Security.evtx)
  └── ── Applications ──
      ├── Discord (3 files)
      ├── Google Chrome (2 files)
      ├── Visual Studio Code (4 files)
      ├── Docker Desktop (1 file)
      └── Slack (1 file)
  ```

#### Navigation & Multiplexing Lifecycle
- **Navigation Jump (Replacement):**
  - Selecting an item from the dropdown replaces the current active view.
  - Calls `App.StopTail(path)` for previously active paths to release OS file locks and stop background goroutines.
  - Clears `lines: []` in `useLogStore`.
- **Single Log File (e.g. Windows System):**
  - Sets `selectedLog: log`, `activeTailedPaths: new Set([log.filePath])`.
  - Invokes `App.StartTail(log.filePath)`.
- **Multi-Log Application (Option C - Interleaved Multi-Tail):**
  - If multiple logs match an application (e.g. Discord with `main.log`, `renderer.log`, `crashpad.log`):
    - Sets `activeTailedPaths: new Set(allMatchingPaths)`.
    - Subheader displays `Discord (3 files)` with a multi-stream badge.
    - Calls `App.StartTail` for each path.
    - Incoming lines from all files are merged chronologically in `useLogStore.appendLines` using `timestamp`.
    - In `VirtualizedGrid`, when `activeTailedPaths.size > 1`, a subtle pill badge is rendered next to the timestamp showing the source file basename (e.g. `main.log`, `renderer.log`), preserving `LogLine.raw` intact.

---

### 3.3 Feature 3: Expanded AI Providers & OpenRouter.ai

#### Supported Providers
1. **OpenRouter.ai** (`openrouter`):
   - Base URL: `https://openrouter.ai/api/v1`
   - Headers: `Authorization: Bearer <key>`, `HTTP-Referer: https://github.com/bangsmackpow/LogPulse`, `X-Title: LogPulse`
   - Presets: `deepseek/deepseek-r1`, `deepseek/deepseek-chat`, `anthropic/claude-3.5-sonnet`, `openai/gpt-4o`, `meta-llama/llama-3.3-70b-instruct`, `mistralai/mistral-large-2411`.
   - Allows typing any custom model ID string.
2. **DeepSeek** (`deepseek`):
   - Base URL: `https://api.deepseek.com/v1`
   - Presets: `deepseek-reasoner` (R1), `deepseek-chat` (V3).
3. **Groq** (`groq`):
   - Base URL: `https://api.groq.com/openai/v1`
   - Presets: `llama-3.3-70b-versatile`, `mixtral-8x7b-32768`.
4. **Existing Direct Providers:** OpenAI, Anthropic, Gemini, Ollama.

#### Backend Dispatcher (`internal/ai/client.go`)
- OpenRouter, DeepSeek, and Groq share standard OpenAI-compatible `/chat/completions` JSON schemas:
  `{"model": "...", "messages": [{"role": "system", "content": "..."}, {"role": "user", "content": "..."}]}`.
- Reuses the existing HTTP client dispatch routine with custom endpoints and authorization headers.
- All outbound prompts run through `ai.ScrubPII(prompt)` prior to dispatch.
- Spend tracking records token usage and estimated cost into SQLite `ai_accounting`.

#### Settings & Storage
- SQLite `settings` table schema supports new keys:
  - `api_key_openrouter`
  - `api_key_deepseek`
  - `api_key_groq`
  - `default_ai_provider`
  - `default_ai_model`
- `SettingsModal.tsx` updated with input fields for each key and provider selection.
- `AIModal.tsx` updated with provider selector tabs and model dropdowns including OpenRouter.

---

## 4. Verification & Testing Plan
- **Backend Tests:**
  - `go test -v ./internal/ai/...` (verify client dispatch for OpenRouter, DeepSeek, Groq, OpenAI, Anthropic, Gemini, Ollama).
  - Test PII scrubber on sensitive logs.
- **Frontend Typecheck & Build:**
  - `npm run build` in `frontend/` succeeds without type errors.
- **Runtime Verification:**
  - Toggle "Newest at top": verify reverse order renders smoothly, line numbers and context menu work.
  - Dropdown select Windows Event Logs: verify instant stream.
  - Dropdown select Discord: verify all Discord log files stream simultaneously with multiplexing.
  - Run AI investigation test with OpenRouter / configured provider.

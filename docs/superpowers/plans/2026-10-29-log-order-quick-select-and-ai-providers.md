# Log Display Ordering, Quick Known Logs Selector, and Expanded AI Providers Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement "Newest at top" log ordering, a Quick Known Logs dropdown with multi-file stream fusion, and universal AI provider support (OpenRouter.ai, DeepSeek, Groq) with custom model selection.

**Architecture:** 
- In the Go backend, extend `internal/ai/client.go` to handle OpenRouter, DeepSeek, and Groq via OpenAI-compatible endpoints with dedicated headers, while keeping pre-flight PII scrubbing and budget limits intact.
- In the frontend Zustand store and virtualization layer, add top-down reverse row mapping with pinned auto-scroll to index 0, and dynamic file pill badges when multiplexing multiple logs.
- Add a modular `QuickLogSelector` component near the search bar that catalogs Windows Event Logs and installed application logs, replacing previous active tails on navigation and fusing multiple app logs into one chronological stream.

**Tech Stack:** Go 1.22+, Wails v2, React 18, TypeScript, Tailwind CSS, `@tanstack/react-virtual`, Zustand.

**Spec:** [`docs/superpowers/specs/2026-10-29-log-order-quick-select-and-ai-providers-design.md`](file:///C:/Users/curtis/Desktop/dev/logpulse/docs/superpowers/specs/2026-10-29-log-order-quick-select-and-ai-providers-design.md)

## Global Constraints

- Never emit 1 IPC event per log line; streaming updates must dispatch via `tailer.Batcher`.
- Always open files with shared locks `FILE_SHARE_READ | FILE_SHARE_WRITE | FILE_SHARE_DELETE`.
- Wails IPC methods in `app.go` must return `(T)` or `(T, error)`.
- No log text may be dispatched to external AI providers without first running through `ai.ScrubPII`.
- Monthly budget ceilings stored in SQLite (`monthly_budget_usd`) must be strictly honored before dispatching outbound HTTP calls.
- Pure-Go SQLite persistence (`modernc.org/sqlite`); do not introduce CGO.

## Review Focus

1. **Reverse Chronological AI Context Bracketing:** When "Newest at top" is enabled and the user right-clicks row #0 (the newest line), the context menu must pass the line's underlying index in `lines` (i.e. `lines.length - 1`), so before/after $N$ lines grabs earlier lines rather than crashing out of bounds.
2. **Scroll Jumping on Fast Ingestion:** When "Newest at top" is enabled, incoming batches must not snap the user to the top if they have scrolled down to inspect an older line.
3. **Multi-File Tail Cleanup:** When navigating to a new log via Quick Jump, previous file watchers must be stopped via `App.StopTail` so open file handles on disk are closed.
4. **OpenRouter Authorization & Referer Headers:** Outbound calls to `https://openrouter.ai/api/v1` must include `Authorization: Bearer <key>`, `HTTP-Referer: https://github.com/bangsmackpow/LogPulse`, and `X-Title: LogPulse`.
5. **Multi-File Badge Overflow:** When multiplexing logs with long filenames, the file pill badge in `VirtualizedGrid` must not clip line text or break grid column widths.

---

### Task 1: Backend AI Client Expansion (OpenRouter, DeepSeek, Groq)

**Files:**
- Modify: `internal/ai/client.go:20-180`
- Modify: `internal/ai/tokenizer.go:40-90`
- Test: `internal/ai/client_test.go`

**Interfaces:**
- Consumes: `ai.MultiProviderClient`, `ai.ScrubPII`, `db.Store`
- Produces: `client.Investigate(ctx, provider, model, prompt)` supporting providers: `"openrouter"`, `"deepseek"`, `"groq"`, `"openai"`, `"anthropic"`, `"gemini"`, `"ollama"`.

- [ ] **Step 1: Write failing unit tests for new AI providers**
  In `internal/ai/client_test.go`, add unit tests verifying:
  - OpenRouter request formatting, base URL `https://openrouter.ai/api/v1`, and custom headers (`HTTP-Referer`, `X-Title`).
  - DeepSeek base URL `https://api.deepseek.com/v1` and endpoint routing.
  - Groq base URL `https://api.groq.com/openai/v1`.
  - PII scrubbing runs before dispatch on all three providers.

- [ ] **Step 2: Run test to verify it fails**
  Run: `go test -v ./internal/ai -run "TestOpenRouter|TestDeepSeek|TestGroq"`
  Expected: FAIL with unsupported provider error.

- [ ] **Step 3: Implement OpenRouter, DeepSeek, and Groq dispatcher in `internal/ai/client.go` and tokenizer in `internal/ai/tokenizer.go`**
  - In `client.go`, map provider base URLs and authorization headers.
  - In `tokenizer.go`, add cost estimation rates for OpenRouter presets (e.g. `$0.003/1k` input, `$0.015/1k` output for Claude 3.5 Sonnet, `$0.0005/1k` for DeepSeek R1).
  - Update `EstimateCost(provider, model, prompt)` to handle new provider strings.

- [ ] **Step 4: Run test to verify it passes**
  Run: `go test -v ./internal/ai/...`
  Expected: PASS

- [ ] **Step 5: Commit**
  ```bash
  git add internal/ai/
  git commit -m "feat(ai): add OpenRouter, DeepSeek, and Groq dispatchers with token pricing"
  ```

---

### Task 2: Frontend State & "Newest at Top" Log Order Toggle

**Files:**
- Modify: `frontend/src/stores/useLogStore.ts:5-65`
- Modify: `frontend/src/App.tsx:85-155`
- Modify: `frontend/src/components/VirtualizedGrid.tsx:30-180`
- Test: `frontend/src/components/VirtualizedGrid.test.tsx` (or verify via `npm run build`)

**Interfaces:**
- Consumes: `useLogStore`, `VirtualizedGrid`
- Produces: `useLogStore.isNewestOnTop`, `useLogStore.setNewestOnTop(val: boolean)`

- [ ] **Step 1: Add `isNewestOnTop` to `useLogStore.ts`**
  Add state and action to `useLogStore`:
  ```ts
  isNewestOnTop: boolean;
  setNewestOnTop: (val: boolean) => void;
  ```
  Default `isNewestOnTop: false`. Persist in `localStorage.getItem('logpulse_newest_on_top')`.

- [ ] **Step 2: Add top-bar toggle in `frontend/src/App.tsx`**
  To the right of the level filters (`ERROR`, `WARN`, `INFO`, `DEBUG`), add:
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

- [ ] **Step 3: Update row mapping and auto-scroll in `VirtualizedGrid.tsx`**
  - Read `isNewestOnTop` from `useLogStore`.
  - In `rowVirtualizer`, when `isNewestOnTop` is true:
    - Map virtual item index `virtualRow.index` to underlying index `rawIndex = lines.length - 1 - virtualRow.index`.
    - Line data: `const line = lines[rawIndex]`.
    - Context actions pass `rawIndex` so right-click AI investigation receives the exact underlying chronological index.
  - Auto-scroll logic:
    - When `isNewestOnTop` is false: if scrolled near bottom, `scrollToOffset(totalHeight)` on new lines.
    - When `isNewestOnTop` is true: if scrolled near top (`scrollTop < 40`), `scrollToOffset(0)` on new lines.

- [ ] **Step 4: Verify typecheck & build**
  Run: `npm --prefix frontend run build`
  Expected: Clean build without TypeScript errors.

- [ ] **Step 5: Commit**
  ```bash
  git add frontend/src/stores/useLogStore.ts frontend/src/App.tsx frontend/src/components/VirtualizedGrid.tsx
  git commit -m "feat(ui): add Newest at top order toggle with pinned top-down virtualization"
  ```

---

### Task 3: Quick Known Logs Dropdown & Multi-File Stream Fusion

**Files:**
- Create: `frontend/src/components/QuickLogSelector.tsx`
- Modify: `frontend/src/stores/useLogStore.ts:40-75`
- Modify: `frontend/src/hooks/useLogStream.ts:70-82`
- Modify: `frontend/src/App.tsx:95-115`
- Modify: `frontend/src/components/VirtualizedGrid.tsx:80-140`

**Interfaces:**
- Consumes: `useLogStore.logs`, `useLogStore.selectLog`, `useLogStore.activeTailedPaths`, `window.go.main.App.StopTail`
- Produces: `QuickLogSelector` dropdown component, multi-file source pill rendering in `VirtualizedGrid`.

- [ ] **Step 1: Implement tail replacement action in `useLogStore.ts`**
  Add store action `switchActiveTails(newPaths: string[], groupLabel?: string)`:
  - For any path currently in `activeTailedPaths` that is NOT in `newPaths`, invoke `window.go?.main?.App?.StopTail(path)`.
  - Set `activeTailedPaths = new Set(newPaths)`.
  - Set `lines: []`.
  - If `groupLabel` is provided, create a synthetic `selectedLog` representing the multi-file group (e.g. `{ filePath: groupLabel, appName: groupLabel }`).

- [ ] **Step 2: Create `QuickLogSelector.tsx`**
  Build component with categorized presets:
  - Windows Event Logs:
    - Windows System (`*System.evtx`)
    - Windows Application (`*Application.evtx`)
    - Windows Security (`*Security.evtx`)
  - Discovered Applications (Discord, Chrome, VS Code, Slack, Docker):
    - Matches discovered logs in `logs` by path/appName.
    - If 1 file: selects directly.
    - If multiple files (Option C): calls `switchActiveTails(allMatchingPaths, `${app} (${count} files)`)`, triggering simultaneous stream multiplexing.

- [ ] **Step 3: Integrate `QuickLogSelector` in `App.tsx`**
  Mount `QuickLogSelector` right next to the regex search input in the header bar.

- [ ] **Step 4: Add file pill badge in `VirtualizedGrid.tsx`**
  When `activeTailedPaths.size > 1`:
  - Extract basename of `line.fileId` (e.g. `main.log`, `renderer.log`, `System.evtx`).
  - Render a compact colored badge next to the timestamp:
    ```tsx
    <span className="rounded bg-canvas-700/80 px-1.5 py-0.5 text-[10px] font-mono text-indigo-300 border border-canvas-600/60 shrink-0">
      {basename(line.fileId)}
    </span>
    ```

- [ ] **Step 5: Verify typecheck & build**
  Run: `npm --prefix frontend run build`
  Expected: Clean build without errors.

- [ ] **Step 6: Commit**
  ```bash
  git add frontend/src/components/QuickLogSelector.tsx frontend/src/stores/useLogStore.ts frontend/src/hooks/useLogStream.ts frontend/src/App.tsx frontend/src/components/VirtualizedGrid.tsx
  git commit -m "feat(ui): add QuickLogSelector with multi-file interleaved streaming"
  ```

---

### Task 4: Extended AI Provider Settings & Investigation UI

**Files:**
- Modify: `frontend/src/types.ts:40-70`
- Modify: `frontend/src/components/SettingsModal.tsx:40-150`
- Modify: `frontend/src/components/AIModal.tsx:50-200`
- Modify: `app.go:210-260`

**Interfaces:**
- Consumes: `window.go.main.App.GetSettings`, `window.go.main.App.SaveSettings`, `window.go.main.App.InvestigateWithAI`
- Produces: OpenRouter, DeepSeek, and Groq configuration UI, custom model ID support.

- [ ] **Step 1: Update frontend types in `frontend/src/types.ts`**
  Update `AIProvider` type:
  ```ts
  export type AIProvider = 'ollama' | 'openai' | 'anthropic' | 'gemini' | 'openrouter' | 'deepseek' | 'groq';
  ```

- [ ] **Step 2: Update `SettingsModal.tsx`**
  - Add API key input fields for:
    - OpenRouter API Key (`api_key_openrouter`)
    - DeepSeek API Key (`api_key_deepseek`)
    - Groq API Key (`api_key_groq`)
  - Add OpenRouter, DeepSeek, and Groq to the default provider dropdown.

- [ ] **Step 3: Update `AIModal.tsx` with OpenRouter & Custom Model Support**
  - Add tabs or select dropdown for OpenRouter, DeepSeek, and Groq.
  - When OpenRouter is selected:
    - Display preset dropdown: `anthropic/claude-3.5-sonnet`, `openai/gpt-4o`, `deepseek/deepseek-r1`, `deepseek/deepseek-chat`, `meta-llama/llama-3.3-70b-instruct`, `mistralai/mistral-large-2411`, `custom`.
    - If `custom` selected, show text input for arbitrary OpenRouter model ID.
  - Update `EstimateCost` and `InvestigateWithAI` calls to pass the selected provider and model.

- [ ] **Step 4: Verify typecheck & build**
  Run: `npm --prefix frontend run build`
  Expected: Clean build without errors.

- [ ] **Step 5: Commit**
  ```bash
  git add frontend/src/types.ts frontend/src/components/SettingsModal.tsx frontend/src/components/AIModal.tsx app.go
  git commit -m "feat(ai): add OpenRouter, DeepSeek, and Groq configuration and model selection UI"
  ```

---

### Task 5: End-to-End Verification & Build Validation

**Files:**
- All modified backend & frontend files.

- [ ] **Step 1: Run complete Go test suite**
  Run: `go test -v ./...`
  Expected: All unit tests pass 100%.

- [ ] **Step 2: Run frontend production build**
  Run: `cd frontend && npm run build`
  Expected: Vite build succeeds with all chunks and workers generated.

- [ ] **Step 3: Compile desktop binary**
  Run: `wails build -s`
  Expected: `build/bin/logpulse.exe` compiles cleanly.

- [ ] **Step 4: Commit and finalize**
  ```bash
  git add .
  git commit -m "chore: complete log order, quick selector, and expanded AI provider features"
  ```

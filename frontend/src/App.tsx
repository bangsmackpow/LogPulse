import { 
  Activity, 
  Search, 
  Settings, 
  Bookmark, 
  RefreshCw, 
  Filter
} from 'lucide-react';
import { useLogStore } from './stores/useLogStore';

export default function App() {
  const { 
    logs, 
    selectedLog, 
    lines, 
    filterLevel, 
    searchTerm, 
    toggleLevel, 
    setSearchTerm, 
    isSessionActive, 
    setSessionActive, 
    addMarker 
  } = useLogStore();

  return (
    <div className="flex h-screen w-screen flex-col bg-canvas-900 text-gray-200 select-none">
      {/* Top Header Bar */}
      <header className="flex h-12 items-center justify-between border-b border-canvas-700 bg-canvas-800 px-4">
        <div className="flex items-center gap-3">
          <div className="flex items-center gap-2">
            <Activity className="h-5 w-5 text-indigo-400 animate-pulse" />
            <span className="font-bold text-base tracking-wide text-white">LogPulse</span>
          </div>
          <span className="rounded bg-canvas-700 px-2 py-0.5 text-xs text-gray-400 font-mono">
            {logs.length} logs discovered
          </span>
        </div>

        {/* Global Search & Filters */}
        <div className="flex items-center gap-3">
          <div className="relative">
            <Search className="absolute left-2.5 top-2 h-4 w-4 text-gray-400" />
            <input
              type="text"
              placeholder="Regex search lines..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              className="h-8 w-64 rounded bg-canvas-900 pl-8 pr-3 text-xs text-gray-200 placeholder-gray-500 border border-canvas-600 focus:border-indigo-500 focus:outline-none font-mono"
            />
          </div>

          {/* Level Filter Buttons */}
          <div className="flex items-center gap-1 rounded bg-canvas-900 p-0.5 border border-canvas-700 text-xs">
            {(['ERROR', 'WARN', 'INFO', 'DEBUG'] as const).map((lvl) => {
              const active = filterLevel.has(lvl);
              return (
                <button
                  key={lvl}
                  onClick={() => toggleLevel(lvl)}
                  className={`px-2 py-1 rounded font-mono text-[10px] font-semibold transition-colors ${
                    active
                      ? lvl === 'ERROR'
                        ? 'bg-red-950 text-red-400 border border-red-800'
                        : lvl === 'WARN'
                        ? 'bg-amber-950 text-amber-400 border border-amber-800'
                        : lvl === 'INFO'
                        ? 'bg-blue-950 text-blue-400 border border-blue-800'
                        : 'bg-gray-800 text-gray-300 border border-gray-700'
                      : 'text-gray-500 hover:text-gray-400'
                  }`}
                >
                  {lvl}
                </button>
              );
            })}
          </div>

          {/* Troubleshooting Session Controls */}
          <button
            onClick={() => {
              if (!isSessionActive) {
                setSessionActive(true);
                addMarker('Session Started');
              } else {
                addMarker(`Incident Mark #${Math.floor(Math.random() * 1000)}`);
              }
            }}
            className={`flex items-center gap-1.5 rounded px-2.5 py-1 text-xs font-medium transition-all ${
              isSessionActive
                ? 'bg-amber-600/20 text-amber-400 border border-amber-500 hover:bg-amber-600/30'
                : 'bg-canvas-700 text-gray-300 hover:bg-canvas-600 border border-canvas-600'
            }`}
          >
            <Bookmark className="h-3.5 w-3.5" />
            {isSessionActive ? 'Drop Mark' : 'Start Session'}
          </button>
        </div>

        {/* Right Action Icons */}
        <div className="flex items-center gap-2">
          <button
            className="rounded p-1.5 text-gray-400 hover:bg-canvas-700 hover:text-gray-200"
            title="Settings"
          >
            <Settings className="h-4 w-4" />
          </button>
        </div>
      </header>

      {/* Main Workspace Body */}
      <div className="flex flex-1 overflow-hidden">
        {/* Left Sidebar: Discovered Logs Catalog */}
        <aside className="w-72 border-r border-canvas-700 bg-canvas-800 flex flex-col">
          <div className="flex items-center justify-between border-b border-canvas-700 px-3 py-2 text-xs font-semibold text-gray-400">
            <span>DISCOVERED LOGS</span>
            <RefreshCw className="h-3.5 w-3.5 cursor-pointer hover:text-white" />
          </div>
          <div className="flex-1 overflow-y-auto p-2 space-y-1">
            {logs.map((log) => (
              <div
                key={log.filePath}
                onClick={() => useLogStore.getState().selectLog(log)}
                className={`group flex flex-col rounded p-2 text-xs cursor-pointer transition-colors ${
                  selectedLog?.filePath === log.filePath
                    ? 'bg-indigo-950/60 border border-indigo-700/60 text-white'
                    : 'bg-canvas-900/40 hover:bg-canvas-700/50 text-gray-300'
                }`}
              >
                <div className="flex items-center justify-between">
                  <span className="font-semibold text-indigo-300 truncate">{log.appName}</span>
                  <span className="text-[10px] text-gray-500 uppercase">{log.format}</span>
                </div>
                <span className="text-[11px] text-gray-400 truncate mt-0.5" title={log.filePath}>
                  {log.filePath}
                </span>
              </div>
            ))}
            {logs.length === 0 && (
              <div className="text-center text-xs text-gray-500 py-6">
                Scanning disk for logs...
              </div>
            )}
          </div>
        </aside>

        {/* Central Log Stream Viewport */}
        <main className="flex flex-1 flex-col overflow-hidden bg-canvas-900">
          <div className="flex h-8 items-center justify-between border-b border-canvas-700 bg-canvas-800/60 px-4 text-xs font-mono text-gray-400">
            <span className="truncate">
              {selectedLog ? selectedLog.filePath : 'Select a log to start live tail'}
            </span>
            <span>{lines.length.toLocaleString()} lines</span>
          </div>

          {/* Virtual Grid Area Placeholder - connected in Task 8 */}
          <div className="flex-1 overflow-auto p-2 font-mono text-xs">
            {lines.length === 0 ? (
              <div className="flex h-full flex-col items-center justify-center text-gray-500">
                <Filter className="h-8 w-8 mb-2 opacity-50" />
                <p>Waiting for log lines or select a file from the sidebar</p>
              </div>
            ) : (
              lines.slice(-100).map((l, i) => (
                <div key={i} className="flex gap-3 py-0.5 hover:bg-canvas-800/40">
                  <span className="w-12 text-right text-gray-600 select-none">{l.lineNumber}</span>
                  <span className={`w-14 font-semibold ${
                    l.level === 'ERROR' ? 'text-red-400' :
                    l.level === 'WARN' ? 'text-amber-400' :
                    l.level === 'DEBUG' ? 'text-gray-400' : 'text-blue-400'
                  }`}>
                    {l.level}
                  </span>
                  <span className="flex-1 text-gray-300 break-all">{l.raw}</span>
                </div>
              ))
            )}
          </div>
        </main>
      </div>
    </div>
  );
}

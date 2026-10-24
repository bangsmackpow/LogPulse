import React, { useState } from 'react';
import { 
  Activity, 
  Search, 
  Settings, 
  Bookmark, 
  Filter
} from 'lucide-react';
import { useLogStore } from './stores/useLogStore';
import { useLogStream } from './hooks/useLogStream';
import { LogTree } from './components/LogTree';
import { VirtualizedGrid } from './components/VirtualizedGrid';
import { ContextMenu } from './components/ContextMenu';
import { AIModal } from './components/AIModal';
import { SettingsModal } from './components/SettingsModal';
import { SessionControl } from './components/SessionControl';
import { DiscoveredLog, LogLine } from './types';

export default function App() {
  useLogStream();

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

  const [showSettings, setShowSettings] = useState(false);
  const [showAIModal, setShowAIModal] = useState(false);
  const [aiTargetIndex, setAiTargetIndex] = useState<number | undefined>(undefined);

  // Context Menu State
  const [contextMenu, setContextMenu] = useState<{
    x: number;
    y: number;
    text: string;
    filePath?: string;
    lineIndex?: number;
  } | null>(null);

  const handleRefresh = () => {
    if (window.go?.main?.App?.RefreshLogs) {
      window.go.main.App.RefreshLogs().then((res) => {
        if (res && res.length > 0) {
          useLogStore.getState().setLogs(res);
        }
      });
    }
  };

  const handleFileContextMenu = (e: React.MouseEvent, log: DiscoveredLog) => {
    e.preventDefault();
    setContextMenu({
      x: e.clientX,
      y: e.clientY,
      text: log.filePath,
      filePath: log.filePath,
    });
  };

  const handleLineContextMenu = (e: React.MouseEvent, line: LogLine, index: number) => {
    e.preventDefault();
    setContextMenu({
      x: e.clientX,
      y: e.clientY,
      text: line.raw,
      filePath: line.fileId,
      lineIndex: index,
    });
  };

  const handleQuickInvestigate = (_line: LogLine, index: number) => {
    setAiTargetIndex(index);
    setShowAIModal(true);
  };

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
            onClick={() => setShowSettings(true)}
            className="rounded p-1.5 text-gray-400 hover:bg-canvas-700 hover:text-gray-200"
            title="Settings & AI Providers"
          >
            <Settings className="h-4 w-4" />
          </button>
        </div>
      </header>

      {/* Main Workspace Body */}
      <div className="flex flex-1 overflow-hidden">
        {/* Left Sidebar: Discovered Logs Catalog */}
        <aside className="w-72 border-r border-canvas-700 bg-canvas-800 flex flex-col">
          <LogTree onRefresh={handleRefresh} onContextMenu={handleFileContextMenu} />
        </aside>

        {/* Central Log Stream Viewport */}
        <main className="flex flex-1 flex-col overflow-hidden bg-canvas-900">
          <SessionControl />
          <div className="flex h-8 items-center justify-between border-b border-canvas-700 bg-canvas-800/60 px-4 text-xs font-mono text-gray-400">
            <span className="truncate">
              {selectedLog ? selectedLog.filePath : 'Select a log to start live tail'}
            </span>
            <span>{lines.length.toLocaleString()} lines in buffer</span>
          </div>

          <div className="flex-1 overflow-hidden">
            {lines.length === 0 ? (
              <div className="flex h-full flex-col items-center justify-center text-gray-500">
                <Filter className="h-8 w-8 mb-2 opacity-40" />
                <p>Select a log file from the catalog to stream live</p>
              </div>
            ) : (
              <VirtualizedGrid
                lines={lines}
                onContextMenu={handleLineContextMenu}
                onQuickInvestigate={handleQuickInvestigate}
              />
            )}
          </div>
        </main>
      </div>

      {/* Right-Click Context Menu */}
      {contextMenu && (
        <ContextMenu
          x={contextMenu.x}
          y={contextMenu.y}
          selectedText={contextMenu.text}
          filePath={contextMenu.filePath}
          onClose={() => setContextMenu(null)}
          onInvestigateAI={() => {
            setAiTargetIndex(contextMenu.lineIndex);
            setShowAIModal(true);
          }}
          onOpenEditor={
            contextMenu.filePath
              ? () => window.go?.main?.App?.OpenFileInEditor(contextMenu.filePath!)
              : undefined
          }
          onRevealExplorer={
            contextMenu.filePath
              ? () => window.go?.main?.App?.RevealInExplorer(contextMenu.filePath!)
              : undefined
          }
          onAddMarker={() => addMarker(`Mark @ line ${contextMenu.lineIndex || lines.length}`)}
        />
      )}

      {/* AI Investigation Modal */}
      <AIModal
        isOpen={showAIModal}
        onClose={() => setShowAIModal(false)}
        targetLineIndex={aiTargetIndex}
      />

      {/* Settings Modal */}
      <SettingsModal
        isOpen={showSettings}
        onClose={() => setShowSettings(false)}
      />
    </div>
  );
}

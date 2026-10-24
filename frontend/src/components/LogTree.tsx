import React, { useState } from 'react';
import { FileText, Folder, RefreshCw, Star, HardDrive } from 'lucide-react';
import { useLogStore } from '../stores/useLogStore';
import { DiscoveredLog } from '../types';

interface LogTreeProps {
  onRefresh: () => void;
  onContextMenu: (e: React.MouseEvent, log: DiscoveredLog) => void;
}

export const LogTree: React.FC<LogTreeProps> = ({ onRefresh, onContextMenu }) => {
  const { logs, selectedLog, selectLog, activeTailedPaths, toggleLogSelection } = useLogStore();
  const [filter, setFilter] = useState('');

  const filteredLogs = logs.filter(
    (l) =>
      l.appName.toLowerCase().includes(filter.toLowerCase()) ||
      l.filePath.toLowerCase().includes(filter.toLowerCase())
  );

  const formatSize = (bytes: number) => {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
  };

  return (
    <div className="flex h-full flex-col bg-canvas-800 select-none">
      <div className="flex items-center justify-between border-b border-canvas-700 px-3 py-2 text-xs font-semibold text-gray-400">
        <div className="flex items-center gap-1.5">
          <HardDrive className="h-3.5 w-3.5 text-gray-400" />
          <span>LOG DISCOVERY CATALOG</span>
        </div>
        <button
          onClick={onRefresh}
          className="rounded p-1 text-gray-400 hover:bg-canvas-700 hover:text-white transition-colors"
          title="Refresh Log Crawl"
        >
          <RefreshCw className="h-3.5 w-3.5" />
        </button>
      </div>

      <div className="p-2 border-b border-canvas-700">
        <input
          type="text"
          placeholder="Filter discovered apps..."
          value={filter}
          onChange={(e) => setFilter(e.target.value)}
          className="w-full h-7 rounded bg-canvas-900 px-2 text-xs text-gray-200 placeholder-gray-500 border border-canvas-600 focus:border-indigo-500 focus:outline-none"
        />
      </div>

      <div className="flex-1 overflow-y-auto p-2 space-y-1">
        {filteredLogs.map((log) => {
          const isSelected = selectedLog?.filePath === log.filePath;
          const isTailed = activeTailedPaths.has(log.filePath);
          return (
            <div
              key={log.filePath}
              onClick={(e) => {
                if (e.ctrlKey || e.metaKey) {
                  toggleLogSelection(log);
                } else {
                  selectLog(log);
                }
              }}
              onContextMenu={(e) => onContextMenu(e, log)}
              className={`group flex flex-col rounded p-2 text-xs cursor-pointer border transition-colors ${
                isSelected
                  ? 'bg-indigo-950/70 border-indigo-600 text-white shadow-sm'
                  : isTailed
                  ? 'bg-indigo-950/30 border-indigo-900 text-indigo-200'
                  : 'bg-canvas-900/50 border-transparent hover:bg-canvas-700/60 hover:border-canvas-600 text-gray-300'
              }`}
            >
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-1.5 truncate">
                  <FileText className={`h-3.5 w-3.5 shrink-0 ${isSelected ? 'text-indigo-400' : 'text-gray-400'}`} />
                  <span className="font-semibold truncate">{log.appName}</span>
                </div>
                <div className="flex items-center gap-1">
                  <span className="rounded bg-canvas-700/80 px-1 py-0.2 text-[9px] font-mono text-gray-400 uppercase">
                    {log.format}
                  </span>
                  {log.isFavorite && <Star className="h-3 w-3 text-amber-400 fill-amber-400" />}
                </div>
              </div>

              <div className="mt-1 flex items-center justify-between text-[10px] text-gray-400">
                <span className="truncate max-w-[160px]" title={log.filePath}>
                  {log.filePath}
                </span>
                <span className="font-mono text-gray-400 shrink-0">{formatSize(log.fileSize)}</span>
              </div>
            </div>
          );
        })}

        {filteredLogs.length === 0 && (
          <div className="flex flex-col items-center justify-center p-6 text-center text-xs text-gray-400">
            <Folder className="h-8 w-8 mb-2 opacity-40 text-gray-400" />
            <p>No log files match filter</p>
          </div>
        )}
      </div>
    </div>
  );
};

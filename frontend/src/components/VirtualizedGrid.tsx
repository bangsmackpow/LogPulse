import React, { useRef, useMemo, useEffect, useState } from 'react';
import { useVirtualizer } from '@tanstack/react-virtual';
import { LogLine } from '../types';
import { useLogStore } from '../stores/useLogStore';
import { Bookmark, Sparkles } from 'lucide-react';

interface VirtualizedGridProps {
  lines: LogLine[];
  onContextMenu: (e: React.MouseEvent, line: LogLine, index: number) => void;
  onQuickInvestigate: (line: LogLine, index: number) => void;
}

export const VirtualizedGrid: React.FC<VirtualizedGridProps> = ({
  lines,
  onContextMenu,
  onQuickInvestigate,
}) => {
  const parentRef = useRef<HTMLDivElement>(null);
  const { filterLevel, searchTerm, markers, isNewestOnTop, activeTailedPaths } = useLogStore();
  const [filteredIndices, setFilteredIndices] = useState<number[]>([]);
  const isAtTopRef = useRef(true);
  const isAtBottomRef = useRef(true);

  // Search Web Worker instance
  const workerRef = useRef<Worker | null>(null);

  useEffect(() => {
    // Instantiate web worker
    workerRef.current = new Worker(
      new URL('../workers/searchWorker.ts', import.meta.url),
      { type: 'module' }
    );

    workerRef.current.onmessage = (e: MessageEvent<{ filteredIndices: number[] }>) => {
      setFilteredIndices(e.data.filteredIndices);
    };

    return () => {
      workerRef.current?.terminate();
    };
  }, []);

  // Post search task to worker whenever lines, searchTerm, or filterLevel change (with trailing debounce)
  useEffect(() => {
    if (!workerRef.current) return;

    const timer = setTimeout(() => {
      workerRef.current?.postMessage({
        lines,
        searchTerm,
        filterLevel: Array.from(filterLevel),
      });
    }, 80);

    return () => clearTimeout(timer);
  }, [lines, searchTerm, filterLevel]);

  // Marker lookup map: lineNumber -> marker label
  const markerMap = useMemo(() => {
    const map = new Map<number, string>();
    for (const m of markers) {
      map.set(m.afterLineNumber, m.label);
    }
    return map;
  }, [markers]);

  // Regex highlighting matcher
  const searchRegex = useMemo(() => {
    if (!searchTerm.trim()) return null;
    try {
      return new RegExp(`(${searchTerm})`, 'gi');
    } catch {
      return null;
    }
  }, [searchTerm]);

  const rowVirtualizer = useVirtualizer({
    count: filteredIndices.length,
    getScrollElement: () => parentRef.current,
    estimateSize: () => 24,
    overscan: 25,
  });

  const handleScroll = (e: React.UIEvent<HTMLDivElement>) => {
    const target = e.currentTarget;
    isAtTopRef.current = target.scrollTop <= 40;
    isAtBottomRef.current = target.scrollHeight - target.scrollTop - target.clientHeight <= 40;
  };

  useEffect(() => {
    if (!parentRef.current || filteredIndices.length === 0) return;
    if (isNewestOnTop) {
      if (isAtTopRef.current) {
        rowVirtualizer.scrollToOffset(0);
      }
    } else {
      if (isAtBottomRef.current) {
        rowVirtualizer.scrollToIndex(filteredIndices.length - 1);
      }
    }
  }, [lines.length, isNewestOnTop, filteredIndices.length]);

  const highlightMatches = (text: string) => {
    if (!searchRegex) return text;
    const parts = text.split(searchRegex);
    return parts.map((part, i) =>
      searchRegex.test(part) ? (
        <mark key={i} className="bg-yellow-500/30 text-yellow-200 rounded px-0.5 font-bold">
          {part}
        </mark>
      ) : (
        part
      )
    );
  };

  return (
    <div
      ref={parentRef}
      onScroll={handleScroll}
      className="relative h-full w-full overflow-auto bg-canvas-900 font-mono text-xs select-text"
      style={{ contain: 'strict' }}
    >
      <div
        style={{
          height: `${rowVirtualizer.getTotalSize()}px`,
          width: '100%',
          position: 'relative',
        }}
      >
        {rowVirtualizer.getVirtualItems().map((virtualRow) => {
          const originalIndex = isNewestOnTop
            ? filteredIndices[filteredIndices.length - 1 - virtualRow.index]
            : filteredIndices[virtualRow.index];
          const line = lines[originalIndex];
          if (!line) return null;

          const markerLabel = markerMap.get(line.lineNumber);

          return (
            <div
              key={virtualRow.key}
              style={{
                position: 'absolute',
                top: 0,
                left: 0,
                width: '100%',
                height: `${virtualRow.size}px`,
                transform: `translateY(${virtualRow.start}px)`,
              }}
              onContextMenu={(e) => onContextMenu(e, line, originalIndex)}
              className={`group flex items-center hover:bg-canvas-800/60 border-b border-canvas-800/30 transition-colors ${
                line.level === 'ERROR' ? 'bg-red-950/20' : ''
              }`}
            >
              {/* Marker Banner if pinned */}
              {markerLabel && (
                <div className="absolute -top-3 left-14 z-10 flex items-center gap-1 rounded bg-amber-500/20 border border-amber-500/50 px-2 py-0.5 text-[10px] text-amber-300 font-sans shadow">
                  <Bookmark className="h-3 w-3 fill-amber-400 text-amber-400" />
                  <span>{markerLabel}</span>
                </div>
              )}

              {/* Line Gutter */}
              <div className="w-14 shrink-0 pr-3 text-right text-gray-400 select-none border-r border-canvas-700/50 bg-canvas-900/80">
                {line.lineNumber}
              </div>

              {/* Level Badge */}
              <div
                className={`w-16 shrink-0 px-2 text-center text-[10px] font-bold select-none ${
                  line.level === 'ERROR'
                    ? 'text-red-400'
                    : line.level === 'WARN'
                    ? 'text-amber-400'
                    : line.level === 'DEBUG'
                    ? 'text-gray-400'
                    : 'text-blue-400'
                }`}
              >
                {line.level}
              </div>

              {/* Source File Badge (Multi-Tail Stream Multiplexing) */}
              {activeTailedPaths.size > 1 && line.fileId && (
                <div
                  className="shrink-0 mx-1 px-1.5 py-0.5 rounded bg-indigo-950/70 border border-indigo-800/50 text-[9px] text-indigo-300 font-mono select-none max-w-[130px] truncate"
                  title={line.fileId}
                >
                  {line.fileId.split(/[/\\]/).pop()}
                </div>
              )}

              {/* Raw Log Line Content */}
              <div className="flex-1 overflow-hidden truncate px-2 text-gray-200">
                {highlightMatches(line.raw)}
              </div>

              {/* Hover Quick Action: Investigate with AI */}
              <div className="hidden group-hover:flex items-center pr-3 select-none">
                <button
                  onClick={() => onQuickInvestigate(line, originalIndex)}
                  className="flex items-center gap-1 rounded bg-indigo-950 px-2 py-0.5 text-[10px] font-sans font-medium text-indigo-300 border border-indigo-700 hover:bg-indigo-900 transition-colors"
                  title="Investigate line with AI"
                >
                  <Sparkles className="h-3 w-3 text-indigo-400" />
                  <span>Ask AI</span>
                </button>
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
};

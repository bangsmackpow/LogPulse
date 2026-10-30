import { create } from 'zustand';
import { DiscoveredLog, LogLine, SessionMarker } from '../types';

interface LogState {
  logs: DiscoveredLog[];
  selectedLog: DiscoveredLog | null;
  activeTailedPaths: Set<string>;
  lines: LogLine[];
  filterLevel: Set<string>;
  searchTerm: string;
  markers: SessionMarker[];
  isSessionActive: boolean;
  isNewestOnTop: boolean;

  setLogs: (logs: DiscoveredLog[]) => void;
  addDiscoveredLog: (log: DiscoveredLog) => void;
  selectLog: (log: DiscoveredLog | null) => void;
  toggleLogSelection: (log: DiscoveredLog) => void;
  appendLines: (newLines: LogLine[]) => void;
  clearLines: () => void;
  toggleLevel: (level: string) => void;
  setSearchTerm: (term: string) => void;
  addMarker: (label: string) => void;
  setSessionActive: (active: boolean) => void;
  setNewestOnTop: (val: boolean) => void;
  switchActiveTails: (newPaths: string[], groupLabel?: string) => void;
}

const MAX_BUFFER = 250000;

export const useLogStore = create<LogState>((set) => ({
  logs: [],
  selectedLog: null,
  activeTailedPaths: new Set(),
  lines: [],
  filterLevel: new Set(['ERROR', 'WARN', 'INFO', 'DEBUG']),
  searchTerm: '',
  markers: [],
  isSessionActive: false,
  isNewestOnTop: typeof window !== 'undefined' && localStorage.getItem('logpulse_newest_on_top') === 'true',

  setLogs: (logs) => set({ logs }),
  setNewestOnTop: (val) => {
    if (typeof window !== 'undefined') {
      localStorage.setItem('logpulse_newest_on_top', String(val));
    }
    set({ isNewestOnTop: val });
  },
  addDiscoveredLog: (log) =>
    set((state) => {
      const exists = state.logs.some((l) => l.filePath === log.filePath);
      if (exists) return state;
      return { logs: [log, ...state.logs] };
    }),

  selectLog: (log) =>
    set((state) => {
      const active = new Set(state.activeTailedPaths);
      if (log) active.add(log.filePath);
      return { selectedLog: log, activeTailedPaths: active, lines: [] };
    }),

  toggleLogSelection: (log) =>
    set((state) => {
      const active = new Set(state.activeTailedPaths);
      if (active.has(log.filePath)) {
        active.delete(log.filePath);
      } else {
        active.add(log.filePath);
      }
      return {
        selectedLog: log,
        activeTailedPaths: active,
      };
    }),

  switchActiveTails: (newPaths, groupLabel) =>
    set((state) => {
      const newSet = new Set(newPaths);
      // Clean up previous tails
      state.activeTailedPaths.forEach((oldPath) => {
        if (!newSet.has(oldPath) && window.go?.main?.App?.StopTail) {
          window.go.main.App.StopTail(oldPath).catch((err) => {
            console.error('Failed to stop tail:', oldPath, err);
          });
        }
      });

      let targetLog: DiscoveredLog | null = null;
      if (newPaths.length === 1) {
        targetLog = state.logs.find((l) => l.filePath === newPaths[0]) || {
          id: -1,
          filePath: newPaths[0],
          appName: groupLabel || 'Log',
          format: 'text',
          fileSize: 0,
          lastModified: new Date().toISOString(),
          isFavorite: false,
          tag: 'active',
        };
      } else if (newPaths.length > 1) {
        targetLog = {
          id: -1,
          filePath: groupLabel || `${newPaths.length} Active Streams`,
          appName: groupLabel || 'Multi-Tail',
          format: 'multiplex',
          fileSize: 0,
          lastModified: new Date().toISOString(),
          isFavorite: false,
          tag: 'multiplex',
        };
      }

      return {
        selectedLog: targetLog,
        activeTailedPaths: newSet,
        lines: [],
      };
    }),

  appendLines: (newLines) =>
    set((state) => {
      const combined = [...state.lines, ...newLines];
      if (combined.length > MAX_BUFFER) {
        return { lines: combined.slice(combined.length - MAX_BUFFER) };
      }
      return { lines: combined };
    }),

  clearLines: () => set({ lines: [] }),

  toggleLevel: (level) =>
    set((state) => {
      const next = new Set(state.filterLevel);
      if (next.has(level)) {
        if (next.size > 1) next.delete(level);
      } else {
        next.add(level);
      }
      return { filterLevel: next };
    }),

  setSearchTerm: (term) => set({ searchTerm: term }),

  addMarker: (label) =>
    set((state) => ({
      markers: [
        ...state.markers,
        {
          id: Math.random().toString(36).substring(7),
          timestamp: new Date().toISOString(),
          label,
          afterLineNumber: state.lines.length,
        },
      ],
    })),

  setSessionActive: (active) => set({ isSessionActive: active }),
}));

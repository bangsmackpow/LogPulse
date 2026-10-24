import { useEffect } from 'react';
import { useLogStore } from '../stores/useLogStore';
import { DiscoveredLog, LogLine, ScrubMatch } from '../types';

declare global {
  interface Window {
    runtime?: {
      EventsOn: (eventName: string, callback: (...args: any[]) => void) => () => void;
      EventsEmit: (eventName: string, ...args: any[]) => void;
    };
    go?: {
      main?: {
        App?: {
          GetDiscoveredLogs: () => Promise<DiscoveredLog[]>;
          RefreshLogs: () => Promise<DiscoveredLog[]>;
          StartTail: (path: string) => Promise<void>;
          StopTail: (path: string) => Promise<void>;
          OpenFileInEditor: (path: string) => Promise<void>;
          RevealInExplorer: (path: string) => Promise<void>;
          GetSettings: () => Promise<Record<string, string>>;
          SaveSettings: (settings: Record<string, string>) => Promise<void>;
          InvestigateWithAI: (req: any) => Promise<any>;
          EstimateCost: (provider: string, model: string, prompt: string) => Promise<any>;
          ScrubText: (text: string) => Promise<{ text: string; matches: ScrubMatch[] }>;
        };
      };
    };
  }
}

export function useLogStream() {
  const { setLogs, addDiscoveredLog, appendLines, activeTailedPaths } = useLogStore();

  useEffect(() => {
    // Load existing discovered logs on mount
    if (window.go?.main?.App?.GetDiscoveredLogs) {
      window.go.main.App.GetDiscoveredLogs().then((logs) => {
        if (logs && logs.length > 0) {
          setLogs(logs);
        }
      });
    }

    // Subscribe to Wails events if available
    if (window.runtime?.EventsOn) {
      const unsubBatch = window.runtime.EventsOn('log:batch', (batch: LogLine[]) => {
        if (batch && Array.isArray(batch)) {
          appendLines(batch);
        }
      });

      const unsubDiscovered = window.runtime.EventsOn('log:discovered', (log: DiscoveredLog) => {
        if (log) {
          addDiscoveredLog(log);
        }
      });

      const unsubTier1 = window.runtime.EventsOn('logs:tier1_complete', (logs: DiscoveredLog[]) => {
        if (logs && Array.isArray(logs)) {
          setLogs(logs);
        }
      });

      return () => {
        unsubBatch();
        unsubDiscovered();
        unsubTier1();
      };
    }
  }, [setLogs, addDiscoveredLog, appendLines]);

  // When activeTailedPaths changes, ensure all paths are being tailed
  useEffect(() => {
    if (window.go?.main?.App?.StartTail) {
      activeTailedPaths.forEach((path) => {
        window.go!.main!.App!.StartTail(path).catch((err) => {
          console.error('Failed to start tail for', path, err);
        });
      });
    }
  }, [activeTailedPaths]);
}

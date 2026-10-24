import React from 'react';
import { Bookmark, Clock, XCircle } from 'lucide-react';
import { useLogStore } from '../stores/useLogStore';

export const SessionControl: React.FC = () => {
  const { markers, isSessionActive, setSessionActive } = useLogStore();

  if (!isSessionActive && markers.length === 0) {
    return null;
  }

  return (
    <div className="flex items-center gap-2 border-b border-canvas-700 bg-canvas-850 px-4 py-1 text-xs select-none">
      <div className="flex items-center gap-1.5 text-amber-400 font-medium">
        <Bookmark className="h-3.5 w-3.5 fill-amber-400" />
        <span>Active Session:</span>
      </div>

      <div className="flex items-center gap-1.5 overflow-x-auto">
        {markers.map((m) => (
          <span
            key={m.id}
            className="flex items-center gap-1 rounded bg-amber-950/60 border border-amber-800/80 px-2 py-0.5 text-[10px] text-amber-200 font-mono"
          >
            <Clock className="h-2.5 w-2.5 text-amber-400" />
            <span>{m.label} (line #{m.afterLineNumber})</span>
          </span>
        ))}
      </div>

      {isSessionActive && (
        <button
          onClick={() => setSessionActive(false)}
          className="ml-auto flex items-center gap-1 text-[11px] text-gray-400 hover:text-red-400 transition-colors"
          title="Stop Troubleshooting Session"
        >
          <XCircle className="h-3.5 w-3.5" />
          <span>Stop Session</span>
        </button>
      )}
    </div>
  );
};

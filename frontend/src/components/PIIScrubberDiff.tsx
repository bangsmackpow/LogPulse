import React, { useState } from 'react';
import { ShieldCheck, Eye, EyeOff } from 'lucide-react';
import { ScrubMatch } from '../types';

interface PIIScrubberDiffProps {
  originalText: string;
  scrubbedText: string;
  matches: ScrubMatch[];
}

export const PIIScrubberDiff: React.FC<PIIScrubberDiffProps> = ({
  originalText,
  scrubbedText,
  matches,
}) => {
  const [showOriginal, setShowOriginal] = useState(false);

  return (
    <div className="flex flex-col rounded-lg border border-canvas-700 bg-canvas-900 overflow-hidden text-xs">
      <div className="flex items-center justify-between border-b border-canvas-700 bg-canvas-800/80 px-3 py-1.5">
        <div className="flex items-center gap-1.5 text-emerald-400 font-medium">
          <ShieldCheck className="h-4 w-4" />
          <span>Local Pre-Flight Privacy Scrubber ({matches.length} redacted)</span>
        </div>
        <button
          onClick={() => setShowOriginal(!showOriginal)}
          className="flex items-center gap-1 text-[11px] text-gray-400 hover:text-gray-200 transition-colors"
        >
          {showOriginal ? <EyeOff className="h-3.5 w-3.5" /> : <Eye className="h-3.5 w-3.5" />}
          <span>{showOriginal ? 'Show Redacted Preview' : 'Reveal Unsanitized'}</span>
        </button>
      </div>

      <div className="p-3 font-mono text-[11px] max-h-48 overflow-y-auto whitespace-pre-wrap break-all leading-relaxed bg-canvas-950">
        {showOriginal ? (
          <span className="text-amber-200/90">{originalText}</span>
        ) : (
          <span className="text-gray-300">{scrubbedText}</span>
        )}
      </div>

      {matches.length > 0 && !showOriginal && (
        <div className="flex flex-wrap gap-1.5 p-2 bg-canvas-900 border-t border-canvas-800">
          {matches.map((m, i) => (
            <span
              key={i}
              className="rounded bg-emerald-950/80 border border-emerald-700/60 px-1.5 py-0.5 text-[10px] text-emerald-300 font-mono"
            >
              {m.type}: {m.replacement}
            </span>
          ))}
        </div>
      )}
    </div>
  );
};

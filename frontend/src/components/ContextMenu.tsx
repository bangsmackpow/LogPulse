import React, { useEffect, useRef } from 'react';
import { Sparkles, Copy, ExternalLink, FolderOpen, Bookmark } from 'lucide-react';

interface ContextMenuProps {
  x: number;
  y: number;
  selectedText: string;
  filePath?: string;
  onClose: () => void;
  onInvestigateAI: () => void;
  onOpenEditor?: () => void;
  onRevealExplorer?: () => void;
  onAddMarker?: () => void;
}

export const ContextMenu: React.FC<ContextMenuProps> = ({
  x,
  y,
  selectedText,
  onClose,
  onInvestigateAI,
  onOpenEditor,
  onRevealExplorer,
  onAddMarker,
}) => {
  const menuRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const handleOutsideClick = (e: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
        onClose();
      }
    };
    window.addEventListener('mousedown', handleOutsideClick);
    return () => window.removeEventListener('mousedown', handleOutsideClick);
  }, [onClose]);

  const handleCopy = () => {
    if (selectedText) {
      navigator.clipboard.writeText(selectedText);
    }
    onClose();
  };

  return (
    <div
      ref={menuRef}
      style={{ top: `${y}px`, left: `${x}px` }}
      className="fixed z-50 min-w-[200px] rounded-lg border border-canvas-600 bg-canvas-800 p-1 shadow-2xl backdrop-blur-md text-xs text-gray-200"
    >
      <button
        onClick={() => {
          onInvestigateAI();
          onClose();
        }}
        className="flex w-full items-center gap-2 rounded px-2.5 py-1.5 font-medium text-indigo-300 hover:bg-indigo-950/60 transition-colors"
      >
        <Sparkles className="h-3.5 w-3.5 text-indigo-400" />
        <span>Investigate with AI</span>
      </button>

      <div className="my-1 border-t border-canvas-700" />

      <button
        onClick={handleCopy}
        className="flex w-full items-center gap-2 rounded px-2.5 py-1.5 hover:bg-canvas-700 text-gray-300 transition-colors"
      >
        <Copy className="h-3.5 w-3.5 text-gray-400" />
        <span>Copy Line</span>
      </button>

      {onAddMarker && (
        <button
          onClick={() => {
            onAddMarker();
            onClose();
          }}
          className="flex w-full items-center gap-2 rounded px-2.5 py-1.5 hover:bg-canvas-700 text-gray-300 transition-colors"
        >
          <Bookmark className="h-3.5 w-3.5 text-amber-400" />
          <span>Mark Troubleshooting Point</span>
        </button>
      )}

      {(onOpenEditor || onRevealExplorer) && (
        <div className="my-1 border-t border-canvas-700" />
      )}

      {onOpenEditor && (
        <button
          onClick={() => {
            onOpenEditor();
            onClose();
          }}
          className="flex w-full items-center gap-2 rounded px-2.5 py-1.5 hover:bg-canvas-700 text-gray-300 transition-colors"
        >
          <ExternalLink className="h-3.5 w-3.5 text-gray-400" />
          <span>Open in Default Editor</span>
        </button>
      )}

      {onRevealExplorer && (
        <button
          onClick={() => {
            onRevealExplorer();
            onClose();
          }}
          className="flex w-full items-center gap-2 rounded px-2.5 py-1.5 hover:bg-canvas-700 text-gray-300 transition-colors"
        >
          <FolderOpen className="h-3.5 w-3.5 text-gray-400" />
          <span>Reveal in File Explorer</span>
        </button>
      )}
    </div>
  );
};

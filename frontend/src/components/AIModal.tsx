import React, { useState, useEffect, useMemo } from 'react';
import { X, Sparkles, Send, Copy, Check, Sliders, Shield, AlertCircle, Info } from 'lucide-react';
import { useAIStore } from '../stores/useAIStore';
import { useLogStore } from '../stores/useLogStore';
import { PIIScrubberDiff } from './PIIScrubberDiff';
import { CostBadge } from './CostBadge';
import { AIResponse, ScrubMatch } from '../types';

interface AIModalProps {
  isOpen: boolean;
  onClose: () => void;
  targetLineIndex?: number;
}

export const AIModal: React.FC<AIModalProps> = ({ isOpen, onClose, targetLineIndex }) => {
  const { lines } = useLogStore();
  const {
    contextBefore,
    contextAfter,
    setContextBefore,
    setContextAfter,
    userNotes,
    setUserNotes,
    includeTelemetry,
    setIncludeTelemetry,
    provider,
    model,
  } = useAIStore();

  const [loading, setLoading] = useState(false);
  const [response, setResponse] = useState<AIResponse | null>(null);
  const [errorMsg, setErrorMsg] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);

  // Compute surrounding context lines based on N before and M after
  const rawContextBlock = useMemo(() => {
    if (targetLineIndex === undefined || targetLineIndex < 0 || targetLineIndex >= lines.length) {
      return lines.slice(-20).map((l) => l.raw).join('\n');
    }
    const start = Math.max(0, targetLineIndex - contextBefore);
    const end = Math.min(lines.length, targetLineIndex + contextAfter + 1);
    return lines.slice(start, end).map((l) => l.raw).join('\n');
  }, [lines, targetLineIndex, contextBefore, contextAfter]);

  // Local PII Scrubbing
  const [scrubbedText, setScrubbedText] = useState('');
  const [scrubMatches, setScrubMatches] = useState<ScrubMatch[]>([]);

  useEffect(() => {
    if (window.go?.main?.App?.ScrubText) {
      window.go.main.App.ScrubText(rawContextBlock).then((res: any) => {
        if (Array.isArray(res) && res.length >= 2) {
          setScrubbedText(res[0]);
          setScrubMatches(res[1] || []);
        } else {
          setScrubbedText(rawContextBlock);
        }
      });
    } else {
      setScrubbedText(rawContextBlock);
    }
  }, [rawContextBlock]);

  // Telemetry string
  const telemetryString = useMemo(() => {
    if (!includeTelemetry) return '';
    return `OS: Windows 11 (build 22631)\nPlatform: x86_64\nLocal Time: ${new Date().toISOString()}\nTarget Application: ${lines[0]?.fileId || 'Unknown'}`;
  }, [includeTelemetry, lines]);

  // Token & Cost Estimation
  const estimatedTokens = useMemo(() => {
    const fullText = `${userNotes}\n${scrubbedText}\n${telemetryString}`;
    return Math.ceil(fullText.length / 4);
  }, [userNotes, scrubbedText, telemetryString]);

  const estimatedCost = useMemo(() => {
    if (provider === 'ollama') return 0;
    return (estimatedTokens / 1_000_000) * 3.0; // conservative average
  }, [estimatedTokens, provider]);

  if (!isOpen) return null;

  const handleInvestigate = async () => {
    setLoading(true);
    setErrorMsg(null);

    const payload = {
      provider,
      model,
      prompt: scrubbedText,
      userNotes,
      telemetry: telemetryString,
    };

    try {
      if (window.go?.main?.App?.InvestigateWithAI) {
        const res = await window.go.main.App.InvestigateWithAI(payload);
        setResponse(res);
      } else {
        // Dev fallback simulation
        setTimeout(() => {
          setResponse({
            text: `### AI Diagnostic Assessment\n\nThe log trace shows a fatal database connection timeout while attempting to reach port 5432. The application retry threshold was exceeded after 3 attempts.\n\n**Actionable Fixes:**\n1. Check if the PostgreSQL service is running locally.\n2. Ensure firewall rules allow loopback communication on port 5432.\n3. Verify database credentials in the configuration file.`,
            inputTokens: estimatedTokens,
            outputTokens: 140,
            costUsd: estimatedCost,
          });
          setLoading(false);
        }, 1200);
        return;
      }
    } catch (err: any) {
      setErrorMsg(err.message || String(err));
    } finally {
      setLoading(false);
    }
  };

  const handleCopyResponse = () => {
    if (response) {
      navigator.clipboard.writeText(response.text);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm p-4 select-text">
      <div className="flex h-[88vh] w-full max-w-4xl flex-col rounded-xl border border-canvas-600 bg-canvas-800 shadow-2xl text-xs text-gray-200 overflow-hidden">
        {/* Modal Header */}
        <div className="flex items-center justify-between border-b border-canvas-700 bg-canvas-850 px-4 py-3">
          <div className="flex items-center gap-2">
            <Sparkles className="h-4 w-4 text-indigo-400" />
            <span className="font-semibold text-sm text-white">Investigate with AI Assist</span>
            <span className="rounded bg-canvas-700 px-2 py-0.5 text-[10px] text-indigo-300 font-mono uppercase">
              {provider} • {model}
            </span>
          </div>
          <button onClick={onClose} className="rounded p-1 text-gray-400 hover:text-white">
            <X className="h-4 w-4" />
          </button>
        </div>

        {/* Modal Body */}
        <div className="flex flex-1 overflow-hidden">
          {/* Left Column: Context Configuration & Scrubber */}
          <div className="w-1/2 border-r border-canvas-700 p-4 flex flex-col gap-3 overflow-y-auto">
            {/* Context Slider */}
            <div className="rounded-lg border border-canvas-700 bg-canvas-900/60 p-3 space-y-2">
              <div className="flex items-center justify-between text-xs text-gray-300 font-medium">
                <span className="flex items-center gap-1.5">
                  <Sliders className="h-3.5 w-3.5 text-gray-400" />
                  Context Bracketing
                </span>
                <span className="font-mono text-[11px] text-indigo-400">
                  -{contextBefore} / +{contextAfter} lines
                </span>
              </div>
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label className="text-[10px] text-gray-400 block mb-1">Lines Before ({contextBefore})</label>
                  <input
                    type="range"
                    min="0"
                    max="50"
                    value={contextBefore}
                    onChange={(e) => setContextBefore(parseInt(e.target.value))}
                    className="w-full h-1.5 bg-canvas-700 rounded-lg appearance-none cursor-pointer accent-indigo-500"
                  />
                </div>
                <div>
                  <label className="text-[10px] text-gray-400 block mb-1">Lines After ({contextAfter})</label>
                  <input
                    type="range"
                    min="0"
                    max="50"
                    value={contextAfter}
                    onChange={(e) => setContextAfter(parseInt(e.target.value))}
                    className="w-full h-1.5 bg-canvas-700 rounded-lg appearance-none cursor-pointer accent-indigo-500"
                  />
                </div>
              </div>
            </div>

            {/* User Notes */}
            <div>
              <label className="block text-[11px] font-medium text-gray-300 mb-1">
                User Notes & Steps to Reproduce
              </label>
              <textarea
                rows={2}
                placeholder="e.g., Clicked export button right before this happened..."
                value={userNotes}
                onChange={(e) => setUserNotes(e.target.value)}
                className="w-full rounded bg-canvas-900 border border-canvas-700 p-2 text-xs text-gray-200 placeholder-gray-500 focus:outline-none focus:border-indigo-500 resize-none font-sans"
              />
            </div>

            {/* System Telemetry Toggle */}
            <div className="flex items-center justify-between rounded border border-canvas-700 bg-canvas-900/60 p-2.5">
              <div className="flex items-center gap-2">
                <Info className="h-3.5 w-3.5 text-gray-400" />
                <span className="text-[11px] text-gray-300 font-medium">Inject System Telemetry</span>
              </div>
              <input
                type="checkbox"
                checked={includeTelemetry}
                onChange={(e) => setIncludeTelemetry(e.target.checked)}
                className="h-4 w-4 rounded bg-canvas-900 border-canvas-600 text-indigo-600 focus:ring-0 cursor-pointer"
              />
            </div>

            {/* Pre-Flight PII Redaction Preview */}
            <div className="flex-1 min-h-[140px] flex flex-col">
              <label className="block text-[11px] font-medium text-gray-300 mb-1">
                Sanitized Prompt Payload
              </label>
              <PIIScrubberDiff
                originalText={rawContextBlock}
                scrubbedText={scrubbedText}
                matches={scrubMatches}
              />
            </div>
          </div>

          {/* Right Column: AI Analysis Result */}
          <div className="w-1/2 p-4 flex flex-col bg-canvas-850">
            <div className="flex items-center justify-between border-b border-canvas-700 pb-2 mb-3">
              <span className="font-semibold text-gray-300 text-xs">DIAGNOSTIC REPORT</span>
              {response && (
                <button
                  onClick={handleCopyResponse}
                  className="flex items-center gap-1 text-[11px] text-gray-400 hover:text-white transition-colors"
                >
                  {copied ? <Check className="h-3.5 w-3.5 text-emerald-400" /> : <Copy className="h-3.5 w-3.5" />}
                  <span>{copied ? 'Copied' : 'Copy Report'}</span>
                </button>
              )}
            </div>

            <div className="flex-1 overflow-y-auto rounded-lg border border-canvas-700 bg-canvas-950 p-4 font-sans text-xs leading-relaxed text-gray-200 whitespace-pre-wrap">
              {loading && (
                <div className="flex h-full flex-col items-center justify-center gap-2 text-indigo-400">
                  <Sparkles className="h-6 w-6 animate-spin" />
                  <span className="text-gray-400 text-xs">Analyzing log context with {model}...</span>
                </div>
              )}

              {errorMsg && (
                <div className="flex items-start gap-2 rounded bg-red-950/60 border border-red-800 p-3 text-red-300">
                  <AlertCircle className="h-4 w-4 shrink-0 mt-0.5" />
                  <div className="text-xs">{errorMsg}</div>
                </div>
              )}

              {!loading && !errorMsg && !response && (
                <div className="flex h-full flex-col items-center justify-center text-center text-gray-500 p-6">
                  <Shield className="h-8 w-8 mb-2 opacity-40 text-indigo-400" />
                  <p className="font-medium text-gray-400">Pre-Flight Review Ready</p>
                  <p className="text-[11px] text-gray-500 mt-1">
                    Review your bracketed context and sanitized diff on the left, then click "Dispatch to AI" below.
                  </p>
                </div>
              )}

              {!loading && response && response.text}
            </div>
          </div>
        </div>

        {/* Modal Footer */}
        <div className="flex items-center justify-between border-t border-canvas-700 bg-canvas-850 px-4 py-2.5">
          <CostBadge tokens={estimatedTokens} costUSD={estimatedCost} />

          <div className="flex items-center gap-2">
            <button
              onClick={onClose}
              className="rounded px-3 py-1.5 text-xs text-gray-400 hover:text-white transition-colors"
            >
              Cancel
            </button>
            <button
              onClick={handleInvestigate}
              disabled={loading}
              className="flex items-center gap-1.5 rounded bg-indigo-600 px-4 py-1.5 text-xs font-semibold text-white hover:bg-indigo-500 disabled:opacity-50 transition-colors shadow"
            >
              <Send className="h-3.5 w-3.5" />
              <span>{loading ? 'Analyzing...' : 'Dispatch to AI'}</span>
            </button>
          </div>
        </div>
      </div>
    </div>
  );
};

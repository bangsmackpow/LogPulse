import React, { useState, useEffect } from 'react';
import { X, Key, DollarSign, Server, Sliders, Check } from 'lucide-react';
import { useAIStore } from '../stores/useAIStore';

interface SettingsModalProps {
  isOpen: boolean;
  onClose: () => void;
}

export const SettingsModal: React.FC<SettingsModalProps> = ({ isOpen, onClose }) => {
  const { provider, model, setProvider, setModel, setBudget } = useAIStore();

  const [openaiKey, setOpenaiKey] = useState('');
  const [anthropicKey, setAnthropicKey] = useState('');
  const [geminiKey, setGeminiKey] = useState('');
  const [ollamaUrl, setOllamaUrl] = useState('http://localhost:11434');
  const [budgetLimit, setBudgetLimit] = useState('10.00');
  const [savedSuccess, setSavedSuccess] = useState(false);

  useEffect(() => {
    if (isOpen && window.go?.main?.App?.GetSettings) {
      window.go.main.App.GetSettings().then((s) => {
        if (s.provider) setProvider(s.provider);
        if (s.model) setModel(s.model);
        if (s.api_key_openai) setOpenaiKey(s.api_key_openai);
        if (s.api_key_anthropic) setAnthropicKey(s.api_key_anthropic);
        if (s.api_key_gemini) setGeminiKey(s.api_key_gemini);
        if (s.ollama_endpoint) setOllamaUrl(s.ollama_endpoint);
        if (s.monthly_budget_usd) {
          setBudgetLimit(s.monthly_budget_usd);
          setBudget(parseFloat(s.monthly_budget_usd) || 10, 0);
        }
      });
    }
  }, [isOpen, setProvider, setModel, setBudget]);

  if (!isOpen) return null;

  const handleSave = async () => {
    const payload = {
      provider,
      model,
      api_key_openai: openaiKey,
      api_key_anthropic: anthropicKey,
      api_key_gemini: geminiKey,
      ollama_endpoint: ollamaUrl,
      monthly_budget_usd: budgetLimit,
    };

    if (window.go?.main?.App?.SaveSettings) {
      await window.go.main.App.SaveSettings(payload);
    }
    setBudget(parseFloat(budgetLimit) || 10, 0);
    setSavedSuccess(true);
    setTimeout(() => {
      setSavedSuccess(false);
      onClose();
    }, 800);
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm p-4">
      <div className="w-full max-w-lg rounded-xl border border-canvas-600 bg-canvas-800 p-5 shadow-2xl text-xs text-gray-200">
        <div className="flex items-center justify-between border-b border-canvas-700 pb-3 mb-4">
          <div className="flex items-center gap-2 text-sm font-semibold text-white">
            <Sliders className="h-4 w-4 text-indigo-400" />
            <span>Settings & AI Providers</span>
          </div>
          <button onClick={onClose} className="rounded p-1 text-gray-400 hover:text-white">
            <X className="h-4 w-4" />
          </button>
        </div>

        <div className="space-y-4">
          {/* Provider Selection */}
          <div>
            <label className="block font-medium text-gray-300 mb-1">Active AI Provider</label>
            <select
              value={provider}
              onChange={(e) => {
                const val = e.target.value;
                setProvider(val);
                if (val === 'ollama') setModel('llama3');
                if (val === 'openai') setModel('gpt-4o');
                if (val === 'anthropic') setModel('claude-3-5-sonnet-20241022');
                if (val === 'gemini') setModel('gemini-1.5-flash');
              }}
              className="w-full h-8 rounded bg-canvas-900 border border-canvas-600 px-2 text-xs text-white focus:outline-none focus:border-indigo-500"
            >
              <option value="ollama">Ollama (Local / Free)</option>
              <option value="openai">OpenAI (GPT-4o / GPT-4o-mini)</option>
              <option value="anthropic">Anthropic (Claude 3.5 Sonnet)</option>
              <option value="gemini">Google Gemini (Gemini 1.5/2.0)</option>
            </select>
          </div>

          {/* Model Name */}
          <div>
            <label className="block font-medium text-gray-300 mb-1">Model Name</label>
            <input
              type="text"
              value={model}
              onChange={(e) => setModel(e.target.value)}
              className="w-full h-8 rounded bg-canvas-900 border border-canvas-600 px-2.5 text-xs text-white font-mono focus:outline-none focus:border-indigo-500"
            />
          </div>

          {/* Provider Specific Settings */}
          {provider === 'ollama' && (
            <div>
              <label className="block font-medium text-gray-300 mb-1 flex items-center gap-1.5">
                <Server className="h-3.5 w-3.5 text-gray-400" />
                <span>Ollama Endpoint URL</span>
              </label>
              <input
                type="text"
                value={ollamaUrl}
                onChange={(e) => setOllamaUrl(e.target.value)}
                placeholder="http://localhost:11434"
                className="w-full h-8 rounded bg-canvas-900 border border-canvas-600 px-2.5 text-xs text-white font-mono focus:outline-none focus:border-indigo-500"
              />
            </div>
          )}

          {provider === 'openai' && (
            <div>
              <label className="block font-medium text-gray-300 mb-1 flex items-center gap-1.5">
                <Key className="h-3.5 w-3.5 text-gray-400" />
                <span>OpenAI API Key</span>
              </label>
              <input
                type="password"
                value={openaiKey}
                onChange={(e) => setOpenaiKey(e.target.value)}
                placeholder="sk-..."
                className="w-full h-8 rounded bg-canvas-900 border border-canvas-600 px-2.5 text-xs text-white font-mono focus:outline-none focus:border-indigo-500"
              />
            </div>
          )}

          {provider === 'anthropic' && (
            <div>
              <label className="block font-medium text-gray-300 mb-1 flex items-center gap-1.5">
                <Key className="h-3.5 w-3.5 text-gray-400" />
                <span>Anthropic API Key</span>
              </label>
              <input
                type="password"
                value={anthropicKey}
                onChange={(e) => setAnthropicKey(e.target.value)}
                placeholder="sk-ant-..."
                className="w-full h-8 rounded bg-canvas-900 border border-canvas-600 px-2.5 text-xs text-white font-mono focus:outline-none focus:border-indigo-500"
              />
            </div>
          )}

          {provider === 'gemini' && (
            <div>
              <label className="block font-medium text-gray-300 mb-1 flex items-center gap-1.5">
                <Key className="h-3.5 w-3.5 text-gray-400" />
                <span>Google Gemini API Key</span>
              </label>
              <input
                type="password"
                value={geminiKey}
                onChange={(e) => setGeminiKey(e.target.value)}
                placeholder="AIzaSy..."
                className="w-full h-8 rounded bg-canvas-900 border border-canvas-600 px-2.5 text-xs text-white font-mono focus:outline-none focus:border-indigo-500"
              />
            </div>
          )}

          {/* Hard Monthly Budget */}
          <div>
            <label className="block font-medium text-gray-300 mb-1 flex items-center gap-1.5">
              <DollarSign className="h-3.5 w-3.5 text-emerald-400" />
              <span>Hard Monthly Budget Ceiling (USD)</span>
            </label>
            <input
              type="number"
              step="0.5"
              value={budgetLimit}
              onChange={(e) => setBudgetLimit(e.target.value)}
              className="w-full h-8 rounded bg-canvas-900 border border-canvas-600 px-2.5 text-xs text-white font-mono focus:outline-none focus:border-indigo-500"
            />
            <span className="text-[10px] text-gray-500 mt-1 block">
              Outbound AI requests are blocked if accumulated spend exceeds this ceiling.
            </span>
          </div>
        </div>

        <div className="mt-6 flex items-center justify-end gap-2 border-t border-canvas-700 pt-3">
          <button
            onClick={onClose}
            className="rounded px-3 py-1.5 text-xs text-gray-400 hover:text-white transition-colors"
          >
            Cancel
          </button>
          <button
            onClick={handleSave}
            className="flex items-center gap-1.5 rounded bg-indigo-600 px-4 py-1.5 text-xs font-semibold text-white hover:bg-indigo-500 transition-colors"
          >
            {savedSuccess ? <Check className="h-4 w-4" /> : null}
            <span>{savedSuccess ? 'Saved!' : 'Save Settings'}</span>
          </button>
        </div>
      </div>
    </div>
  );
};

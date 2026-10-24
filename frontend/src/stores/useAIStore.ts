import { create } from 'zustand';
import { AIResponse, ScrubMatch } from '../types';

interface AIState {
  isModalOpen: boolean;
  selectedLines: string[];
  contextBefore: number;
  contextAfter: number;
  userNotes: string;
  includeTelemetry: boolean;
  scrubbedPrompt: string;
  scrubMatches: ScrubMatch[];
  isInvestigating: boolean;
  response: AIResponse | null;
  provider: string;
  model: string;
  monthlyBudgetUSD: number;
  monthlySpendUSD: number;

  openModal: (lines: string[]) => void;
  closeModal: () => void;
  setContextBefore: (n: number) => void;
  setContextAfter: (m: number) => void;
  setUserNotes: (notes: string) => void;
  setIncludeTelemetry: (include: boolean) => void;
  setScrubResult: (prompt: string, matches: ScrubMatch[]) => void;
  setIsInvestigating: (loading: boolean) => void;
  setResponse: (resp: AIResponse | null) => void;
  setProvider: (p: string) => void;
  setModel: (m: string) => void;
  setBudget: (budget: number, spend: number) => void;
}

export const useAIStore = create<AIState>((set) => ({
  isModalOpen: false,
  selectedLines: [],
  contextBefore: 15,
  contextAfter: 15,
  userNotes: '',
  includeTelemetry: true,
  scrubbedPrompt: '',
  scrubMatches: [],
  isInvestigating: false,
  response: null,
  provider: 'ollama',
  model: 'llama3',
  monthlyBudgetUSD: 10.0,
  monthlySpendUSD: 0.0,

  openModal: (lines) =>
    set({
      isModalOpen: true,
      selectedLines: lines,
      response: null,
      userNotes: '',
    }),
  closeModal: () => set({ isModalOpen: false, response: null }),
  setContextBefore: (n) => set({ contextBefore: n }),
  setContextAfter: (m) => set({ contextAfter: m }),
  setUserNotes: (userNotes) => set({ userNotes }),
  setIncludeTelemetry: (includeTelemetry) => set({ includeTelemetry }),
  setScrubResult: (scrubbedPrompt, scrubMatches) => set({ scrubbedPrompt, scrubMatches }),
  setIsInvestigating: (isInvestigating) => set({ isInvestigating }),
  setResponse: (response) => set({ response }),
  setProvider: (provider) => set({ provider }),
  setModel: (model) => set({ model }),
  setBudget: (monthlyBudgetUSD, monthlySpendUSD) => set({ monthlyBudgetUSD, monthlySpendUSD }),
}));

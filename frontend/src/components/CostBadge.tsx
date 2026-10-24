import React from 'react';
import { AlertTriangle } from 'lucide-react';
import { useAIStore } from '../stores/useAIStore';

interface CostBadgeProps {
  tokens: number;
  costUSD: number;
}

export const CostBadge: React.FC<CostBadgeProps> = ({ tokens, costUSD }) => {
  const { monthlyBudgetUSD, monthlySpendUSD } = useAIStore();
  const willExceed = monthlySpendUSD + costUSD > monthlyBudgetUSD;

  return (
    <div className="flex items-center gap-2 text-xs">
      <div className="flex items-center gap-1 rounded bg-canvas-900 border border-canvas-700 px-2 py-1 font-mono text-gray-300">
        <span>~{tokens.toLocaleString()} tokens</span>
        <span className="text-gray-500">•</span>
        <span className="text-emerald-400 font-semibold">
          {costUSD === 0 ? 'FREE (Local)' : `$${costUSD.toFixed(5)}`}
        </span>
      </div>

      {willExceed && (
        <div className="flex items-center gap-1 text-amber-400 text-[11px] font-sans">
          <AlertTriangle className="h-3.5 w-3.5" />
          <span>Exceeds monthly limit (${monthlyBudgetUSD.toFixed(2)})</span>
        </div>
      )}
    </div>
  );
};

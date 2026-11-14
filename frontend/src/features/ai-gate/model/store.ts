import { create } from 'zustand';

import type { AiGateDenial } from '@/entities/ai-credit';

interface AiGateStore {
  /** The denial the sheet explains; null = closed. UI state only — the balance itself is a query. */
  denial: AiGateDenial | null;
  open: (denial: AiGateDenial) => void;
  close: () => void;
}

export const useAiGateStore = create<AiGateStore>((set) => ({
  denial: null,
  open: (denial) => set({ denial }),
  close: () => set({ denial: null }),
}));

/** Opens the global gate sheet for `denial` (callable outside React). */
export function openAiGate(denial: AiGateDenial): void {
  useAiGateStore.getState().open(denial);
}

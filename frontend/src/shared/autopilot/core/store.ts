'use client';

import { create } from 'zustand';

import { EMPTY_CATALOG, type PilotCatalog } from '@/shared/pilot';

/** What the overlay paints. UI state only (§8); no script values ever land here except the `say` caption. */

export type PilotStatus = 'idle' | 'running' | 'paused';

export interface PilotBox {
  x: number;
  y: number;
  w: number;
  h: number;
}

export interface PilotPointer {
  x: number;
  y: number;
  /** Duration of the move that brings it here. */
  travelMs: number;
}

interface AutopilotState {
  status: PilotStatus;
  step: number;
  total: number;
  caption: string | null;
  pointer: PilotPointer | null;
  ring: PilotBox | null;
  /** Bumped per press, so the ripple animation restarts. */
  press: number;
  /** A `confirmTargets` step waiting for the user's answer. */
  confirm: { targetId: string; answer: (allowed: boolean) => void } | null;
  /** Whether the user touching the screen should interrupt (off for `interrupt.onUserInput: 'ignore'`). */
  shield: boolean;
  set: (patch: Partial<Omit<AutopilotState, 'set' | 'reset'>>) => void;
  reset: () => void;
}

const IDLE = {
  status: 'idle' as PilotStatus,
  step: 0,
  total: 0,
  caption: null,
  pointer: null,
  ring: null,
  press: 0,
  confirm: null,
  shield: false,
};

export const useAutopilotStore = create<AutopilotState>((set) => ({
  ...IDLE,
  set: (patch) => set(patch),
  reset: () => set(IDLE),
}));

/** Run state for a screen that starts scripts (a progress line, a disabled button). */
export function useAutopilotStatus(): { status: PilotStatus; step: number; total: number } {
  const status = useAutopilotStore((s) => s.status);
  const step = useAutopilotStore((s) => s.step);
  const total = useAutopilotStore((s) => s.total);
  return { status, step, total };
}

/** The catalog the mounted host was given — kept apart from the run state, which is wiped after every run. */
export const useAutopilotCatalogStore = create<{ catalog: PilotCatalog }>(() => ({ catalog: EMPTY_CATALOG }));

/** The app's catalog (empty until `AutopilotHost` mounts). For building the agent manifest. */
export function useAutopilotCatalog(): PilotCatalog {
  return useAutopilotCatalogStore((s) => s.catalog);
}

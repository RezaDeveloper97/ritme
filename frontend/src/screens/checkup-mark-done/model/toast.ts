import { create } from 'zustand';

/**
 * The «ثبت شد» toast outlives the sheet that raises it (the sheet closes on
 * success), so its text lives here and `MarkDoneToast` — mounted by the route —
 * shows it. UI-only state (§8). Holds UI copy only, never health data.
 */
interface ToastState {
  message: string | null;
  tone: 'ok' | 'warn';
  show: (message: string, tone?: 'ok' | 'warn') => void;
  clear: () => void;
}

export const useMarkDoneToast = create<ToastState>((set) => ({
  message: null,
  tone: 'ok',
  show: (message, tone = 'ok') => set({ message, tone }),
  clear: () => set({ message: null }),
}));

import { create } from 'zustand';

/** The skip the user can still undo today. UI-only state (§8); only the date string lives in storage. */
interface RecentSkip {
  expected: string;
  day: string;
}

const KEY = 'ritme_skip_undo';

function read(): RecentSkip | null {
  try {
    const raw = window.localStorage.getItem(KEY);
    const v = raw ? (JSON.parse(raw) as Partial<RecentSkip>) : null;
    return v && typeof v.expected === 'string' && typeof v.day === 'string' ? { expected: v.expected, day: v.day } : null;
  } catch {
    return null;
  }
}

function write(v: RecentSkip | null): void {
  try {
    if (v) window.localStorage.setItem(KEY, JSON.stringify(v));
    else window.localStorage.removeItem(KEY);
  } catch {
    // storage unavailable: the undo link only lives as long as the toast
  }
}

interface SkipState {
  recent: RecentSkip | null;
  toast: boolean;
  /** Call from an effect: reads the stored skip once on the client. */
  hydrate: () => void;
  record: (expected: string, day: string) => void;
  hideToast: () => void;
  clear: () => void;
}

export const useSkipStore = create<SkipState>((set, get) => ({
  recent: null,
  toast: false,
  hydrate: () => {
    if (!get().recent) set({ recent: read() });
  },
  record: (expected, day) => {
    const recent = { expected, day };
    write(recent);
    set({ recent, toast: true });
  },
  hideToast: () => set({ toast: false }),
  clear: () => {
    write(null);
    set({ recent: null, toast: false });
  },
}));

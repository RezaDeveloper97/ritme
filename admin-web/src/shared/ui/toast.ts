'use client';

import { create } from 'zustand';

export type ToastTone = 'success' | 'error' | 'info';
export interface ToastItem {
  id: number;
  tone: ToastTone;
  text: string;
}

interface ToastState {
  items: ToastItem[];
  push: (tone: ToastTone, text: string) => void;
  dismiss: (id: number) => void;
}

let seq = 0;

export const useToastStore = create<ToastState>((set, get) => ({
  items: [],
  push: (tone, text) => {
    const id = ++seq;
    // Newest last; at most 4 on screen.
    set({ items: [...get().items, { id, tone, text }].slice(-4) });
    window.setTimeout(() => get().dismiss(id), tone === 'error' ? 7000 : 4000);
  },
  dismiss: (id) => set({ items: get().items.filter((t) => t.id !== id) }),
}));

/** Callable from anywhere (event handlers, mutation callbacks). Text must already be translated. */
export const toast = {
  success: (text: string) => useToastStore.getState().push('success', text),
  error: (text: string) => useToastStore.getState().push('error', text),
  info: (text: string) => useToastStore.getState().push('info', text),
};

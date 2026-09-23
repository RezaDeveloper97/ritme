'use client';

import { create } from 'zustand';

export interface ConfirmRequest {
  title?: string;
  message: string;
  confirmLabel?: string;
  /** `danger` for destructive actions (delete, block). */
  tone?: 'default' | 'danger';
}

interface ConfirmState {
  current: (ConfirmRequest & { resolve: (ok: boolean) => void }) | null;
  open: (req: ConfirmRequest) => Promise<boolean>;
  close: (ok: boolean) => void;
}

export const useConfirmStore = create<ConfirmState>((set, get) => ({
  current: null,
  open: (req) =>
    new Promise<boolean>((resolve) => {
      get().current?.resolve(false);
      set({ current: { ...req, resolve } });
    }),
  close: (ok) => {
    get().current?.resolve(ok);
    set({ current: null });
  },
}));

/** `if (await confirm({ message: t('confirmDelete'), tone: 'danger' })) mutate()` */
export const confirm = (req: ConfirmRequest): Promise<boolean> => useConfirmStore.getState().open(req);

'use client';

import { create } from 'zustand';

import type { SheetTarget } from './types';

interface SheetStore {
  /**
   * Open sheets, innermost last. A stack rather than a single slot because a
   * sheet can legitimately open another (the article library opens an article),
   * and dismissing that one must return the reader to the list they were
   * browsing — not to the screen three levels down.
   */
  stack: SheetTarget[];
  /**
   * Replaces the stack **without touching history**. History is owned by
   * `controller.ts`; keeping the two apart is what lets `popstate` feed the
   * store without the store pushing a new entry back and looping.
   */
  setStack: (stack: SheetTarget[]) => void;
}

export const useSheetStore = create<SheetStore>((set) => ({
  stack: [],
  setStack: (stack) => set({ stack }),
}));

/** The sheet actually on screen, or `null` when the screen stands alone. */
export function topOf(stack: SheetTarget[]): SheetTarget | null {
  return stack.length > 0 ? stack[stack.length - 1] : null;
}

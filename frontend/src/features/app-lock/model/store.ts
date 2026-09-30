'use client';

import { useSyncExternalStore } from 'react';

import { onSessionEnd } from '@/shared/session';

import type { KeyValueStore } from './config';
import { createLockController, type LockController, type LockSnapshot } from './controller';

/**
 * The page's single lock controller, created on first use in the browser.
 * Never persisted: a reload creates a new one, which starts locked (see
 * controller.ts). On the server there is none — the gate renders its
 * «undecided» state and the pre-paint script keeps the shell hidden.
 */
let controller: LockController | null = null;

const memoryStore = (): KeyValueStore => {
  const m = new Map<string, string>();
  return {
    getItem: (k) => m.get(k) ?? null,
    setItem: (k, v) => void m.set(k, v),
    removeItem: (k) => void m.delete(k),
  };
};

function browserStorage(): KeyValueStore {
  try {
    return window.localStorage;
  } catch {
    return memoryStore(); // blocked storage: a lock can't be kept, so none is configured
  }
}

export function getLockController(): LockController | null {
  if (typeof window === 'undefined') return null;
  controller ??= createLockController({ storage: browserStorage(), now: () => Date.now() });
  return controller;
}

// The lock belongs to the signed-in session: logout, account deletion or a
// revoked token wipes it, so the next account on this device starts unlocked.
onSessionEnd(() => {
  const c = getLockController();
  if (c) c.reset();
});

const noop = () => () => undefined;

/** The lock state; `null` until mounted in the browser (server render, hydration). */
export function useAppLock(): LockSnapshot | null {
  return useSyncExternalStore(
    (cb) => getLockController()?.subscribe(cb) ?? noop(),
    () => getLockController()?.getSnapshot() ?? null,
    () => null,
  );
}

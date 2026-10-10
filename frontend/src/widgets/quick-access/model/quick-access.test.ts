import { beforeEach, describe, expect, it } from 'vitest';

import { availableShortcuts, QUICK_ACCESS_MAX, visibleShortcuts } from './shortcuts';
import { QUICK_ACCESS_KEY, readQuickAccess, useQuickAccessStore, writeQuickAccess } from './store';

const mem = new Map<string, string>();
Object.defineProperty(globalThis, 'window', {
  value: {
    localStorage: {
      getItem: (k: string) => mem.get(k) ?? null,
      setItem: (k: string, v: string) => void mem.set(k, v),
      removeItem: (k: string) => void mem.delete(k),
    },
  },
  configurable: true,
});

const ids = (list: { id: string }[]) => list.map((s) => s.id);

describe('quick access shortcuts (CB-NAV-03)', () => {
  it('never offers a shortcut whose screen has not shipped', () => {
    for (const mode of ['cycle', 'ttc', 'ivf', 'pregnancy', 'postpartum', 'menopause', 'teen'] as const) {
      expect(availableShortcuts(mode).every((s) => s.href && s.href.startsWith('/'))).toBe(true);
      expect(ids(availableShortcuts(mode))).not.toContain('doctor');
    }
  });

  it('keeps mode-only shortcuts in their mode, and doctor/labs out of teen', () => {
    expect(ids(availableShortcuts('menopause'))).toContain('hotFlash');
    expect(ids(availableShortcuts('cycle'))).not.toContain('hotFlash');
    expect(ids(availableShortcuts('ivf'))).toContain('ivfMeds');
    expect(ids(availableShortcuts('teen'))).not.toContain('labs');
    expect(ids(availableShortcuts('teen'))).not.toContain('contraception');
  });

  it('defaults to four of the board picks the app can open', () => {
    expect(ids(visibleShortcuts('cycle', null))).toEqual(['record', 'painDiary', 'courses', 'checkups']);
    expect(ids(visibleShortcuts('menopause', null))[0]).toBe('hotFlash');
  });

  it('shows the stored picks in their order, drops unavailable ones, caps at the max', () => {
    expect(ids(visibleShortcuts('cycle', ['todo', 'hotFlash', 'vitals']))).toEqual(['todo', 'vitals']);
    expect(ids(visibleShortcuts('cycle', []))).toEqual([]);
    expect(ids(visibleShortcuts('cycle', ['hotFlash']))).toEqual(ids(visibleShortcuts('cycle', null)));
    const all = ids(availableShortcuts('cycle'));
    expect(visibleShortcuts('cycle', all).length).toBe(Math.min(all.length, QUICK_ACCESS_MAX));
  });
});

describe('quick access storage', () => {
  beforeEach(() => {
    mem.clear();
    useQuickAccessStore.setState({ owner: null, ids: null });
  });

  it('round-trips per account and drops another account’s list', () => {
    writeQuickAccess('1', ['todo', 'record']);
    expect(readQuickAccess('1')).toEqual(['todo', 'record']);
    expect(readQuickAccess('2')).toBeNull();
    expect(mem.has(QUICK_ACCESS_KEY)).toBe(false);
  });

  it('ignores garbage', () => {
    mem.set(QUICK_ACCESS_KEY, '{nope');
    expect(readQuickAccess('1')).toBeNull();
    mem.set(QUICK_ACCESS_KEY, JSON.stringify({ owner: '1', ids: ['ok', 3, '<script>'] }));
    expect(readQuickAccess('1')).toEqual(['ok']);
  });

  it('the store loads once per owner, saves and resets to defaults', () => {
    writeQuickAccess('1', ['todo']);
    useQuickAccessStore.getState().load('1');
    expect(useQuickAccessStore.getState().ids).toEqual(['todo']);
    useQuickAccessStore.getState().save('1', ['vitals']);
    expect(readQuickAccess('1')).toEqual(['vitals']);
    useQuickAccessStore.getState().save('1', null);
    expect(mem.has(QUICK_ACCESS_KEY)).toBe(false);
    expect(useQuickAccessStore.getState().ids).toBeNull();
  });
});

import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

const sessionEnd = vi.hoisted(() => ({ callbacks: [] as (() => void)[] }));

vi.mock('@/shared/session', () => ({
  onSessionEnd: (cb: () => void) => {
    sessionEnd.callbacks.push(cb);
    return () => undefined;
  },
}));

function fakeStorage() {
  const store = new Map<string, string>();
  const localStorage = {
    getItem: (k: string) => store.get(k) ?? null,
    setItem: (k: string, v: string) => void store.set(k, v),
    removeItem: (k: string) => void store.delete(k),
  };
  vi.stubGlobal('window', { localStorage });
  return store;
}

async function load() {
  vi.resetModules();
  sessionEnd.callbacks = [];
  return import('./ttc-hint');
}

describe('home TTC layout hint', () => {
  let store: Map<string, string>;

  beforeEach(() => {
    store = fakeStorage();
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('is unknown until the profile decided once, then remembers only the boolean', async () => {
    const hint = await load();
    expect(hint.readTtcHint()).toBeNull();
    hint.writeTtcHint(true);
    expect(hint.readTtcHint()).toBe(true);
    hint.writeTtcHint(false);
    expect(hint.readTtcHint()).toBe(false);
    expect([...store.values()]).toEqual(['0']);
  });

  it('is wiped when the session ends', async () => {
    const hint = await load();
    hint.writeTtcHint(true);
    expect(sessionEnd.callbacks).toHaveLength(1);
    sessionEnd.callbacks[0]();
    expect(hint.readTtcHint()).toBeNull();
    expect(store.size).toBe(0);
  });

  it('degrades to unknown when storage is blocked', async () => {
    vi.stubGlobal('window', {
      get localStorage(): Storage {
        throw new Error('blocked');
      },
    });
    const hint = await load();
    expect(() => hint.writeTtcHint(true)).not.toThrow();
    expect(hint.readTtcHint()).toBeNull();
  });
});

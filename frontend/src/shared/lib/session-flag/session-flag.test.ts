import { afterEach, describe, expect, it, vi } from 'vitest';

import { readSessionFlag, writeSessionFlag } from './session-flag';

describe('session flags', () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('reads back what was written, per key', () => {
    const store = new Map<string, string>();
    vi.stubGlobal('window', {
      sessionStorage: {
        getItem: (k: string) => store.get(k) ?? null,
        setItem: (k: string, v: string) => void store.set(k, v),
        removeItem: (k: string) => void store.delete(k),
      },
    });
    expect(readSessionFlag('a')).toBe(false);
    writeSessionFlag('a');
    expect(readSessionFlag('a')).toBe(true);
    expect(readSessionFlag('b')).toBe(false);
    writeSessionFlag('a', false);
    expect(readSessionFlag('a')).toBe(false);
  });

  it('reads «not set» when storage throws', () => {
    vi.stubGlobal('window', {
      sessionStorage: {
        getItem: () => {
          throw new Error('blocked');
        },
        setItem: () => {
          throw new Error('blocked');
        },
        removeItem: () => undefined,
      },
    });
    expect(() => writeSessionFlag('a')).not.toThrow();
    expect(readSessionFlag('a')).toBe(false);
  });
});

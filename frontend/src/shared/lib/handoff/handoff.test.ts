import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

function fakeSessionStorage(): Storage {
  const map = new Map<string, string>();
  return {
    get length() {
      return map.size;
    },
    clear: () => map.clear(),
    getItem: (k) => map.get(k) ?? null,
    key: (i) => [...map.keys()][i] ?? null,
    removeItem: (k) => void map.delete(k),
    setItem: (k, v) => void map.set(k, String(v)),
  };
}

async function load() {
  vi.resetModules();
  return import('./handoff');
}

describe('handoff', () => {
  beforeEach(() => {
    vi.stubGlobal('sessionStorage', fakeSessionStorage());
  });
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('puts only a random id in the URL, never the data', async () => {
    const { withHandoff, readHandoff } = await load();
    const href = withHandoff('/reminders/appointment/new?kind=in_person', {
      title: 'NT scan',
      topic: 'ultrasound',
      date: '2026-10-05',
    });
    const url = new URL(href, 'https://x');
    expect([...url.searchParams.keys()]).toEqual(['kind', 'prefill']);
    expect(href).not.toMatch(/NT|scan|ultrasound|2026/);
    expect(readHandoff(url.searchParams.get('prefill'))).toEqual({
      title: 'NT scan',
      topic: 'ultrasound',
      date: '2026-10-05',
    });
  });

  it('survives a reload (sessionStorage) but not an unknown or replaced id', async () => {
    const first = await load();
    const id = first.stashHandoff({ title: 'a' });
    const reloaded = await load(); // fresh module = fresh memory, same tab storage
    expect(reloaded.readHandoff(id)).toEqual({ title: 'a' });
    const next = reloaded.stashHandoff({ title: 'b' });
    expect(reloaded.readHandoff(id)).toBeNull();
    expect(reloaded.readHandoff(next)).toEqual({ title: 'b' });
    expect(reloaded.readHandoff('nope')).toBeNull();
    expect(reloaded.readHandoff(null)).toBeNull();
  });

  it('clearHandoff drops it everywhere', async () => {
    const { stashHandoff, readHandoff, clearHandoff } = await load();
    const id = stashHandoff({ title: 'a' });
    clearHandoff();
    expect(readHandoff(id)).toBeNull();
    expect(sessionStorage.getItem('ritme_handoff')).toBeNull();
  });

  it('works in memory when storage is blocked', async () => {
    vi.stubGlobal('sessionStorage', {
      getItem: () => {
        throw new Error('blocked');
      },
      setItem: () => {
        throw new Error('blocked');
      },
      removeItem: () => {
        throw new Error('blocked');
      },
    });
    const { stashHandoff, readHandoff } = await load();
    const id = stashHandoff({ title: 'a' });
    expect(readHandoff(id)).toEqual({ title: 'a' });
  });

  it('ignores tampered storage', async () => {
    const { readHandoff } = await load();
    sessionStorage.setItem('ritme_handoff', JSON.stringify({ id: 'abcdefgh12', data: { title: 1, topic: 'lab' } }));
    expect(readHandoff('abcdefgh12')).toEqual({ topic: 'lab' });
    sessionStorage.setItem('ritme_handoff', '{broken');
    expect(readHandoff('abcdefgh12')).toBeNull();
  });
});

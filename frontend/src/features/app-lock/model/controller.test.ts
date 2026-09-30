import { describe, expect, it } from 'vitest';

import { appLockInitScript, type KeyValueStore, LOCK_KEY } from './config';
import { backoffMs, createLockController, FREE_ATTEMPTS, MAX_FAILURES } from './controller';

function memory(): KeyValueStore & { dump: () => Record<string, string> } {
  const m = new Map<string, string>();
  return {
    getItem: (k) => m.get(k) ?? null,
    setItem: (k, v) => void m.set(k, v),
    removeItem: (k) => void m.delete(k),
    dump: () => Object.fromEntries(m),
  };
}

function clock(start = 1_790_000_000_000) {
  let t = start;
  return { now: () => t, advance: (ms: number) => (t += ms) };
}

/** A "page load": a new controller over the same device storage (what a reload / new tab does). */
function load(storage: KeyValueStore, c = clock()) {
  return createLockController({ storage, now: c.now });
}

describe('app lock — reload and back navigation cannot bypass it', () => {
  it('a reload after unlocking starts locked again', async () => {
    const storage = memory();
    const first = load(storage);
    await first.enable({ passcode: '2580' });
    expect(first.getSnapshot()).toMatchObject({ enabled: true, locked: false });

    const afterReload = load(storage);
    expect(afterReload.getSnapshot()).toMatchObject({ enabled: true, locked: true });

    expect(await afterReload.unlockWithPasscode('2580')).toBe('ok');
    expect(afterReload.getSnapshot().locked).toBe(false);
    // …and the next reload locks again: «unlocked» was never persisted.
    expect(load(storage).getSnapshot().locked).toBe(true);
  });

  it('never persists an unlocked flag anywhere', async () => {
    const storage = memory();
    const c = load(storage);
    await c.enable({ passcode: '135790' });
    await load(storage).unlockWithPasscode('135790');
    const stored = storage.dump();
    expect(Object.keys(stored)).toEqual([LOCK_KEY]);
    const config = JSON.parse(stored[LOCK_KEY] ?? '{}') as Record<string, unknown>;
    expect(Object.keys(config).sort()).toEqual(
      ['blockedUntil', 'credentialId', 'failures', 'hash', 'iterations', 'length', 'salt', 'timeoutMin', 'v'].sort(),
    );
    expect(JSON.stringify(config)).not.toContain('135790');
  });

  it('history / route changes do not touch the lock; only a correct passcode does', async () => {
    const storage = memory();
    await load(storage).enable({ passcode: '1111' });
    const c = load(storage);
    // Back/forward and client navigation happen above the gate and never call
    // the controller; a bfcache restore arrives as pagehide → pageshow.
    c.onHidden();
    c.onVisible();
    expect(c.getSnapshot().locked).toBe(true);
    expect(await c.unlockWithPasscode('2222')).toBe('wrong');
    expect(c.getSnapshot().locked).toBe(true);
  });

  it('a corrupted config still locks, and no passcode opens it', async () => {
    const storage = memory();
    storage.setItem(LOCK_KEY, '{"v":1,"locked":false');
    const c = load(storage);
    expect(c.getSnapshot()).toMatchObject({ enabled: true, locked: true });
    expect(await c.unlockWithPasscode('0000')).toBe('wrong');
    expect(c.getSnapshot().locked).toBe(true);
  });
});

describe('app lock — resume after N minutes', () => {
  it('re-locks only after the configured time in the background', async () => {
    const storage = memory();
    const t = clock();
    const c = load(storage, t);
    await c.enable({ passcode: '4321', timeoutMin: 5 });

    c.onHidden();
    t.advance(4 * 60_000);
    c.onVisible();
    expect(c.getSnapshot().locked).toBe(false);

    c.onHidden();
    t.advance(5 * 60_000);
    c.onVisible();
    expect(c.getSnapshot().locked).toBe(true);
  });

  it('timeout 0 locks on every resume', async () => {
    const storage = memory();
    const c = load(storage);
    await c.enable({ passcode: '4321', timeoutMin: 0 });
    c.onHidden();
    c.onVisible();
    expect(c.getSnapshot().locked).toBe(true);
  });

  it('tracks hidden for the preview veil', async () => {
    const c = load(memory());
    c.setHidePreview(true);
    c.onHidden();
    expect(c.getSnapshot()).toMatchObject({ hidePreview: true, hidden: true });
    c.onVisible();
    expect(c.getSnapshot().hidden).toBe(false);
  });
});

describe('app lock — wrong passcodes', () => {
  it('backs off after the free attempts, and a reload does not reset it', async () => {
    const storage = memory();
    const t = clock();
    await load(storage, t).enable({ passcode: '9999' });
    const c = load(storage, t);
    for (let i = 1; i < FREE_ATTEMPTS; i++) expect(await c.unlockWithPasscode('0000')).toBe('wrong');
    expect(await c.unlockWithPasscode('0000')).toBe('blocked');

    const reloaded = load(storage, t);
    expect(reloaded.getSnapshot().blockedUntil).toBeGreaterThan(t.now());
    expect(await reloaded.unlockWithPasscode('9999')).toBe('blocked');

    t.advance(backoffMs(FREE_ATTEMPTS));
    expect(await reloaded.unlockWithPasscode('9999')).toBe('ok');
  });

  it('doubles the wait up to a cap', () => {
    expect(backoffMs(FREE_ATTEMPTS - 1)).toBe(0);
    expect(backoffMs(FREE_ATTEMPTS)).toBe(30_000);
    expect(backoffMs(FREE_ATTEMPTS + 1)).toBe(60_000);
    expect(backoffMs(FREE_ATTEMPTS + 10)).toBe(5 * 60_000);
  });
});

describe('app lock — failures across tabs and the lock-out', () => {
  it('re-reads the stored config before checking: failures counted in another tab still count', async () => {
    const storage = memory();
    const t = clock();
    await load(storage, t).enable({ passcode: '9999' });
    const tabA = load(storage, t);
    const tabB = load(storage, t);
    for (let i = 1; i < FREE_ATTEMPTS; i++) await tabA.unlockWithPasscode('0000');
    // Tab B's in-memory copy is stale (0 failures); the check must use the stored count.
    expect(await tabB.unlockWithPasscode('0000')).toBe('blocked');
  });

  it('a passcode changed in another tab is the one that unlocks', async () => {
    const storage = memory();
    const old = load(storage);
    await old.enable({ passcode: '1111' });
    const stale = load(storage);
    await load(storage).enable({ passcode: '2222' });
    expect(await stale.unlockWithPasscode('1111')).toBe('wrong');
    expect(await stale.unlockWithPasscode('2222')).toBe('ok');
  });

  it(`after ${MAX_FAILURES} wrong passcodes the app is locked out, even after a reload and with the right code`, async () => {
    const storage = memory();
    const t = clock();
    await load(storage, t).enable({ passcode: '9999' });
    const c = load(storage, t);
    let last = '';
    for (let i = 0; i < MAX_FAILURES; i++) {
      t.advance(backoffMs(i) + 1); // wait out any back-off so each try is counted
      last = await c.unlockWithPasscode('0000');
    }
    expect(last).toBe('lockedOut');
    expect(c.getSnapshot()).toMatchObject({ locked: true, lockedOut: true });

    const reloaded = load(storage, t);
    expect(reloaded.getSnapshot().lockedOut).toBe(true);
    t.advance(10 * 60_000);
    expect(await reloaded.unlockWithPasscode('9999')).toBe('lockedOut');
  });

  it('wrong passcodes while turning the lock off count too, and lock the app out', async () => {
    const storage = memory();
    const t = clock();
    const c = load(storage, t);
    await c.enable({ passcode: '2468' });
    for (let i = 0; i < MAX_FAILURES; i++) {
      t.advance(backoffMs(i) + 1);
      expect(await c.verifyPasscode('1357')).toBe(false);
    }
    expect(c.getSnapshot()).toMatchObject({ locked: true, lockedOut: true });
    expect(await c.verifyPasscode('2468')).toBe(false);
  });
});

describe('app lock — turning it off and session end', () => {
  it('verifyPasscode gates disable; reset wipes the config', async () => {
    const storage = memory();
    const c = load(storage);
    await c.enable({ passcode: '2468' });
    expect(await c.verifyPasscode('1357')).toBe(false);
    expect(await c.verifyPasscode('2468')).toBe(true);
    c.reset();
    expect(storage.getItem(LOCK_KEY)).toBeNull();
    expect(load(storage).getSnapshot()).toMatchObject({ enabled: false, locked: false });
  });

  it('rejects passcodes outside 4–6 digits', async () => {
    const c = load(memory());
    await expect(c.enable({ passcode: '123' })).rejects.toThrow();
    await expect(c.enable({ passcode: '12a4' })).rejects.toThrow();
  });
});

describe('appLockInitScript', () => {
  function run(stored: Record<string, string>) {
    const attrs: Record<string, string> = {};
    const fakeWindow = {
      localStorage: { getItem: (k: string) => stored[k] ?? null },
      document: { documentElement: { setAttribute: (k: string, v: string) => void (attrs[k] = v) } },
    };
    new Function('localStorage', 'document', appLockInitScript)(fakeWindow.localStorage, fakeWindow.document);
    return attrs;
  }

  it('marks <html> before paint only when a lock is configured', () => {
    expect(run({ [LOCK_KEY]: '{}' })).toHaveProperty('data-app-locked');
    expect(run({})).not.toHaveProperty('data-app-locked');
  });
});

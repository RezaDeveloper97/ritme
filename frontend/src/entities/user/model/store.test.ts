import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

/**
 * The persisted onboarding answers are personal data (name, mobile, birth
 * date, weight, height, pregnancy dates). Every session end must remove them
 * from the device, without touching device preferences (CLAUDE.md §11.1).
 */

const ONBOARDING_KEY = 'ritme-onboarding';

function fakeBrowser() {
  const store = new Map<string, string>();
  const localStorage = {
    getItem: (k: string) => store.get(k) ?? null,
    setItem: (k: string, v: string) => void store.set(k, v),
    removeItem: (k: string) => void store.delete(k),
    clear: vi.fn(() => store.clear()),
    key: (i: number) => [...store.keys()][i] ?? null,
    get length() {
      return store.size;
    },
  };
  let cookies = '';
  vi.stubGlobal('localStorage', localStorage);
  vi.stubGlobal('window', { localStorage, dispatchEvent: vi.fn() });
  vi.stubGlobal('document', {
    get cookie() {
      return cookies;
    },
    set cookie(line: string) {
      cookies = line;
    },
  });
  vi.stubGlobal('fetch', vi.fn(() => Promise.resolve(new Response(null, { status: 204 }))));
  return { store, localStorage };
}

async function load() {
  vi.resetModules();
  const user = await import('./store');
  const session = await import('@/shared/session');
  return { ...user, ...session };
}

describe('onboarding store and session end', () => {
  let env: ReturnType<typeof fakeBrowser>;

  beforeEach(() => {
    env = fakeBrowser();
  });
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('clearAuthToken removes the persisted answers and resets the store', async () => {
    const { useOnboardingStore, resetOnboardingFor, setAuthToken, clearAuthToken } = await load();
    env.store.set('ritme_theme', 'dark');
    setAuthToken('a.b.c');
    resetOnboardingFor(7);
    const s = useOnboardingStore.getState();
    s.setName('Sara');
    s.setPhone('09120000001');
    s.setWeight(62.5);
    s.setPregnancyBasis({ source: 'lmp', lmp: { year: 1405, month: 5, day: 2 } });
    expect(env.store.get(ONBOARDING_KEY)).toContain('Sara');

    clearAuthToken();

    expect(env.store.has(ONBOARDING_KEY)).toBe(false);
    expect(env.store.has('ritme_token')).toBe(false);
    const after = useOnboardingStore.getState();
    expect(after.userId).toBeNull();
    expect(after.name).toBe('');
    expect(after.phone).toBe('');
    expect(after.weight).toBe(60);
    expect(after.pregnancyBasis.lmp).toBeNull();
    // Device preferences stay, and nothing wipes the whole origin.
    expect(env.store.get('ritme_theme')).toBe('dark');
    expect(env.localStorage.clear).not.toHaveBeenCalled();
  });

  it('a calendar sync that changes nothing does not re-create the removed key', async () => {
    const { useOnboardingStore, clearAuthToken } = await load();
    useOnboardingStore.getState().setName('Sara');
    clearAuthToken();

    useOnboardingStore.getState().syncCalendar('fa');

    expect(env.store.has(ONBOARDING_KEY)).toBe(false);
  });

  it('a calendar sync that does change something still converts the dates', async () => {
    const { useOnboardingStore } = await load();
    useOnboardingStore.getState().syncCalendar('en');

    const s = useOnboardingStore.getState();
    expect(s.locale).toBe('en');
    // 25/10/1373 Jalali is 15 January 1995.
    expect(s.birth).toEqual({ y: 1995, m: 1, d: 15 });
  });

  it('the next account starts from empty answers after a logout', async () => {
    const { useOnboardingStore, resetOnboardingFor, clearAuthToken } = await load();
    resetOnboardingFor(7);
    useOnboardingStore.getState().setName('Sara');
    clearAuthToken();

    resetOnboardingFor(8);

    expect(useOnboardingStore.getState().name).toBe('');
    expect(useOnboardingStore.getState().userId).toBe(8);
  });
});

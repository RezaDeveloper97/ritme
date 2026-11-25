import { describe, expect, it, vi } from 'vitest';

import { createBackStack, LAYER_STATE_KEY, nextTrail, type BackStackHost } from './stack';

const ORIGIN = 'https://web.ritme.app';

/**
 * A browser history in miniature: `pushState`/`replaceState` are synchronous,
 * `go()` is queued and fires `popstate` when flushed — the same ordering a real
 * browser has, which is what the double-back bugs depend on.
 */
function fakeBrowser(start: string, withNavigationApi = true) {
  const entries: { url: string; state: unknown }[] = [{ url: ORIGIN + start, state: null }];
  let index = 0;
  const listeners = new Set<() => void>();
  const tasks: (() => void)[] = [];

  const history = {
    get state() {
      return entries[index].state;
    },
    pushState(data: unknown, _unused: string, url?: string | null) {
      entries.splice(index + 1);
      entries.push({ url: new URL(url ?? entries[index].url, ORIGIN).href, state: data });
      index += 1;
    },
    replaceState(data: unknown, _unused: string, url?: string | null) {
      entries[index] = { url: new URL(url ?? entries[index].url, ORIGIN).href, state: data };
    },
    go(delta: number) {
      tasks.push(() => {
        const target = Math.min(Math.max(index + delta, 0), entries.length - 1);
        if (target === index) return;
        index = target;
        listeners.forEach((l) => l());
      });
    },
  };

  const host: BackStackHost = {
    history,
    location: {
      get href() {
        return entries[index].url;
      },
      get pathname() {
        return new URL(entries[index].url).pathname;
      },
    },
    navigation: withNavigationApi
      ? {
          get currentEntry() {
            return { index };
          },
          entries: () => entries.map((e) => ({ url: e.url })),
        }
      : undefined,
    addEventListener: (_type, l) => listeners.add(l),
    removeEventListener: (_type, l) => listeners.delete(l),
    defer: (fn) => tasks.push(fn),
    now: () => clock.now,
  };
  const clock = { now: 10_000 };

  return {
    host,
    clock,
    /** A router navigation to another screen. */
    navigate(path: string) {
      history.pushState({ __NA: true }, '', path);
    },
    /** The system Back: one step, then everything it sets off. */
    pressBack() {
      history.go(-1);
      this.flush();
    },
    flush() {
      while (tasks.length) tasks.shift()!();
    },
    get path() {
      return new URL(entries[index].url).pathname;
    },
    get length() {
      return entries.length;
    },
    get index() {
      return index;
    },
    get tagged() {
      return entries.filter((e) => (e.state as Record<string, unknown> | null)?.[LAYER_STATE_KEY]).length;
    },
  };
}

/** A component-shaped layer: Back calls `onClose`, which flips it shut and releases. */
function openLayer(stack: ReturnType<typeof createBackStack>) {
  const state = { open: true, closedByBack: 0 };
  const handle = stack.push(() => {
    state.closedByBack += 1;
    state.open = false;
    handle.release(); // the effect cleanup runs right after the close
  });
  return {
    state,
    /** ×, swipe, backdrop, save. */
    closeFromUi() {
      state.open = false;
      handle.release();
    },
  };
}

describe('back stack — the reference flow (home → cycle → log sheet)', () => {
  it('Back closes the sheet, the next Back returns home', () => {
    const b = fakeBrowser('/fa/home');
    const stack = createBackStack(b.host);
    b.navigate('/fa/cycle');
    const sheet = openLayer(stack);

    b.pressBack();
    expect(sheet.state).toEqual({ open: false, closedByBack: 1 });
    expect(b.path).toBe('/fa/cycle');

    b.pressBack();
    expect(b.path).toBe('/fa/home');
  });

  it('a close from inside the sheet leaves no entry behind', () => {
    const b = fakeBrowser('/fa/home');
    const stack = createBackStack(b.host);
    b.navigate('/fa/cycle');
    const sheet = openLayer(stack);
    expect(b.index).toBe(2);

    sheet.closeFromUi();
    b.flush();
    expect(sheet.state.closedByBack).toBe(0);
    expect(b.index).toBe(1);
    expect(b.path).toBe('/fa/cycle');

    // The next Back goes to the previous screen — not dead, not the sheet again.
    b.pressBack();
    expect(b.path).toBe('/fa/home');
  });

  it('a Back that already popped the entry never pops a second one', () => {
    const b = fakeBrowser('/fa/home');
    const stack = createBackStack(b.host);
    b.navigate('/fa/cycle');
    openLayer(stack);

    b.pressBack();
    b.flush();
    expect(b.path).toBe('/fa/cycle');
    expect(b.index).toBe(1);
  });

  it('nested overlays close one per Back', () => {
    const b = fakeBrowser('/fa/cycle');
    const stack = createBackStack(b.host);
    const outer = openLayer(stack);
    const inner = openLayer(stack);

    b.pressBack();
    expect(inner.state.open).toBe(false);
    expect(outer.state.open).toBe(true);

    b.pressBack();
    expect(outer.state.open).toBe(false);
    expect(stack.depth()).toBe(0);
  });

  it('closing both from the UI in one tick pops both entries', () => {
    const b = fakeBrowser('/fa/home');
    const stack = createBackStack(b.host);
    b.navigate('/fa/cycle');
    const outer = openLayer(stack);
    const inner = openLayer(stack);

    outer.closeFromUi(); // not on top: left stale…
    inner.closeFromUi(); // …and skipped right after this one is popped
    b.flush();
    expect(b.index).toBe(1);

    b.pressBack();
    expect(b.path).toBe('/fa/home');
  });
});

describe('back stack — under a URL sheet', () => {
  it('a URL sheet closing itself goes back past the inline sheet inside it', () => {
    const b = fakeBrowser('/fa/cycle');
    const stack = createBackStack(b.host);
    const under = openLayer(stack); // an inline sheet on the screen…
    b.host.history.pushState(null, '', '/fa/cycle?sheet=log'); // …opens a URL sheet…
    const inner = openLayer(stack); // …which opens an inline panel

    expect(stack.layersOnTop()).toBe(1);
    b.host.history.go(-(1 + stack.layersOnTop())); // what closeSheet() does
    b.flush();
    expect(inner.state.open).toBe(false);
    expect(under.state.open).toBe(true);
    expect(b.index).toBe(1);
  });

  it('Back inside the URL sheet closes only the inline panel', () => {
    const b = fakeBrowser('/fa/cycle');
    const stack = createBackStack(b.host);
    const under = openLayer(stack);
    b.host.history.pushState(null, '', '/fa/cycle?sheet=log');
    const inner = openLayer(stack);

    b.pressBack();
    expect(inner.state.open).toBe(false);
    expect(under.state.open).toBe(true);
    expect(b.host.location.href).toContain('sheet=log');
  });
});

describe('back stack — a sheet closed by following a link', () => {
  it('leaves the router alone, and Back steps over the stale entry', () => {
    const b = fakeBrowser('/fa/home');
    const stack = createBackStack(b.host);
    b.navigate('/fa/cycle');
    const sheet = openLayer(stack);

    stack.noteNavigation();
    sheet.closeFromUi();
    b.flush(); // no back(): it would cancel the navigation below
    b.navigate('/fa/articles');
    expect(b.tagged).toBe(1);

    b.pressBack(); // lands on the stale entry, skips to the screen under it
    expect(b.path).toBe('/fa/cycle');
    expect(b.index).toBe(1);

    b.pressBack();
    expect(b.path).toBe('/fa/home');
  });

  it('pops its entry normally once the navigation grace has passed', () => {
    const b = fakeBrowser('/fa/cycle');
    const stack = createBackStack(b.host);
    stack.noteNavigation();
    b.clock.now += 5_000;
    const sheet = openLayer(stack);
    sheet.closeFromUi();
    b.flush();
    expect(b.index).toBe(0);
  });
});

describe('back stack — the header back arrow', () => {
  it.each([true, false])('goes back through history when the parent is behind (Navigation API: %s)', (nav) => {
    const b = fakeBrowser('/fa/home', nav);
    const stack = createBackStack(b.host);
    b.navigate('/fa/cycle');
    const replace = vi.fn();

    stack.back('/fa/home', replace);
    b.flush();
    expect(replace).not.toHaveBeenCalled();
    expect(b.path).toBe('/fa/home');
    expect(b.index).toBe(0); // no forward entry for the system Back to bounce into
  });

  it.each([true, false])('replaces with the parent after a deep link (Navigation API: %s)', (nav) => {
    const b = fakeBrowser('/fa/cycle', nav);
    const stack = createBackStack(b.host);
    const replace = vi.fn();

    stack.back('/fa/home', replace);
    b.flush();
    expect(replace).toHaveBeenCalledOnce();
  });

  it('replaces when a different screen is behind', () => {
    const b = fakeBrowser('/fa/services');
    const stack = createBackStack(b.host);
    b.navigate('/fa/cycle');
    const replace = vi.fn();

    stack.back('/fa/home', replace);
    expect(replace).toHaveBeenCalledOnce();
  });

  it('crosses the screen’s own stale entries', () => {
    const b = fakeBrowser('/fa/home');
    const stack = createBackStack(b.host);
    b.navigate('/fa/cycle');
    const sheet = openLayer(stack);
    stack.noteNavigation();
    sheet.closeFromUi();
    b.flush(); // stale entry stays on top of /cycle

    stack.back('/fa/home', vi.fn());
    b.flush();
    expect(b.path).toBe('/fa/home');
  });
});

describe('nextTrail', () => {
  it('ignores writes that keep the screen', () => {
    expect(nextTrail(['/a'], 'push', '/a')).toEqual(['/a']);
  });

  it('pushes, replaces and pops screens', () => {
    expect(nextTrail(['/a'], 'push', '/b')).toEqual(['/a', '/b']);
    expect(nextTrail(['/a', '/b'], 'replace', '/c')).toEqual(['/a', '/c']);
    expect(nextTrail(['/a', '/b'], 'pop', '/a')).toEqual(['/a']);
  });

  it('treats a pop it cannot place as a replace', () => {
    expect(nextTrail(['/a', '/b'], 'pop', '/z')).toEqual(['/a', '/z']);
  });
});

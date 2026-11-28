/**
 * Back, one layer at a time.
 *
 * The system Back (gesture or three-button bar, in the browser, the installed
 * PWA and the TWA alike) is nothing but `history.back()`, and a TWA gives the
 * page no other hook: there is no reliable JavaScript "back button" event. So
 * everything that Back must close before it leaves the screen is modelled as a
 * history entry of its own:
 *
 * - **Layer** — an overlay opened in place (an inline `AppSheet`). Opening it
 *   pushes exactly ONE entry with the same URL, tagged in `history.state`.
 *   - Back pops that entry → the `popstate` handler closes the layer. It never
 *     calls `back()` itself: the browser already went back.
 *   - The UI closes it (×, swipe, backdrop, Escape, a save) → `release()` pops
 *     the layer's own entry, and the `popstate` that follows is swallowed. So
 *     no stale entry is left for the next Back to stumble over.
 * - URL sheets (`?sheet=…`, `shared/sheet/controller`) follow the same rule
 *   with their own entry; the two never share one.
 *
 * Whatever is not an overlay is a page, and Back between pages is the
 * browser's own: back to the previous screen, and out of the app (the normal
 * Android behaviour) when there is no previous screen.
 *
 * The keyboard is not modelled at all: with the soft keyboard up, Android's
 * IME consumes the first Back before the page ever sees it.
 */

/** Key of the tag a layer entry carries in `history.state`. */
export const LAYER_STATE_KEY = '__ritmeLayer';

/**
 * After a router push/replace (or a link click) a layer that closes is NOT
 * allowed to pop its entry: the router's own history write is still in flight,
 * and a `back()` racing it would cancel the navigation. The entry is left
 * behind as stale instead and skipped on the way back.
 */
const NAVIGATION_GRACE_MS = 1500;

export interface HistoryLike {
  readonly state: unknown;
  pushState(data: unknown, unused: string, url?: string | null): void;
  replaceState(data: unknown, unused: string, url?: string | null): void;
  go(delta: number): void;
}

/** The slice of the Navigation API we read (Chrome, hence the TWA, has it). */
export interface NavigationLike {
  readonly currentEntry: { readonly index: number } | null;
  entries(): ReadonlyArray<{ readonly url: string | null }>;
}

export interface BackStackHost {
  readonly history: HistoryLike;
  readonly location: { readonly href: string; readonly pathname: string };
  readonly navigation?: NavigationLike;
  addEventListener(type: 'popstate', listener: () => void): void;
  removeEventListener(type: 'popstate', listener: () => void): void;
  /** Runs `fn` after the current task — past the commit that closed a layer. */
  defer(fn: () => void): void;
  now(): number;
}

export interface LayerHandle {
  /** The overlay closed from inside the UI: pop its entry if it is still on top. */
  release(): void;
}

export interface BackStack {
  /** Pushes the layer's entry. `onClose` runs when Back pops it. */
  push(onClose: () => void): LayerHandle;
  /**
   * The on-screen back arrow: returns to `parentPathname` through history when
   * that is the screen right behind this one, so the system Back afterwards
   * doesn't bounce back here. Otherwise (a deep link, a notification, another
   * route behind) calls `replace`, so the arrow still has a destination and
   * the system Back still leaves the app from here.
   */
  back(parentPathname: string, replace: () => void): void;
  /**
   * How many open layers sit directly on top of the current entry's URL — what
   * a URL sheet has to go back past to pop its own entry when it closes itself.
   */
  layersOnTop(): number;
  /** A router navigation is starting (see {@link NAVIGATION_GRACE_MS}). */
  noteNavigation(): void;
  /** Number of open layers — for tests and diagnostics. */
  depth(): number;
  dispose(): void;
}

interface Layer {
  id: string;
  href: string;
  pathname: string;
  /** Position of its entry in history, where the Navigation API can tell. */
  navIndex: number | null;
  onClose: () => void;
}

export type TrailEvent = 'push' | 'replace' | 'pop';

/**
 * The pathnames behind the current screen, newest last — the fallback for
 * browsers without the Navigation API. Same-pathname writes (a layer, a sheet,
 * a `?date=` replace) are not screens and leave it alone.
 */
export function nextTrail(trail: readonly string[], event: TrailEvent, pathname: string): string[] {
  const top = trail[trail.length - 1];
  if (top === pathname) return [...trail];
  if (event === 'push') return [...trail, pathname];
  if (event === 'pop' && trail[trail.length - 2] === pathname) return trail.slice(0, -1);
  // A replace, or a pop we can't place (forward, an entry from before a reload).
  return top === undefined ? [pathname] : [...trail.slice(0, -1), pathname];
}

const pathnameOf = (url: string | null | undefined): string | null => {
  if (!url) return null;
  try {
    return new URL(url).pathname;
  } catch {
    return null;
  }
};

const tagOf = (state: unknown): string | null => {
  if (!state || typeof state !== 'object') return null;
  const tag = (state as Record<string, unknown>)[LAYER_STATE_KEY];
  return typeof tag === 'string' ? tag : null;
};

export function createBackStack(host: BackStackHost): BackStack {
  // Tags are unique per page load, so an entry left over from before a reload
  // can never be mistaken for a live layer.
  const session = Math.random().toString(36).slice(2, 10);
  let seq = 0;
  let layers: Layer[] = [];
  let swallow = 0;
  let navigatedAt = Number.NEGATIVE_INFINITY;
  let trail: string[] = [host.location.pathname];

  const isLive = (tag: string | null) => tag !== null && layers.some((l) => l.id === tag);
  const isStale = (tag: string | null) => tag !== null && !isLive(tag);

  /** Steps over an entry whose layer is gone; its `popstate` is swallowed too. */
  const skipStale = () => {
    swallow += 1;
    host.history.go(-1);
  };

  const onPop = () => {
    trail = nextTrail(trail, 'pop', host.location.pathname);
    const tag = tagOf(host.history.state);

    if (swallow > 0) {
      swallow -= 1;
      if (swallow === 0 && isStale(tag)) skipStale();
      return;
    }

    // Every layer whose entry was above the one we landed on was popped by
    // this Back (one, normally; several after `go(-n)`). Positions are exact
    // with the Navigation API; without it the tag decides, which is wrong only
    // for a layer under a URL sheet under another layer.
    const index = host.navigation?.currentEntry?.index;
    const kept =
      index !== undefined
        ? layers.filter((l) => l.navIndex === null || l.navIndex <= index)
        : layers.slice(0, (tag === null ? -1 : layers.findIndex((l) => l.id === tag)) + 1);
    const popped = layers.filter((l) => !kept.includes(l)).reverse();
    layers = kept;
    for (const layer of popped) layer.onClose();

    // Landed on the entry of a layer that closed while a navigation was in
    // flight (or before a reload): it shows the same screen as the entry under
    // it, so stopping here would make Back look dead.
    if (isStale(tag)) skipStale();
  };

  // History is a global, so a write by anyone (Next's router, the sheet
  // controller) must keep the trail current. Wrapped, never replaced: Next
  // patches these too and relies on calling through.
  const { pushState, replaceState } = host.history;
  const wrap =
    (original: HistoryLike['pushState'], event: TrailEvent) =>
    (...args: Parameters<HistoryLike['pushState']>) => {
      original.apply(host.history, args);
      trail = nextTrail(trail, event, host.location.pathname);
    };
  const history = host.history as { -readonly [K in keyof HistoryLike]: HistoryLike[K] };
  history.pushState = wrap(pushState, 'push');
  history.replaceState = wrap(replaceState, 'replace');
  host.addEventListener('popstate', onPop);

  const stepsBackTo = (parent: string): number | null => {
    const nav = host.navigation;
    const here = host.location.pathname;
    if (nav?.currentEntry) {
      const entries = nav.entries();
      let i = nav.currentEntry.index - 1;
      // Entries of this same screen (its layers, stale or not) are crossed too.
      while (i >= 0 && pathnameOf(entries[i]?.url) === here) i -= 1;
      return i >= 0 && pathnameOf(entries[i]?.url) === parent ? nav.currentEntry.index - i : null;
    }
    // No Navigation API: the trail knows screens, not entries. Our own live
    // layers are the only entries we know sit on top; a stale one below is
    // skipped by the `popstate` handler.
    return trail[trail.length - 2] === parent ? 1 + layers.length : null;
  };

  return {
    push(onClose) {
      seq += 1;
      const id = `${session}:${seq}`;
      host.history.pushState({ [LAYER_STATE_KEY]: id }, '', host.location.href);
      const layer: Layer = {
        id,
        href: host.location.href,
        pathname: host.location.pathname,
        navIndex: host.navigation?.currentEntry?.index ?? null,
        onClose,
      };
      layers.push(layer);
      return {
        release() {
          // Back already popped it (and called onClose, which is how we got here).
          if (!layers.includes(layer)) return;
          layers = layers.filter((l) => l !== layer);
          host.defer(() => {
            if (host.now() - navigatedAt < NAVIGATION_GRACE_MS) return;
            if (host.location.pathname !== layer.pathname) return;
            // Only our own entry may be popped. Anything else on top (a URL
            // sheet opened from inside this one) leaves ours stale instead.
            if (tagOf(host.history.state) !== layer.id) return;
            swallow += 1;
            host.history.go(-1);
          });
        },
      };
    },

    back(parent, replace) {
      const steps = stepsBackTo(parent);
      if (steps === null) {
        replace();
        return;
      }
      host.history.go(-steps);
    },

    layersOnTop() {
      if (!isLive(tagOf(host.history.state))) return 0;
      let n = 0;
      for (let i = layers.length - 1; i >= 0 && layers[i].href === host.location.href; i -= 1) n += 1;
      return n;
    },

    noteNavigation() {
      navigatedAt = host.now();
    },

    depth: () => layers.length,

    dispose() {
      host.removeEventListener('popstate', onPop);
      history.pushState = pushState;
      history.replaceState = replaceState;
      layers = [];
    },
  };
}

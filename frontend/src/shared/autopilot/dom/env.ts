import { closeSheet, openSheet, topOf, useSheetStore } from '@/shared/sheet';

import { matchesAny, PILOT_ATTR, type PilotCatalog } from '@/shared/pilot';
import { fail, OK, type ActContext, type PilotEnv } from '../core/runner';
import { useAutopilotStore } from '../core/store';
import { cursorTravelMs, jittered } from '../core/timing';
import { operate, type Hand } from './drivers';
import { findTarget, isVisible, scrollIntoComfort, settle, until } from './locate';

/** What the mounted host lends the engine: the router (a hook's product) and the catalog (owned by `app`). */
export interface PilotRuntime {
  /** Locale-agnostic push (`useRouter()` of `@/shared/i18n`). */
  push: (href: string) => void;
  /** The current path without its locale prefix. */
  path: () => string;
  catalog: PilotCatalog;
}

const SHELL = '.app-shell';
const LOCK_ATTR = 'data-app-locked';

/** Thrown by the hand when a control may not be pressed; `act` turns it into a `blocked` step. */
class NotPressable extends Error {}

/**
 * Is this control what a finger would actually hit? A control under an open sheet, a dialog or the urgent
 * health modal still has a box, so "visible" is not enough. The autopilot's own layer is looked through.
 */
function pressable(el: HTMLElement): boolean {
  if (document.documentElement.hasAttribute(LOCK_ATTR)) return false;
  const rect = el.getBoundingClientRect();
  const hit = document
    .elementsFromPoint(rect.left + rect.width / 2, rect.top + rect.height / 2)
    .find((node) => !node.closest('.ap-layer'));
  if (!hit || !(el.contains(hit) || hit.contains(el))) return false;
  // A link is only followed inside the app.
  const anchor = el.closest('a[href]') as HTMLAnchorElement | null;
  return !anchor || anchor.origin === window.location.origin;
}

const sleep = (ms: number): Promise<void> => new Promise((resolve) => window.setTimeout(resolve, ms));

/** A box in the overlay's coordinates: the overlay fills `.app-shell`, so everything is relative to it. */
function boxOf(el: HTMLElement) {
  const shell = document.querySelector(SHELL)?.getBoundingClientRect();
  const rect = el.getBoundingClientRect();
  const left = shell?.left ?? 0;
  const top = shell?.top ?? 0;
  return { x: rect.left - left, y: rect.top - top, w: rect.width, h: rect.height };
}

function createHand(): Hand {
  const store = () => useAutopilotStore.getState();

  const reach: Hand['reach'] = async (el, ctx) => {
    const { config, timing } = ctx;
    if (scrollIntoComfort(el, !config.still)) await ctx.wait(timing.scrollSettleMs);
    await settle(el);
    if (config.still) return;

    const box = boxOf(el);
    const to = { x: box.x + box.w / 2, y: box.y + box.h / 2 };
    const from = store().pointer ?? { x: to.x, y: to.y + 160 };
    const travelMs = config.visuals.cursor ? cursorTravelMs(Math.hypot(to.x - from.x, to.y - from.y), timing) : 0;
    store().set({
      pointer: config.visuals.cursor ? { ...to, travelMs } : null,
      ring: config.visuals.highlight ? box : null,
    });
    await ctx.wait(travelMs + jittered(timing.hoverMs, timing.jitter, ctx.rng));
  };

  const click = async (el: HTMLElement, ctx: ActContext, holdMs: number) => {
    if (!ctx.config.still && ctx.config.visuals.ripple) store().set({ press: store().press + 1 });
    await ctx.wait(jittered(holdMs, ctx.timing.jitter, ctx.rng));
    if (!pressable(el)) {
      // A sticky footer or header may sit on it: bring it to the middle of its scroller and look again.
      el.scrollIntoView({ block: 'center', inline: 'nearest', behavior: 'auto' });
      await ctx.wait(ctx.timing.scrollSettleMs);
      await settle(el);
      if (!pressable(el)) throw new NotPressable();
    }
    // A real click event: the component's own onClick runs, exactly as for a finger.
    el.click();
  };

  return {
    reach,
    press: async (el, ctx) => {
      await reach(el, ctx);
      await click(el, ctx, ctx.timing.pressHoldMs);
    },
    pressAgain: (el, ctx) => click(el, ctx, ctx.timing.pressHoldMs * 0.6),
  };
}

export function createBrowserEnv(runtime: PilotRuntime): PilotEnv {
  const hand = createHand();
  const store = () => useAutopilotStore.getState();
  const sheets = () => useSheetStore.getState().stack;

  return {
    sleep,
    locked: () => document.documentElement.hasAttribute(LOCK_ATTR),
    exists: (target) => findTarget(target) !== null,
    waitFor: (target, state, timeoutMs) => until(() => (findTarget(target) !== null) === (state === 'visible'), timeoutMs),

    surfaceOpen: (surface) => {
      if (surface.sheet) return sheets().some((s) => s.id === surface.sheet);
      if (!surface.route) return true;
      if (runtime.path() !== surface.route) return false;
      const search = new URLSearchParams(window.location.search);
      return Object.entries(surface.query ?? {}).every(([key, value]) => search.get(key) === value);
    },

    open: async (where, ctx) => {
      if ('sheet' in where) {
        // Only sheets the catalog lists: an id nobody registered would leave an empty panel on screen.
        if (!runtime.catalog.sheets.some((s) => s.sheet === where.sheet)) return fail('blocked');
        const top = topOf(sheets());
        if (top?.id === where.sheet && (top.arg ?? undefined) === (where.arg ?? undefined)) return OK;
        openSheet(where.sheet, where.arg);
        const shown = await until(() => {
          const panel = document.querySelector('.osheet');
          return panel !== null && isVisible(panel);
        }, ctx.config.timeouts.navigationMs);
        if (!shown) return fail('navigation_failed');
        await ctx.wait(ctx.timing.afterSheetMs);
        return OK;
      }
      const query = where.query ? new URLSearchParams(where.query).toString() : '';
      if (runtime.path() === where.route && !query) return OK;
      runtime.push(query ? `${where.route}?${query}` : where.route);
      const arrived = await until(() => runtime.path() === where.route, ctx.config.timeouts.navigationMs);
      if (!arrived) return fail('navigation_failed');
      await ctx.wait(ctx.timing.afterNavigateMs);
      return OK;
    },

    close: async (ctx) => {
      const depth = sheets().length;
      if (depth === 0) return OK;
      closeSheet();
      const closed = await until(() => sheets().length < depth, ctx.config.timeouts.navigationMs);
      if (!closed) return fail('navigation_failed');
      await ctx.wait(ctx.timing.afterSheetMs);
      return OK;
    },

    act: async (verb, target, value, ctx) => {
      const el = findTarget(target);
      if (!el) return fail('target_not_found');
      if (typeof target !== 'string') {
        // A label locator must not reach a control the config protects under its own name.
        const own = el.closest(`[${PILOT_ATTR}]`)?.getAttribute(PILOT_ATTR);
        const { blockedTargets, confirmTargets } = ctx.config.safety;
        if (own && matchesAny([...blockedTargets, ...confirmTargets], own)) return fail('blocked');
      }
      try {
        return await operate(verb, el, value, ctx, hand);
      } catch (error) {
        if (error instanceof NotPressable) return fail('blocked');
        throw error;
      }
    },

    caption: (text) => store().set({ caption: text }),
    progress: (step, total) => store().set({ step, total }),

    confirm: (targetId) =>
      new Promise<boolean>((resolve) => {
        store().set({
          confirm: {
            targetId,
            answer: (allowed) => {
              store().set({ confirm: null });
              resolve(allowed);
            },
          },
        });
      }),
  };
}

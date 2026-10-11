/**
 * First-party product analytics emitter (bloom B-N9-08, D-89).
 *
 * Privacy: an event is a NAME from `EVENT_NAMES` plus, at most, the enumerated props below — never a cycle, symptom,
 * pregnancy or lab value, never free text (the type makes that impossible to call, and the server refuses anything
 * else with a 422). Opt-in: the server only records users who granted the «anonymous statistics» consent and answers
 * `tracking: false` otherwise; the emitter then stops for six hours (or until the consent switch is changed).
 * Events are batched in memory (nothing is persisted, the queue dies with the tab and with the session) and never
 * retried: analytics must not cost the user anything.
 */

export type AnalyticsEvent =
  | { name: 'app_open' }
  | { name: 'onboarding_started' }
  | { name: 'onboarding_done' }
  | { name: 'log_saved'; props: { source: 'quick' | 'full' } }
  | {
      name: 'plus_paywall_view';
      props: { surface: 'home' | 'settings' | 'assistant' | 'labs' | 'telemed' | 'learning' | 'other' };
    }
  | { name: 'plus_purchase'; props: { plan_months: number } }
  | { name: 'assistant_message_sent' }
  | { name: 'lab_upload_started' }
  | { name: 'telemed_booking_started' }
  | { name: 'learning_lesson_started' };

export const BATCH_SIZE = 10;
export const FLUSH_DELAY_MS = 5_000;
export const MAX_QUEUE = 50;
export const OPT_OUT_MS = 6 * 60 * 60 * 1000;
const OFF_KEY = 'ritme_analytics_off_until';

export type Sender = (events: { name: string; props?: Record<string, unknown>; occurred_at: string }[]) => Promise<{
  tracking: boolean;
}>;

interface Deps {
  send: Sender;
  isAuthenticated: () => boolean;
  now?: () => number;
  storage?: Pick<Storage, 'getItem' | 'setItem' | 'removeItem'> | null;
}

export function createTracker(deps: Deps) {
  const now = deps.now ?? Date.now;
  let queue: { name: string; props?: Record<string, unknown>; occurred_at: string }[] = [];
  let timer: ReturnType<typeof setTimeout> | null = null;
  let offUntil = 0;

  const store = () => {
    try {
      return deps.storage ?? null;
    } catch {
      return null;
    }
  };
  const readOff = () => {
    if (offUntil > now()) return true;
    try {
      const raw = store()?.getItem(OFF_KEY);
      offUntil = raw ? Number(raw) || 0 : 0;
    } catch {
      offUntil = 0;
    }
    return offUntil > now();
  };

  async function flush(): Promise<void> {
    if (timer) {
      clearTimeout(timer);
      timer = null;
    }
    if (queue.length === 0) return;
    const batch = queue;
    queue = [];
    if (!deps.isAuthenticated() || readOff()) return;
    try {
      const res = await deps.send(batch);
      if (!res.tracking) {
        offUntil = now() + OPT_OUT_MS;
        try {
          store()?.setItem(OFF_KEY, String(offUntil));
        } catch {
          // storage may be blocked; the in-memory flag still holds
        }
      }
    } catch {
      // never retried, never surfaced
    }
  }

  function track(event: AnalyticsEvent): void {
    if (readOff()) return;
    if (queue.length >= MAX_QUEUE) queue.shift();
    queue.push({
      name: event.name,
      ...('props' in event ? { props: event.props } : {}),
      occurred_at: new Date(now()).toISOString(),
    });
    if (queue.length >= BATCH_SIZE) void flush();
    else timer ??= setTimeout(() => void flush(), FLUSH_DELAY_MS);
  }

  /** A change of the anonymous-statistics switch: forget the opt-out so the next event asks the server again. */
  function consentChanged(): void {
    offUntil = 0;
    try {
      store()?.removeItem(OFF_KEY);
    } catch {
      // ignore
    }
  }

  /** The session ended: drop what is queued. */
  function reset(): void {
    queue = [];
    if (timer) clearTimeout(timer);
    timer = null;
  }

  return { track, flush, consentChanged, reset };
}

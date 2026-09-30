/** What the /welcome route shows. */
export type WelcomeView = { kind: 'intro'; slide: number } | { kind: 'card' };

const SLIDE_COUNT = 5;

/**
 * `?step=welcome` → the welcome card; `?slide=N` (1-based) → that intro slide
 * (deep links + QA screenshots); otherwise the card once the intro has been
 * seen, the first slide before.
 */
export function resolveWelcomeView(search: string, seenIntro: boolean): WelcomeView {
  const q = new URLSearchParams(search);
  if (q.get('step') === 'welcome') return { kind: 'card' };
  const n = Number(q.get('slide'));
  if (Number.isInteger(n) && n >= 1 && n <= SLIDE_COUNT) return { kind: 'intro', slide: n - 1 };
  return seenIntro ? { kind: 'card' } : { kind: 'intro', slide: 0 };
}

import type { IvfDangerSign, IvfGuidance, IvfNextStep, IvfOutcomeResult } from '@/entities/ivf';

/*
 * Pure helpers of the two-week wait (nbl_IVF_TWW, CB-IVF-05). No React, no
 * locale: the UI turns the keys into copy.
 */

/** What the ring's centre says. */
export type Countdown =
  | { kind: 'days'; days: number }
  | { kind: 'today' }
  | { kind: 'passed' }
  | { kind: 'since'; day: number }
  | { kind: 'unknown' };

export function countdown(daysToBeta: number | null, daysSinceTransfer: number | null): Countdown {
  if (daysToBeta !== null) {
    if (daysToBeta > 0) return { kind: 'days', days: daysToBeta };
    return daysToBeta === 0 ? { kind: 'today' } : { kind: 'passed' };
  }
  if (daysSinceTransfer !== null) return { kind: 'since', day: daysSinceTransfer };
  return { kind: 'unknown' };
}

/**
 * Ring fill = the part of the wait already behind her (transfer → beta). Full
 * on the beta day; empty while either end is unknown.
 */
export function waitProgress(daysSinceTransfer: number | null, daysToBeta: number | null): number {
  if (daysToBeta !== null && daysToBeta <= 0) return 1;
  if (daysSinceTransfer === null || daysToBeta === null) return 0;
  const span = daysSinceTransfer + daysToBeta;
  return span > 0 ? Math.min(1, Math.max(0, daysSinceTransfer / span)) : 0;
}

/** Catalog text by code (admin-editable), else null → the bundled fallback copy. */
export function guidanceBody(items: readonly IvfGuidance[] | undefined, code: string): string | null {
  return items?.find((item) => item.code === code)?.body ?? null;
}

/** The danger card: the OHSS row leads (title = the heading), every other row adds its body. */
export function dangerCopy(signs: readonly IvfDangerSign[] | undefined): {
  title: string | null;
  bodies: string[];
  hotlines: string[];
} {
  const list = signs ?? [];
  const lead = list.find((s) => s.code === 'ohss') ?? list[0];
  const ordered = lead ? [lead, ...list.filter((s) => s !== lead)] : [];
  const bodies = ordered.flatMap((s) => (s.body ? [s.body] : []));
  const hotlines = [...new Set(ordered.flatMap((s) => s.hotlines))];
  return { title: lead?.title ?? null, bodies, hotlines: hotlines.length ? hotlines : ['115'] };
}

/** The follow-ups shown after an outcome, in screen order; falls back to the API's documented defaults. */
export function followUps(result: IvfOutcomeResult, nextSteps: readonly IvfNextStep[]): IvfNextStep[] {
  const fallback: IvfNextStep[] =
    result === 'positive' ? ['pregnancy_setup'] : result === 'negative' ? ['loss', 'new_cycle'] : ['new_cycle'];
  const steps = nextSteps.length ? nextSteps : fallback;
  // A negative or cancelled result never offers pregnancy setup; only a negative one offers the loss path.
  return steps.filter((s) => (s === 'pregnancy_setup' ? result === 'positive' : s === 'loss' ? result === 'negative' : true));
}

/** Routes of the follow-ups: bloom's pregnancy setup, CB-LOSS-02's `/loss` (never a second loss flow), the IVF home. */
export const FOLLOW_UP_HREF: Record<IvfNextStep, string> = {
  pregnancy_setup: '/pregnancy/setup',
  loss: '/loss',
  new_cycle: '/ivf',
};

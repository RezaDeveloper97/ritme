import type { EpdsItem, EpdsKind, EpdsResult, SafetyMessage } from '@/entities/postpartum';

/*
 * EPDS check flow (nbl_v15_MoodCheck, B-N5-01 scoring). Pure state so the
 * safety path is unit-tested: questions → (submit) → result | urgent safety.
 * Answers live only in this state and the one request; nothing persists them.
 */

/** Questions per page — the artboard shows the weekly check's 3 on one page. */
export const PAGE_SIZE = 3;

/** EPDS item 10 (self-harm) and the full-check cut-off — mirror of the server rule, used only as a fallback. */
export const SELF_HARM_ITEM = 'q10';
export const FULL_URGENT_TOTAL = 13;

/** The national lines the urgent screen always offers (B-N5-01: 115 / 123 / 1480), labels come from i18n. */
export const FALLBACK_LINES = [
  { number: '115', labelKey: 'emergency' },
  { number: '123', labelKey: 'social' },
  { number: '1480', labelKey: 'counsel' },
] as const;

export type Answers = Readonly<Record<string, number>>;

export type FlowState =
  | { step: 'questions'; page: number; answers: Answers }
  | { step: 'submitting'; page: number; answers: Answers }
  | { step: 'result'; result: EpdsResult }
  | { step: 'safety'; safety: SafetyMessage | null; saved: boolean; answers: Answers | null };

export type FlowEvent =
  | { type: 'answer'; code: string; score: number }
  | { type: 'next'; pageCount: number }
  | { type: 'prev' }
  | { type: 'submit' }
  | { type: 'success'; result: EpdsResult }
  | { type: 'failure'; kind: EpdsKind; items: readonly EpdsItem[] }
  | { type: 'restart' };

export const initialFlow: FlowState = { step: 'questions', page: 0, answers: {} };

export function pagesOf<T>(items: readonly T[], size = PAGE_SIZE): T[][] {
  const pages: T[][] = [];
  for (let i = 0; i < items.length; i += size) pages.push(items.slice(i, i + size));
  return pages;
}

export function pageComplete(page: readonly EpdsItem[], answers: Answers): boolean {
  return page.every((it) => answers[it.code] !== undefined);
}

/** The POST body's answers: exactly the questionnaire's items, nothing else. */
export function answersPayload(items: readonly EpdsItem[], answers: Answers): Record<string, number> | null {
  const out: Record<string, number> = {};
  for (const it of items) {
    const v = answers[it.code];
    if (v === undefined) return null;
    out[it.code] = v;
  }
  return out;
}

/** Client-side mirror of the urgent rule, only for when the request fails: item 10 > 0, or a full total ≥ 13. */
export function locallyUrgent(kind: EpdsKind, items: readonly EpdsItem[], answers: Answers): boolean {
  if ((answers[SELF_HARM_ITEM] ?? 0) > 0) return true;
  if (kind !== 'full') return false;
  const total = items.reduce((sum, it) => sum + (answers[it.code] ?? 0), 0);
  return total >= FULL_URGENT_TOTAL;
}

/** The scored result goes to the safety screen when the server says urgent (message level or the check flag). */
export function isUrgent(result: EpdsResult): boolean {
  return result.check.urgent || result.safety?.level === 'urgent';
}

export function flowReducer(state: FlowState, event: FlowEvent): FlowState {
  switch (event.type) {
    case 'answer':
      if (state.step !== 'questions') return state;
      return { ...state, answers: { ...state.answers, [event.code]: event.score } };
    case 'next':
      if (state.step !== 'questions') return state;
      return { ...state, page: Math.min(event.pageCount - 1, state.page + 1) };
    case 'prev':
      if (state.step !== 'questions') return state;
      return { ...state, page: Math.max(0, state.page - 1) };
    case 'submit':
      if (state.step !== 'questions') return state;
      return { step: 'submitting', page: state.page, answers: state.answers };
    case 'success':
      // The answers are dropped here: from now on only totals and the message exist.
      return isUrgent(event.result)
        ? { step: 'safety', safety: event.result.safety, saved: true, answers: null }
        : { step: 'result', result: event.result };
    case 'failure':
      if (state.step !== 'submitting') return state;
      // Never block the safety path on an error: an urgent pattern shows the call screen anyway, and keeps
      // the answers only so «retry save» can resend them.
      if (locallyUrgent(event.kind, event.items, state.answers)) {
        return { step: 'safety', safety: null, saved: false, answers: state.answers };
      }
      return { step: 'questions', page: state.page, answers: state.answers };
    case 'restart':
      return initialFlow;
  }
}

export interface CallLine {
  number: string;
  /** Server label, or null → the screen uses the i18n fallback for `labelKey`. */
  label: string | null;
  labelKey: (typeof FALLBACK_LINES)[number]['labelKey'] | null;
}

/** Call buttons of the urgent screen: the server's call actions, else (or when missing) the national lines. */
export function safetyCalls(safety: SafetyMessage | null): CallLine[] {
  const calls: CallLine[] = (safety?.actions ?? []).flatMap((a) =>
    a.type === 'call'
      ? [{ number: a.number, label: a.label, labelKey: FALLBACK_LINES.find((l) => l.number === a.number)?.labelKey ?? null }]
      : [],
  );
  for (const line of FALLBACK_LINES) {
    if (!calls.some((c) => c.number === line.number)) calls.push({ number: line.number, label: null, labelKey: line.labelKey });
  }
  return calls;
}

import { describe, expect, it } from 'vitest';

import type { EpdsItem, EpdsResult } from '@/entities/postpartum';

import {
  answersPayload,
  flowReducer,
  initialFlow,
  isUrgent,
  locallyUrgent,
  pageComplete,
  pagesOf,
  safetyCalls,
  type FlowState,
} from './flow';

const item = (code: string): EpdsItem => ({
  code,
  number: Number(code.slice(1)),
  text: code,
  options: [0, 1, 2, 3].map((score) => ({ score, label: String(score) })),
});
const SHORT = ['q3', 'q4', 'q5'].map(item);
const FULL = Array.from({ length: 10 }, (_, i) => item(`q${i + 1}`));

const check = { id: 1, kind: 'full' as const, takenOn: '2026-09-23', week: 4, total: 2, max: 30, band: 'low', bandLabel: null, urgent: false };
const result = (over: Partial<EpdsResult> = {}): EpdsResult => ({ check, safety: null, followUp: null, checkin: null, ...over });

function answerAll(state: FlowState, items: EpdsItem[], score: number): FlowState {
  return items.reduce((s, it) => flowReducer(s, { type: 'answer', code: it.code, score }), state);
}

describe('EPDS flow paging', () => {
  it('splits the weekly check into one page and the full check into four', () => {
    expect(pagesOf(SHORT)).toHaveLength(1);
    expect(pagesOf(FULL).map((p) => p.length)).toEqual([3, 3, 3, 1]);
  });

  it('a page continues only when every question on it is answered', () => {
    let s = flowReducer(initialFlow, { type: 'answer', code: 'q3', score: 1 });
    expect(s.step === 'questions' && pageComplete(SHORT, s.answers)).toBe(false);
    s = answerAll(s, SHORT, 0);
    expect(s.step === 'questions' && pageComplete(SHORT, s.answers)).toBe(true);
  });

  it('next / prev stay inside the pages', () => {
    let s = flowReducer(initialFlow, { type: 'next', pageCount: 4 });
    s = flowReducer(s, { type: 'next', pageCount: 4 });
    s = flowReducer(s, { type: 'next', pageCount: 4 });
    s = flowReducer(s, { type: 'next', pageCount: 4 });
    expect(s).toMatchObject({ page: 3 });
    s = flowReducer(flowReducer(initialFlow, { type: 'prev' }), { type: 'prev' });
    expect(s).toMatchObject({ page: 0 });
  });

  it('the payload holds exactly the questionnaire items (null until complete)', () => {
    expect(answersPayload(SHORT, { q3: 1 })).toBeNull();
    expect(answersPayload(SHORT, { q3: 1, q4: 0, q5: 2, q10: 3 })).toEqual({ q3: 1, q4: 0, q5: 2 });
  });
});

describe('EPDS result routing', () => {
  const submitted = flowReducer(answerAll(initialFlow, FULL, 0), { type: 'submit' });

  it('a calm result goes to the result screen and drops the answers', () => {
    const s = flowReducer(submitted, { type: 'success', result: result() });
    expect(s.step).toBe('result');
    expect(JSON.stringify(s)).not.toContain('"answers"');
  });

  it('an urgent server message goes to the safety screen', () => {
    const safety = { level: 'urgent', title: 't', body: 'b', actions: [] };
    const s = flowReducer(submitted, { type: 'success', result: result({ safety }) });
    expect(s).toEqual({ step: 'safety', safety, saved: true, answers: null });
  });

  it('the check flag alone is enough to be urgent', () => {
    expect(isUrgent(result({ check: { ...check, urgent: true } }))).toBe(true);
    expect(isUrgent(result({ safety: { level: 'follow_up', title: null, body: null, actions: [] } }))).toBe(false);
  });

  it('a failed request with item 10 > 0 still shows the safety screen (never blocked by errors)', () => {
    const s0 = flowReducer(answerAll(answerAll(initialFlow, FULL, 0), [item('q10')], 1), { type: 'submit' });
    const s = flowReducer(s0, { type: 'failure', kind: 'full', items: FULL });
    expect(s).toMatchObject({ step: 'safety', safety: null, saved: false });
  });

  it('a failed full check with total ≥ 13 shows the safety screen', () => {
    const s0 = flowReducer(answerAll(initialFlow, FULL.slice(0, 9), 2), { type: 'answer', code: 'q10', score: 0 });
    expect(locallyUrgent('full', FULL, (s0 as { answers: Record<string, number> }).answers)).toBe(true);
  });

  it('a failed calm check goes back to the questions with the answers kept for a retry', () => {
    const s = flowReducer(submitted, { type: 'failure', kind: 'full', items: FULL });
    expect(s).toMatchObject({ step: 'questions', page: 0 });
  });

  it('the weekly check is never urgent on its total alone', () => {
    expect(locallyUrgent('short', SHORT, { q3: 3, q4: 3, q5: 3 })).toBe(false);
  });
});

describe('safety calls', () => {
  it('keeps the server order and labels, then adds any missing national line', () => {
    const calls = safetyCalls({
      level: 'urgent',
      title: null,
      body: null,
      actions: [
        { type: 'call', number: '123', label: 'Social' },
        { type: 'open_check', kind: 'full', label: null },
      ],
    });
    expect(calls.map((c) => c.number)).toEqual(['123', '115', '1480']);
    expect(calls[0]).toEqual({ number: '123', label: 'Social', labelKey: 'social' });
  });

  it('with no message at all it still offers 115 / 123 / 1480', () => {
    expect(safetyCalls(null).map((c) => c.number)).toEqual(['115', '123', '1480']);
  });
});

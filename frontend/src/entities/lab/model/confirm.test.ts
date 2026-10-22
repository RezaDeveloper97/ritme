import { describe, expect, it } from 'vitest';

import { explainedOutOfRange, markersToConfirm, referenceKind, showsNearEdge } from './confirm';
import type { LabInterpretation, LabMarker } from './types';

const marker = (over: Partial<LabMarker>): LabMarker => ({
  id: 1,
  code: null,
  name: 'x',
  printedName: 'x',
  subtitle: null,
  value: 1,
  valueText: null,
  unit: null,
  reference: { low: null, high: null, text: null, source: null, label: null, note: null },
  state: 'normal',
  stateLabel: '',
  attention: false,
  band: null,
  nearEdge: false,
  confidence: null,
  lowConfidence: false,
  needsConfirmation: false,
  confirmedAt: null,
  source: 'extracted',
  ...over,
});

describe('markersToConfirm', () => {
  it('lists only the values waiting for a confirmation, in sheet order', () => {
    const list = [
      marker({ id: 3, needsConfirmation: true }),
      marker({ id: 4, lowConfidence: true, confirmedAt: '2026-10-01T10:00:00+03:30' }),
      marker({ id: 5, needsConfirmation: true }),
    ];
    expect(markersToConfirm(list)).toEqual([3, 5]);
    expect(markersToConfirm([])).toEqual([]);
  });
});

describe('explainedOutOfRange', () => {
  const interp: LabInterpretation = {
    summary: 's',
    source: 'ai',
    stale: false,
    outOfRange: [
      { markerId: 1, code: 'ferritin', name: 'فریتین', state: 'low', stateLabel: 'پایین', explanation: 'ممکن است مرتبط باشد.' },
      { markerId: 2, code: null, name: 'Hb', state: 'borderline_low', stateLabel: 'پایین مرزی', explanation: null },
    ],
    redFlags: [],
    doctorQuestions: [],
    disclaimer: '',
  };
  it('keeps the explained items only', () => {
    expect(explainedOutOfRange(interp).map((o) => o.markerId)).toEqual([1]);
    expect(explainedOutOfRange(null)).toEqual([]);
  });
});

describe('referenceKind / showsNearEdge', () => {
  it('tells the sheet range from the general one', () => {
    expect(referenceKind({ low: 1, high: 2, text: null, source: 'sheet', label: null, note: null })).toBe('sheet');
    expect(referenceKind({ low: 1, high: 2, text: null, source: 'typical', label: 'محدوده عمومی', note: null })).toBe('general');
    expect(referenceKind({ low: null, high: null, text: null, source: null, label: null, note: null })).toBeNull();
  });
  it('hints near the edge only without a borderline state', () => {
    expect(showsNearEdge({ nearEdge: true, state: 'low' })).toBe(true);
    expect(showsNearEdge({ nearEdge: true, state: 'borderline_low' })).toBe(false);
    expect(showsNearEdge({ nearEdge: false, state: 'high' })).toBe(false);
  });
});

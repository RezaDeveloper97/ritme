import { describe, expect, it } from 'vitest';

import type { WeightGain } from '../api/schema';
import { weightNote } from './chart';

const w = (over: Partial<WeightGain>): WeightGain => ({ missing: null, minLogs: 2, logsNeeded: 0, ...over }) as WeightGain;

describe('weightNote (§4.1 pregnancy weight minimum, T-M10-26b)', () => {
  it('names the weights still needed when only the minimum blocks the card', () => {
    expect(weightNote(w({ logsNeeded: 1 }))).toEqual({ kind: 'notEnough', min: 2, left: 1 });
  });
  it('keeps the missing reason first', () => {
    expect(weightNote(w({ missing: 'height', logsNeeded: 1 }))).toEqual({ kind: 'missing', reason: 'height' });
    expect(weightNote(w({ missing: 'weights', logsNeeded: 2 }))).toEqual({ kind: 'missing', reason: 'weights' });
    expect(weightNote(null)).toEqual({ kind: 'missing', reason: 'weights' });
  });
  it('falls back to the missing line on an older backend without logs_needed', () => {
    expect(weightNote(w({ logsNeeded: null, minLogs: null }))).toEqual({ kind: 'missing', reason: 'weights' });
  });
});

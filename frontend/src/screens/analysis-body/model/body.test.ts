import { describe, expect, it } from 'vitest';

import { headlineWeight, maxIndex, signed, weekdayValues } from './body';

describe('analysis-body model', () => {
  it('re-orders weekdays into the locale week', () => {
    const v = [1, 2, 3, 4, 5, 6, 7];
    expect(weekdayValues(v, ['sat', 'sun', 'mon', 'tue', 'wed', 'thu', 'fri']).map((d) => d.value)).toEqual(v);
    expect(weekdayValues(v, ['sun', 'mon', 'tue', 'wed', 'thu', 'fri', 'sat']).map((d) => d.value)).toEqual([2, 3, 4, 5, 6, 7, 1]);
  });

  it('finds the max and formats deltas', () => {
    expect(maxIndex([6.8, null, 7.1, 6.3])).toBe(2);
    expect(maxIndex([null, 0])).toBe(-1);
    expect(signed(-0.63)).toEqual({ sign: '−', abs: '0.6' });
    expect(signed(2)).toEqual({ sign: '+', abs: '2' });
    expect(signed(0.01)).toEqual({ sign: '', abs: '0' });
  });

  it('prefers the latest moving average', () => {
    expect(headlineWeight([{ value: 58, avg7: 58.1 }, { value: 57.8, avg7: null }], 57.8)).toBe(58.1);
    expect(headlineWeight([], 60)).toBe(60);
  });
});

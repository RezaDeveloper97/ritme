import { describe, expect, it } from 'vitest';

import { childTone, dueIn, parseDecimalInput, roundPercentile } from './format';

describe('dueIn', () => {
  it('splits overdue, today, days and months', () => {
    expect(dueIn(-3)).toEqual({ kind: 'overdue' });
    expect(dueIn(0)).toEqual({ kind: 'today' });
    expect(dueIn(18)).toEqual({ kind: 'days', n: 18 });
    expect(dueIn(59)).toEqual({ kind: 'days', n: 59 });
    expect(dueIn(62)).toEqual({ kind: 'months', n: 2 });
  });
});

describe('parseDecimalInput', () => {
  it('reads Persian digits and the Persian decimal separator', () => {
    expect(parseDecimalInput('۳٫۲')).toEqual({ text: '3.2', value: 3.2 });
    expect(parseDecimalInput('50')).toEqual({ text: '50', value: 50 });
  });
  it('keeps one point, caps decimals and integer digits', () => {
    expect(parseDecimalInput('3..25', 1)).toEqual({ text: '3.2', value: 3.2 });
    expect(parseDecimalInput('12345')).toEqual({ text: '123', value: 123 });
    expect(parseDecimalInput('abc')).toEqual({ text: '', value: undefined });
    expect(parseDecimalInput('.')).toEqual({ text: '0.', value: undefined });
  });
});

describe('roundPercentile / childTone', () => {
  it('rounds and clamps', () => {
    expect(roundPercentile(47.2)).toBe(47);
    expect(roundPercentile(null)).toBeNull();
    expect(roundPercentile(100.4)).toBe(100);
  });
  it('maps sex to a tone', () => {
    expect(childTone('girl')).toBe('bloom');
    expect(childTone('boy')).toBe('data');
    expect(childTone(null)).toBe('brand');
  });
});

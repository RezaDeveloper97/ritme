import { describe, expect, it } from 'vitest';

import { formatValue, parseValue } from './format';

const f = (name: string) => ({ name, default: 0, min: 0, max: 1000 });
const num = (n: number) => String(n);

describe('formatValue', () => {
  it('prints pairs and singles', () => {
    expect(formatValue([f('systolic'), f('diastolic')], { systolic: 180, diastolic: 120 }, num)).toBe('180 / 120');
    expect(formatValue([f('min'), f('max')], { min: 24, max: 38 }, num)).toBe('24–38');
    expect(formatValue([f('score')], { score: 13 }, num)).toBe('13');
    expect(formatValue([f('before'), f('after')], { before: 5, after: 1 }, num)).toBe('before 5 · after 1');
  });
});

describe('parseValue', () => {
  it('needs every field as a number', () => {
    expect(parseValue([f('min'), f('max')], { min: '55', max: '100' })).toEqual({ min: 55, max: 100 });
    expect(parseValue([f('min'), f('max')], { min: '55', max: '' })).toBeNull();
    expect(parseValue([f('min')], { min: 'x' })).toBeNull();
  });
});

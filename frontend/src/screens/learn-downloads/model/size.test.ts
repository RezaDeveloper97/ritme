import { describe, expect, it } from 'vitest';

import { meterPercent, sizeParts } from './size';

describe('sizeParts', () => {
  it('picks a unit', () => {
    expect(sizeParts(0)).toEqual({ unit: 'mb', n: 0 });
    expect(sizeParts(500)).toEqual({ unit: 'kb', n: 1 });
    expect(sizeParts(4 * 1024 * 1024)).toEqual({ unit: 'mb', n: 4 });
    expect(sizeParts(214 * 1024 * 1024 + 9000)).toEqual({ unit: 'mb', n: 214 });
    expect(sizeParts(3.2 * 1024 ** 3)).toEqual({ unit: 'gb', n: 3.2 });
  });
});

describe('meterPercent', () => {
  it('clamps and handles unknown quota', () => {
    expect(meterPercent(10, null)).toBe(0);
    expect(meterPercent(1, 1e9)).toBe(1);
    expect(meterPercent(420, 3200)).toBe(13);
    expect(meterPercent(5000, 3200)).toBe(100);
  });
});

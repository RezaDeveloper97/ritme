import { describe, expect, it } from 'vitest';

import { bpsToPercentText, percentToBps, rialsToToman, tomanToRials } from './money';

describe('plus money helpers', () => {
  it('converts rials and toman', () => {
    expect(rialsToToman(990000)).toBe(99000);
    expect(tomanToRials('99000')).toBe(990000);
    expect(tomanToRials('')).toBeNull();
    expect(tomanToRials('12.5')).toBeNull();
  });
  it('converts VAT percent and basis points', () => {
    expect(bpsToPercentText(1000)).toBe('10');
    expect(bpsToPercentText(950)).toBe('9.5');
    expect(percentToBps('9.5')).toBe(950);
    expect(percentToBps('0')).toBe(0);
    expect(percentToBps('101')).toBeNull();
    expect(percentToBps('')).toBeNull();
  });
});

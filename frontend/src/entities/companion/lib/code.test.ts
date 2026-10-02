import { describe, expect, it } from 'vitest';

import { isCompleteCompanionCode, normalizeCompanionCode } from './code';

describe('normalizeCompanionCode', () => {
  it('upper-cases and strips separators', () => {
    expect(normalizeCompanionCode('rt7-k2x')).toBe('RT7K2X');
  });
  it('reads Persian and Arabic digits', () => {
    expect(normalizeCompanionCode('ab۲٣cd')).toBe('AB23CD');
  });
  it('keeps at most six characters of a pasted sentence', () => {
    expect(normalizeCompanionCode('کد همدم من: RT7K2X')).toBe('RT7K2X');
    expect(normalizeCompanionCode('RT7K2X99')).toBe('RT7K2X');
  });
  it('drops non-Latin letters', () => {
    expect(normalizeCompanionCode('سلامAB')).toBe('AB');
  });
  it('knows a complete code', () => {
    expect(isCompleteCompanionCode('RT7K2X')).toBe(true);
    expect(isCompleteCompanionCode('RT7')).toBe(false);
  });
});

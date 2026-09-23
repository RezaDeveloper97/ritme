import { describe, expect, it } from 'vitest';

import {
  BBT_DEFAULT,
  BBT_MAX,
  BBT_MIN,
  clampBbt,
  formatBbt,
  isBbtInRange,
  parseBbt,
  roundBbt,
  stepBbt,
} from './bbt';

describe('formatBbt', () => {
  it('uses Persian digits and «٫» with 2 decimals in fa', () => {
    expect(formatBbt(36.42, 'fa')).toBe('۳۶٫۴۲');
    expect(formatBbt(36.4, 'fa')).toBe('۳۶٫۴۰');
    expect(formatBbt(37, 'fa')).toBe('۳۷٫۰۰');
  });

  it('keeps Latin digits and a dot elsewhere', () => {
    expect(formatBbt(36.42, 'en')).toBe('36.42');
    expect(formatBbt(36.4, 'en')).toBe('36.40');
  });

  it('supports 1-decimal axis ticks', () => {
    expect(formatBbt(36.2, 'fa', 1)).toBe('۳۶٫۲');
    expect(formatBbt(36.8, 'en', 1)).toBe('36.8');
  });

  it('renders nothing for a missing reading', () => {
    expect(formatBbt(null, 'fa')).toBe('');
    expect(formatBbt(Number.NaN, 'en')).toBe('');
  });
});

describe('parseBbt', () => {
  it('reads Latin input', () => {
    expect(parseBbt('36.42')).toBe(36.42);
    expect(parseBbt(' 36.4 ')).toBe(36.4);
    expect(parseBbt('37')).toBe(37);
  });

  it('reads Persian / Arabic digits and every decimal separator', () => {
    expect(parseBbt('۳۶٫۴۲')).toBe(36.42);
    expect(parseBbt('۳۶.۴۲')).toBe(36.42);
    expect(parseBbt('٣٦٫٤٢')).toBe(36.42);
    expect(parseBbt('36,42')).toBe(36.42);
    expect(parseBbt('۳۶/۴۲')).toBe(36.42);
  });

  it('tolerates a trailing unit', () => {
    expect(parseBbt('36.42°C')).toBe(36.42);
    expect(parseBbt('۳۶٫۴۲°')).toBe(36.42);
  });

  it('rounds to 2 decimals', () => {
    expect(parseBbt('36.425')).toBe(36.43);
    expect(parseBbt('36.421')).toBe(36.42);
  });

  it('rejects empty and non-numeric input', () => {
    expect(parseBbt('')).toBeNull();
    expect(parseBbt('   ')).toBeNull();
    expect(parseBbt('abc')).toBeNull();
    expect(parseBbt('36.4.2')).toBeNull();
    expect(parseBbt('-36')).toBeNull();
    expect(parseBbt('365')).toBeNull();
  });

  it('does not enforce the range (the form explains it)', () => {
    expect(parseBbt('34.9')).toBe(34.9);
    expect(isBbtInRange(34.9)).toBe(false);
  });
});

describe('range & steps', () => {
  it('accepts the API range inclusively', () => {
    expect(isBbtInRange(BBT_MIN)).toBe(true);
    expect(isBbtInRange(BBT_MAX)).toBe(true);
    expect(isBbtInRange(36.42)).toBe(true);
    expect(isBbtInRange(34.99)).toBe(false);
    expect(isBbtInRange(38.51)).toBe(false);
    expect(isBbtInRange(Number.NaN)).toBe(false);
  });

  it('steps by 0.01 without float drift', () => {
    expect(stepBbt(36.42, 1)).toBe(36.43);
    expect(stepBbt(36.42, -1)).toBe(36.41);
    expect(stepBbt(36.1, 1)).toBe(36.11);
    let v = 36.0;
    for (let i = 0; i < 7; i++) v = stepBbt(v, 1);
    expect(v).toBe(36.07);
  });

  it('clamps at the range ends', () => {
    expect(stepBbt(BBT_MAX, 1)).toBe(BBT_MAX);
    expect(stepBbt(BBT_MIN, -1)).toBe(BBT_MIN);
    expect(clampBbt(40)).toBe(BBT_MAX);
    expect(clampBbt(30)).toBe(BBT_MIN);
  });

  it('starts an empty field at the default', () => {
    expect(stepBbt(null, 1)).toBe(BBT_DEFAULT);
    expect(stepBbt(null, -1)).toBe(BBT_DEFAULT);
  });

  it('rounds to the column precision', () => {
    expect(roundBbt(36.4249)).toBe(36.42);
    expect(roundBbt(36.005)).toBe(36.01);
  });
});

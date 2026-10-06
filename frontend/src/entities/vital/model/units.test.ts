import { describe, expect, it } from 'vitest';

import { convertGlucose, fromMgDl, glucoseText, stepGlucose, toMgDl, unitSymbol } from './units';

describe('glucose units (server factor 18)', () => {
  it('converts typed values to canonical mg/dL with one decimal', () => {
    expect(toMgDl(5.2, 'mmol_l')).toBe(93.6);
    expect(toMgDl(94, 'mg_dl')).toBe(94);
  });

  it('shows mg/dL whole and mmol/L with one decimal', () => {
    expect(fromMgDl(93.6, 'mg_dl')).toBe(94);
    expect(fromMgDl(94, 'mmol_l')).toBe(5.2);
    expect(glucoseText(5.2, 'mmol_l')).toBe('5.2');
    expect(glucoseText(5, 'mmol_l')).toBe('5.0');
    expect(glucoseText(93.6, 'mg_dl')).toBe('94');
  });

  it('keeps the meaning of the number when the unit toggle flips', () => {
    expect(convertGlucose(94, 'mg_dl', 'mmol_l')).toBe(5.2);
    expect(convertGlucose(5.2, 'mmol_l', 'mg_dl')).toBe(94);
    expect(convertGlucose(100, 'mg_dl', 'mg_dl')).toBe(100);
  });

  it('steps by 1 mg/dL or 0.1 mmol/L without float noise', () => {
    expect(stepGlucose(5.2, 'mmol_l', 1)).toBe(5.3);
    expect(stepGlucose(0.2, 'mmol_l', 1)).toBe(0.3);
    expect(stepGlucose(94, 'mg_dl', -1)).toBe(93);
  });

  it('prints the Latin unit symbol', () => {
    expect(unitSymbol('mg_dl')).toBe('mg/dL');
    expect(unitSymbol('mmol_l')).toBe('mmol/L');
  });
});

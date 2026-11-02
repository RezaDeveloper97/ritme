import { describe, expect, it } from 'vitest';

import type { CycleView } from '@/entities/cycle';

import { whyThisDate } from './why';

const view = (source: string, basedOn: number | null, reasons: string[]) =>
  ({ forecast: { source, confidenceReasons: reasons }, confidenceReasons: [], calculatedValues: { basedOnCycles: basedOn } }) as unknown as CycleView;

describe('whyThisDate', () => {
  it('maps the forecast layer, basis and reasons', () => {
    expect(whyThisDate(view('recent_valid_cycles', 3, ['a', 'b']))).toEqual({ source: 'sourceRecent', basisCycles: 3, reasonCount: 2 });
    expect(whyThisDate(view('profile', null, []))?.source).toBe('sourceProfile');
    expect(whyThisDate(view('default', null, []))?.source).toBe('sourceDefault');
  });
  it('is null without a forecast', () => {
    expect(whyThisDate(null)).toBeNull();
    expect(whyThisDate({ forecast: null } as unknown as CycleView)).toBeNull();
  });
});

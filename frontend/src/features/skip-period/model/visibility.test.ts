import { describe, expect, it } from 'vitest';

import type { CycleTodayUi } from '@/entities/cycle';

import { canOfferSkip, type SkipOfferInput } from './visibility';

const ui = (o: Partial<CycleTodayUi>): CycleTodayUi => ({ ringAction: 'start', banner: null, showPregnancyTestButton: false, ...o }) as CycleTodayUi;
const base: SkipOfferInput = { isToday: true, todayUi: ui({}), predictedNext: '2026-10-20', todayIso: '2026-10-22', inPeriod: false };

describe('canOfferSkip', () => {
  it('offers on the ring start prompt once the expected day came', () => {
    expect(canOfferSkip(base)).toBe(true);
    expect(canOfferSkip({ ...base, todayIso: '2026-10-20' })).toBe(true);
  });
  it('offers on the overdue banner', () => {
    expect(canOfferSkip({ ...base, todayUi: ui({ ringAction: 'edit', banner: 'overdue' }) })).toBe(true);
  });
  it('hides before the expected day, off today, in a period, on a legacy payload, with ttc ring', () => {
    expect(canOfferSkip({ ...base, todayIso: '2026-10-19' })).toBe(false);
    expect(canOfferSkip({ ...base, isToday: false })).toBe(false);
    expect(canOfferSkip({ ...base, inPeriod: true })).toBe(false);
    expect(canOfferSkip({ ...base, todayUi: null })).toBe(false);
    expect(canOfferSkip({ ...base, ttcRing: true })).toBe(false);
    expect(canOfferSkip({ ...base, predictedNext: null })).toBe(false);
  });
  it('hides under ask_last_period and when neither entry point applies', () => {
    expect(canOfferSkip({ ...base, todayUi: ui({ banner: 'ask_last_period' }) })).toBe(false);
    expect(canOfferSkip({ ...base, todayUi: ui({ ringAction: 'edit' }) })).toBe(false);
  });
});

import { describe, expect, it } from 'vitest';

import { cycleViewSchema } from '@/entities/cycle/api/schema';

import { homeLimits } from './limits';

const view = (extra: Record<string, unknown>) => cycleViewSchema.parse({ date: '2026-07-18', ...extra });

describe('homeLimits', () => {
  it('changes nothing for a Laravel payload or no view', () => {
    const off = { hideFertility: false, withdrawalBleed: false, note: null, promptPregnancy: false };
    expect(homeLimits(null)).toEqual(off);
    expect(homeLimits(view({}))).toEqual(off);
  });

  it('pill: hides fertility, withdrawal bleed, hormonal note', () => {
    const r = homeLimits(
      view({
        warnings: ['hormonal_contraception_active'],
        limits: { fertility_suppressed: true, suppression_reason: 'hormonal_contraception' },
      }),
    );
    expect(r).toEqual({ hideFertility: true, withdrawalBleed: true, note: 'hormonal', promptPregnancy: false });
  });

  it('copper IUD: nothing suppressed, normal rows', () => {
    const r = homeLimits(view({ limits: { fertility_suppressed: false, suppression_reason: null } }));
    expect(r.hideFertility).toBe(false);
    expect(r.withdrawalBleed).toBe(false);
    expect(r.note).toBeNull();
  });

  it('postpartum not returned: hides fertility with the postpartum note, no withdrawal bleed', () => {
    const r = homeLimits(
      view({
        warnings: ['postpartum_cycles_not_returned'],
        limits: { fertility_suppressed: true, suppression_reason: 'postpartum_not_returned' },
      }),
    );
    expect(r).toEqual({ hideFertility: true, withdrawalBleed: false, note: 'postpartum', promptPregnancy: false });
  });

  it('age 46: calm note, fertility stays', () => {
    const r = homeLimits(view({ warnings: ['perimenopause_age_range'], limits: { fertility_suppressed: false } }));
    expect(r).toMatchObject({ hideFertility: false, note: 'age45' });
  });

  it('positive test: prompts pregnancy mode', () => {
    expect(homeLimits(view({ limits: { prompt_pregnancy_mode: true } })).promptPregnancy).toBe(true);
  });
});

import { describe, expect, it } from 'vitest';

import en from '../../../../messages/en.json';
import fa from '../../../../messages/fa.json';
import { HINT_KEYS, hintFor } from './hints';

const MENO_GROUPS = ['meno_score_items', 'meno_score_bands', 'meno_alerts', 'meno_tips', 'meno_checkup_groups'];

describe('menopause hints (00022_menopause, menopause.md §5)', () => {
  it('every meno_* group has its own hint', () => {
    expect(MENO_GROUPS.map((g) => hintFor(g).key)).toEqual([
      'menoScoreItems',
      'menoScoreBands',
      'menoAlerts',
      'menoTips',
      'menoCheckupGroups',
    ]);
  });
  it('examples follow the seeded meta shapes', () => {
    expect(hintFor('meno_score_items').example).toMatchObject({ domain: 'somatic', max: 4 });
    expect((hintFor('meno_score_items').example as { log: string[] }).log.every((s) => s.split('.').length === 3)).toBe(true);
    expect(Object.keys(hintFor('meno_score_bands').example ?? {})).toEqual(['min', 'max']);
    expect(hintFor('meno_alerts').example).toMatchObject({ severity: 'urgent', stages: ['meno', 'post'] });
    expect(hintFor('meno_tips').example).toMatchObject({ placement: 'treatment_lifestyle', goal_unit: 'minutes' });
    expect(Array.isArray((hintFor('meno_checkup_groups').example as { checkups: unknown }).checkups)).toBe(true);
  });
});

describe('shipped groups from other epics', () => {
  it('missed_pill_rules carries methods (CB-CONTRA-01)', () => {
    expect(hintFor('missed_pill_rules').example).toMatchObject({ methods: ['combined_pill'], missed: 1, severity: 'caution' });
  });
  it('condition groups (CB-COND-01)', () => {
    expect(hintFor('condition_programs').example).toHaveProperty('logs.fa');
    expect(hintFor('pain_associated').example).toEqual({ log: 'symptoms.digestive.bloating' });
    expect(hintFor('condition_alerts').example).toHaveProperty('hotlines.0.number', '1480');
    expect(hintFor('pain_types')).toEqual({ key: 'painTypes', example: null });
    expect(hintFor('pmdd_items')).toEqual({ key: 'pmddItems', example: null });
  });
  it('suffix conventions still cover unlisted groups', () => {
    expect(hintFor('pelvic_alerts').key).toBe('alerts');
    expect(hintFor('teen_score_items').example).toEqual({ domain: 'somatic', max: 4 });
  });
});

describe('hint messages', () => {
  it('every hint key has fa and en text', () => {
    for (const messages of [en, fa]) {
      const hints = messages.catalog.hint as Record<string, string>;
      for (const key of HINT_KEYS) expect(hints[key], key).toBeTruthy();
    }
  });
});

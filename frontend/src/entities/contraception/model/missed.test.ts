import { describe, expect, it } from 'vitest';

import { missedRulesSchema } from '../api/schema';
import { daysUntil, methodReminder, missedGuide } from './missed';
import type { MethodReminder } from './types';

/* Mirrors GET /catalog/missed_pill_rules (fa) for the CB-CONTRA-01 seed. */
const rules = missedRulesSchema.parse({
  group: 'missed_pill_rules',
  locale: 'fa',
  items: [
    {
      code: 'combined_one',
      title: 'یک قرص',
      body: 'راهنمای عمومی',
      meta: { methods: ['combined_pill'], missed: 1, severity: 'caution', steps: ['الف', 'ب'] },
      audiences: null,
      needs_review: true,
    },
    {
      code: 'combined_two_plus',
      title: 'دو قرص یا بیشتر',
      body: 'راهنمای عمومی',
      meta: { methods: ['combined_pill'], missed: 2, severity: 'caution', steps: ['الف', { fa: 'ب', en: 'b' }, 'ج'] },
      audiences: null,
      needs_review: true,
    },
    {
      code: 'week1_unprotected',
      title: 'هفته اول',
      body: 'اضطراری',
      meta: { methods: ['combined_pill'], severity: 'urgent', pack_week: 1 },
      audiences: null,
      needs_review: true,
    },
    {
      code: 'progestin_note',
      title: 'تک‌هورمونی',
      body: 'قواعد فرق دارد',
      meta: { methods: ['progestin_pill'], severity: 'caution', steps: ['الف'] },
      audiences: null,
      needs_review: true,
    },
    { code: 'odd', title: null, body: null, meta: null, audiences: null, needs_review: false },
  ],
});

describe('missedRulesSchema', () => {
  it('maps meta and picks an unpicked translation', () => {
    expect(rules[1]!.steps).toEqual(['الف', 'ب', 'ج']);
    expect(rules[2]).toMatchObject({ severity: 'urgent', missed: null, packWeek: 1, steps: [] });
    expect(rules[4]).toMatchObject({ severity: 'info', methods: null, steps: [] });
  });
});

describe('missedGuide', () => {
  it('gives the combined pill two count chips and the danger card', () => {
    const guide = missedGuide(rules, 'combined_pill');
    expect(guide.countRules.map((r) => r.code)).toEqual(['combined_one', 'combined_two_plus']);
    expect(guide.urgent.map((r) => r.code)).toEqual(['week1_unprotected']);
    expect(guide.baseRule).toBeNull();
  });

  it('gives the progestogen-only pill its count-free steps and no chips', () => {
    const guide = missedGuide(rules, 'progestin_pill');
    expect(guide.countRules).toEqual([]);
    expect(guide.baseRule?.code).toBe('progestin_note');
    expect(guide.urgent).toEqual([]);
  });
});

describe('methodReminder / daysUntil', () => {
  const reminder = (kind: string): MethodReminder => ({
    kind,
    reminderId: 1,
    type: 'custom',
    title: '',
    dueOn: null,
    scheduledAt: null,
    recurrence: 'monthly',
    isActive: true,
  });

  it('finds a reminder by kind', () => {
    expect(methodReminder([reminder('iud_string_check')], 'iud_string_check')?.kind).toBe('iud_string_check');
    expect(methodReminder([reminder('iud_string_check')], 'iud_followup')).toBeNull();
  });

  it('counts whole days across months', () => {
    expect(daysUntil('2026-10-19', '2026-10-01')).toBe(18);
    expect(daysUntil('2026-09-30', '2026-10-01')).toBe(-1);
    expect(daysUntil('2027-03-21', '2027-03-20')).toBe(1);
  });
});

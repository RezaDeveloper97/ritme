import { describe, expect, it } from 'vitest';

import type { DailyMessage } from '@/entities/message';
import type { WeekTip } from '@/entities/pregnancy';

import { MAX_EXTRAS, selectPregnancySmartTip } from './smart-tip';

const week: WeekTip = { week: 12, title: 'هفتهٔ ۱۲', body: 'جنین اندازهٔ یک لیمو است.', readMinutes: 2, link: null };

const daily = (over: Partial<DailyMessage> = {}): DailyMessage => ({
  mode: 'pregnancy',
  date: '2026-11-25',
  phase: null,
  phaseLabel: null,
  cycleDay: null,
  primary: { shortMessage: '', longMessage: '', actionSuggestion: 'آب بنوش', dos: [], donts: [], override: null },
  correlations: [],
  patterns: [],
  supplements: { nutritionFocus: '', nutritionTip: 'فولات', sleepFocus: '', sleepTips: ['به پهلو بخواب'], exerciseTip: 'پیاده‌روی' },
  tips: [],
  ...over,
});

describe('selectPregnancySmartTip', () => {
  it('shows the week tip with the engine action and the general extras', () => {
    expect(selectPregnancySmartTip(week, daily())).toEqual({
      kind: 'week',
      title: 'هفتهٔ ۱۲',
      body: 'جنین اندازهٔ یک لیمو است.',
      action: 'آب بنوش',
      extras: ['فولات', 'به پهلو بخواب', 'پیاده‌روی'],
    });
  });

  it('a symptom logged today takes the card, correlations come first among the extras', () => {
    const tip = selectPregnancySmartTip(
      week,
      daily({
        primary: {
          shortMessage: '',
          longMessage: '',
          actionSuggestion: '',
          dos: [],
          donts: [],
          override: { type: 'nausea', label: 'حالت تهوع', message: 'وعده‌های کوچک بخور.', action: 'زنجبیل' },
        },
        correlations: [{ type: 'first_trimester_combo', insightMessage: 'تهوع و خستگی با هم', action: '', isPremiumOnly: false }],
      }),
    );
    expect(tip).toMatchObject({ kind: 'symptom', title: 'حالت تهوع', body: 'وعده‌های کوچک بخور.', action: 'زنجبیل' });
    expect(tip?.extras[0]).toBe('تهوع و خستگی با هم');
    expect(tip?.extras).toHaveLength(MAX_EXTRAS);
  });

  it('keeps the week tip alone while the bundle is missing, and ignores a cycle bundle', () => {
    expect(selectPregnancySmartTip(week, null)?.extras).toEqual([]);
    expect(selectPregnancySmartTip(week, daily({ mode: 'cycle' }))).toMatchObject({ action: '', extras: [] });
  });

  it('falls back to the engine copy without a week tip, and to nothing without any text', () => {
    expect(selectPregnancySmartTip(null, daily({ primary: { ...daily().primary, longMessage: 'سه‌ماههٔ دوم' } }))).toMatchObject({
      kind: 'week',
      title: '',
      body: 'سه‌ماههٔ دوم',
    });
    expect(selectPregnancySmartTip(null, null)).toBeNull();
  });

  it('never repeats a line', () => {
    const tip = selectPregnancySmartTip(week, daily({ supplements: { ...daily().supplements, nutritionTip: 'آب بنوش', exerciseTip: 'به پهلو بخواب' } }));
    expect(tip?.extras).toEqual(['به پهلو بخواب']);
  });
});

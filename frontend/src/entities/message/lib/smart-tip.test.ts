import { describe, expect, it } from 'vitest';

import { dailyMessageSchema } from '../api/schema';
import type { DailyMessage } from '../model/types';
import { selectSmartTip } from './smart-tip';

const message = (over: Partial<DailyMessage> = {}): DailyMessage => ({
  mode: 'cycle',
  date: '2026-07-25',
  phase: 'luteal',
  phaseLabel: 'لوتیال',
  cycleDay: 21,
  primary: { shortMessage: '', longMessage: '', actionSuggestion: '', dos: [], donts: [] },
  correlations: [],
  patterns: [],
  supplements: {
    nutritionFocus: '',
    nutritionTip: '',
    sleepFocus: '',
    sleepTips: [],
    exerciseTip: '',
  },
  tips: [],
  ...over,
});

describe('selectSmartTip', () => {
  it('returns nothing without a message, so the card can stay hidden', () => {
    expect(selectSmartTip(undefined)).toBeNull();
    expect(selectSmartTip(message())).toBeNull();
  });

  it('prefers the personalized long message and its action', () => {
    const tip = selectSmartTip(
      message({
        primary: {
          shortMessage: 'کوتاه',
          longMessage: 'بلند',
          actionSuggestion: 'آب بنوش',
          dos: [],
          donts: [],
        },
      }),
    );
    expect(tip).toMatchObject({ body: 'بلند', action: 'آب بنوش', source: 'message' });
  });

  it('falls back to a detected correlation when no primary message exists', () => {
    const tip = selectSmartTip(
      message({
        correlations: [
          { type: 'sleep_mood', insightMessage: 'خواب و خلق', action: 'زودتر بخواب', isPremiumOnly: false },
        ],
      }),
    );
    expect(tip).toMatchObject({ body: 'خواب و خلق', action: 'زودتر بخواب', source: 'correlation' });
  });

  it('falls back to a pattern, then to the supplement modules', () => {
    const pattern = selectSmartTip(
      message({
        patterns: [{ patternType: 'irregular', alertLevel: 'info', message: 'روند', recommendation: 'پیگیری کن' }],
      }),
    );
    expect(pattern).toMatchObject({ body: 'روند', action: 'پیگیری کن', source: 'pattern' });

    const supplement = selectSmartTip(
      message({ supplements: { ...message().supplements, nutritionTip: 'آهن بخور' } }),
    );
    expect(supplement).toMatchObject({ body: 'آهن بخور', source: 'supplement' });
  });

  it('never repeats the body or the action in the extras', () => {
    const tip = selectSmartTip(
      message({
        primary: { shortMessage: '', longMessage: 'بلند', actionSuggestion: 'اقدام', dos: [], donts: [] },
        correlations: [{ type: 'x', insightMessage: 'بلند', action: '', isPremiumOnly: false }],
        tips: ['اقدام', 'نکته تازه'],
      }),
    );
    expect(tip?.extras).toEqual(['نکته تازه']);
  });
});

describe('dailyMessageSchema', () => {
  it('parses an empty engine result (PHP sends [] for empty objects)', () => {
    const parsed = dailyMessageSchema.parse({
      mode: 'cycle',
      date: '2026-07-25',
      context_info: [],
      primary_message: [],
      correlations: [],
      patterns: [],
      supplements: [],
      tips: [],
    });
    expect(parsed.primary.longMessage).toBe('');
    expect(selectSmartTip(parsed)).toBeNull();
  });

  it('keeps correlations, patterns and supplement tips from a full payload', () => {
    const parsed = dailyMessageSchema.parse({
      mode: 'cycle',
      date: '2026-07-25',
      context_info: { phase: 'luteal', phase_label: 'لوتیال', cycle_day: 21 },
      primary_message: { long_message: 'بلند', action_suggestion: 'اقدام', dos: {}, donts: [] },
      correlations: [
        { type: 'sleep_mood', insight_message: 'خواب و خلق', action: 'زودتر بخواب', is_premium_only: false },
      ],
      patterns: [
        { pattern_type: 'irregular', alert_level: 'warning', message: 'روند', recommendation: 'پیگیری' },
      ],
      supplements: {
        nutrition: { focus: 'آهن', tip: 'عدس بخور', foods: ['عدس'], avoid: [] },
        sleep: { quality_focus: 'خواب عمیق', tips: ['زود بخواب'], recommended_hours: '۸' },
        exercise: { tip: 'پیاده‌روی', intensity: 'low' },
      },
      tips: [],
    });

    expect(parsed.correlations[0]?.insightMessage).toBe('خواب و خلق');
    expect(parsed.patterns[0]?.alertLevel).toBe('warning');
    expect(parsed.supplements.sleepTips).toEqual(['زود بخواب']);
    expect(parsed.primary.dos).toEqual([]);
  });
});

import { describe, expect, it } from 'vitest';

import { MESSAGES_FLOW, PROFILE_PERI, TODAY_EMPTY, TODAY_FLOW } from './fixtures.test-data';
import { toMenopauseProfileBody } from './queries';
import { menopauseMessagesSchema, menopauseProfileSchema, menopauseTodaySchema } from './schema';

describe('menopause schemas (CB-MENO-02 / CB-MENO-12 goldens)', () => {
  it('parses the home read model with a score, a bleeding alert and checkups', () => {
    const today = menopauseTodaySchema.parse(TODAY_FLOW);
    expect(today.profile.stage).toBe('meno');
    expect(today.profile.monthsWithoutPeriod).toBe(13);
    expect(today.profile.tip?.code).toBe('stage_meno');
    expect(today.hotFlashes).toEqual({ count: 2, nightCount: 1, running: null });
    expect(today.nightSweats.count).toBe(1);
    expect(today.score.latest).toMatchObject({ total: 14, max: 44, band: { code: 'moderate' }, delta: null });
    expect(today.score.trend).toHaveLength(6);
    expect(today.score.trend.at(-1)).toEqual({ month: '2026-09-23', total: 14, band: 'moderate' });
    expect(today.checkups).toHaveLength(1);
    expect(today.bleeding.alert).toBe(true);
    expect(today.bleeding.alertItem?.code).toBe('postmenopausal_bleeding');
  });

  it('parses an empty home that still needs the stage', () => {
    const today = menopauseTodaySchema.parse(TODAY_EMPTY);
    expect(today.profile.needsStage).toBe(true);
    expect(today.profile.stage).toBeNull();
    expect(today.score.latest).toBeNull();
    expect(today.sleep).toBeNull();
    expect(today.treatment).toEqual([]);
  });

  it('reads a lifestyle weekly goal on the home treatment card (CB-MENO-13)', () => {
    const row = { taken_today: false, days_taken: 0, days: 7, review_on: null };
    const today = menopauseTodaySchema.parse({
      ...(TODAY_EMPTY as Record<string, unknown>),
      treatment: [
        { ...row, id: 1, kind: 'hrt', name: 'Gel', weekly_goal: null, goal_unit: null, amount: 0 },
        { ...row, id: 2, kind: 'lifestyle', name: 'Walk', weekly_goal: 150, goal_unit: 'minutes', amount: 30 },
      ],
    });
    expect(today.treatment.map((x) => x.goal)).toEqual([null, { target: 150, unit: 'minutes', amount: 30 }]);
  });

  it('keeps the stored answer apart from the effective stage', () => {
    const profile = menopauseProfileSchema.parse(PROFILE_PERI);
    expect(profile).toMatchObject({ stage: 'peri', storedStage: 'peri', suggestedStage: 'meno', lastPeriod: '2025-08-01' });
  });

  it('parses the menopause messages in order', () => {
    const messages = menopauseMessagesSchema.parse(MESSAGES_FLOW);
    expect(messages.map((m) => m.key)).toEqual(['postmenopausal_bleeding', 'checkup_due', 'stage_meno']);
    expect(messages[0]).toMatchObject({ kind: 'alert', priority: 'high', link: '/menopause/alert' });
  });

  it('drops a malformed message instead of failing the list', () => {
    const messages = menopauseMessagesSchema.parse({ messages: [{ key: 'x' }, { key: 'tip', kind: 'tip', title: 'T', body: null, action: null, link: null }] });
    expect(messages.map((m) => m.key)).toEqual(['tip']);
  });

  it('sends every stage field, null clearing', () => {
    expect(toMenopauseProfileBody({ stage: 'unsure', lastPeriod: '2025-08-01', surgical: null, hrt: true })).toEqual({
      stage: 'unsure',
      last_period: '2025-08-01',
      surgical: null,
      hrt: true,
    });
  });
});

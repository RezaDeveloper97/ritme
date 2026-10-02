import { describe, expect, it } from 'vitest';

import type { PostpartumAlert, PostpartumRecovery } from '@/entities/postpartum';

import { durationKey, headlineWeek, homeAlerts, sixWeekVisitDue, splitScheduled, tileSummary } from './home';
import { pressedChips, toggleChip } from './mood';
import { birthDateProblem } from './setup';

const recovery = (over: Partial<PostpartumRecovery> = {}): PostpartumRecovery => ({
  date: '2026-09-23',
  lochiaAmount: null,
  lochiaColor: null,
  painLevel: null,
  painLocations: [],
  breasts: null,
  feedsCount: null,
  sleepHours: null,
  alerts: [],
  ...over,
});
const alert = (key: string): PostpartumAlert => ({ key, level: 'info', title: key, body: null, actionLabel: null, action: null });

describe('postpartum hero', () => {
  it('picks the duration line', () => {
    expect(durationKey({ weeks: 2, days: 3 })).toBe('weeksDays');
    expect(durationKey({ weeks: 2, days: 0 })).toBe('weeks');
    expect(durationKey({ weeks: 0, days: 5 })).toBe('days');
  });

  it('uses completed weeks for the headline (QUESTIONS #97)', () => {
    expect(headlineWeek({ weeks: 2 })).toBe(2);
    expect(headlineWeek({ weeks: 0 })).toBe(1);
  });

  it('suggests the six-week visit only before day 42', () => {
    expect(sixWeekVisitDue({ daysSinceBirth: 17, puerperiumDays: 42 })).toBe(true);
    expect(sixWeekVisitDue({ daysSinceBirth: 42, puerperiumDays: 42 })).toBe(false);
  });
});

describe('postpartum tiles and alerts', () => {
  it('summarises today, null when not logged', () => {
    expect(tileSummary(null)).toEqual({ bleeding: null, feeds: null, sleep: null });
    expect(tileSummary(recovery({ lochiaAmount: 'light', lochiaColor: 'pink_brown', feedsCount: 8, sleepHours: 4.5 }))).toEqual({
      bleeding: { amount: 'light', color: 'pink_brown' },
      feeds: 8,
      sleep: 4.5,
    });
  });

  it('puts recovery alerts first and drops duplicates', () => {
    const list = homeAlerts([alert('checkin_full'), alert('heavy_bleeding')], recovery({ alerts: [alert('heavy_bleeding')] }));
    expect(list.map((a) => a.key)).toEqual(['heavy_bleeding', 'checkin_full']);
  });

  it('splits a scheduled time', () => {
    expect(splitScheduled('2026-10-14 09:00:00')).toEqual({ date: '2026-10-14', time: '09:00' });
    expect(splitScheduled(null)).toBeNull();
  });
});

describe('mood chips on the taxonomy rows', () => {
  it('reads what the log sheet stored', () => {
    expect([...pressedChips({ mood: { moods: ['calm', 'sad'] }, appetite_energy: { energy: 'very_low' } })].sort()).toEqual([
      'good',
      'sad',
      'tired',
    ]);
    expect(pressedChips(undefined).size).toBe(0);
  });

  it('toggles a mood code on and off without touching the others', () => {
    const on = toggleChip({ mood: { moods: ['sad'] } }, 'restless');
    expect(on.changes).toEqual({ mood: { moods: ['sad', 'anxious'] } });
    const off = toggleChip({ mood: { moods: ['happy', 'calm'] } }, 'good');
    expect(off.changes).toEqual({ mood: { moods: null } });
    expect(off.draft).toEqual({});
  });

  it('tired sets and clears the energy level', () => {
    expect(toggleChip({}, 'tired').changes).toEqual({ appetite_energy: { energy: 'low' } });
    const off = toggleChip({ appetite_energy: { energy: 'low', appetite: 'normal' } }, 'tired');
    expect(off.changes).toEqual({ appetite_energy: { energy: null } });
    expect(off.draft).toEqual({ appetite_energy: { appetite: 'normal' } });
  });
});

describe('activation birth date', () => {
  it('refuses the future and more than a year ago', () => {
    expect(birthDateProblem('2026-09-24', '2026-09-23')).toBe('future');
    expect(birthDateProblem('2025-09-01', '2026-09-23')).toBe('tooOld');
    expect(birthDateProblem('2026-09-06', '2026-09-23')).toBeNull();
  });
});

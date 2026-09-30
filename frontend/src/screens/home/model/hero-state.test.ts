import { describe, expect, it } from 'vitest';

import { heroState, phaseFromMainPhase, type HeroInput } from './hero-state';
import { summarizeTodayLog } from './today-log';

const base: HeroInput = {
  isToday: true,
  inPeriod: false,
  daysLeft: 11,
  daysLate: 0,
  periodOpen: false,
  cycleDay: 18,
  periodLength: 5,
  startDismissed: false,
  endDismissed: false,
};

describe('heroState', () => {
  it('normal mid-cycle', () => {
    expect(heroState(base)).toEqual({ variant: 'normal', startPrompt: false, endPrompt: false, endMissing: false });
  });

  it('near period asks whether it started, until dismissed', () => {
    expect(heroState({ ...base, daysLeft: 1 })).toMatchObject({ variant: 'near', startPrompt: true });
    expect(heroState({ ...base, daysLeft: 1, startDismissed: true })).toMatchObject({ variant: 'near', startPrompt: false });
    expect(heroState({ ...base, daysLeft: 1, isToday: false }).startPrompt).toBe(false);
  });

  it('late period is its own variant with the start prompt', () => {
    expect(heroState({ ...base, daysLeft: null, daysLate: 4 })).toMatchObject({ variant: 'late', startPrompt: true });
  });

  it('during an open period asks whether it ended; past the usual length flags the missing end', () => {
    const during = { ...base, inPeriod: true, periodOpen: true, cycleDay: 3 };
    expect(heroState(during)).toMatchObject({ variant: 'during', endPrompt: true, endMissing: false });
    expect(heroState({ ...during, cycleDay: 6 })).toMatchObject({ endPrompt: true, endMissing: true });
    expect(heroState({ ...during, endDismissed: true })).toMatchObject({ endPrompt: false, endMissing: false });
    expect(heroState({ ...during, periodOpen: false })).toMatchObject({ variant: 'during', endPrompt: false });
  });
});

describe('summarizeTodayLog', () => {
  it('reads bleeding, symptoms and moods', () => {
    expect(
      summarizeTodayLog({
        log_date: '2026-10-01',
        bleeding_intensity: 'medium',
        breast_pain_intensity: 'low',
        bloating_intensity: 'high',
        diarrhea: false,
        fatigue: true,
        moods: ['calm'],
      }),
    ).toEqual({ bleeding: 'medium', symptoms: ['breast_pain_intensity', 'bloating_intensity', 'fatigue'], moods: ['calm'] });
  });

  it('falls back to spotting and handles an empty day', () => {
    expect(summarizeTodayLog({ spotting: true }).bleeding).toBe('spotting');
    expect(summarizeTodayLog(null)).toEqual({ bleeding: null, symptoms: [], moods: [] });
  });
});

describe('phaseFromMainPhase', () => {
  it('maps the engine main phase onto the page phase', () => {
    expect(phaseFromMainPhase('menstrual')).toBe('period');
    expect(phaseFromMainPhase('fertile')).toBe('fertile');
    expect(phaseFromMainPhase('period_expected')).toBe('luteal');
    expect(phaseFromMainPhase('unknown')).toBeNull();
    expect(phaseFromMainPhase(null)).toBeNull();
  });
});

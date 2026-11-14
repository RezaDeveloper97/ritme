import { describe, expect, it } from 'vitest';

import { heroState, phaseFromMainPhase, type HeroInput } from './hero-state';
import type { LogCategory, LogParam } from '@/entities/health-log';

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
    const during = { ...base, inPeriod: true, periodOpen: true, cycleDay: 4 };
    // Before the day ahead of the usual length (5 → day 4) the ring keeps «ویرایش», like the engine.
    expect(heroState({ ...during, cycleDay: 1 })).toMatchObject({ variant: 'during', endPrompt: false, endMissing: false });
    expect(heroState({ ...during, cycleDay: 3 })).toMatchObject({ variant: 'during', endPrompt: false });
    expect(heroState({ ...during, cycleDay: null })).toMatchObject({ endPrompt: true, endMissing: false });
    expect(heroState(during)).toMatchObject({ variant: 'during', endPrompt: true, endMissing: false });
    expect(heroState({ ...during, cycleDay: 6 })).toMatchObject({ endPrompt: true, endMissing: true });
    expect(heroState({ ...during, endDismissed: true })).toMatchObject({ endPrompt: false, endMissing: false });
    expect(heroState({ ...during, periodOpen: false })).toMatchObject({ variant: 'during', endPrompt: false });
  });
});

describe('summarizeTodayLog', () => {
  const param = (code: string, type: LogParam['type'], label: string, options: [string, string][] = []): LogParam => ({
    code, type, label, modes: [], detail: false, alert: false,
    options: options.map(([value, l]) => ({ value, label: l, modes: null, legacyOnly: false })),
    levels: [], score: null, range: null, scale: null, unit: null, maxLength: null, dynamic: false, source: null,
  });
  const cat = (code: string, params: LogParam[]): LogCategory => ({
    code, label: code, group: { value: 'body', label: 'body' }, modes: [], conditions: {}, params,
  });
  const taxonomy = [
    cat('bleeding', [param('flow', 'single', 'شدت', [['medium', 'متوسط']]), param('spotting', 'bool', 'لکه‌بینی')]),
    cat('symptoms', [
      param('digestive', 'items', 'گوارشی', [['diarrhea', 'اسهال'], ['constipation', 'یبوست'], ['nausea', 'تهوع']]),
      param('general', 'items', 'عمومی', [['fatigue', 'خستگی'], ['dizziness', 'سرگیجه'], ['chills', 'لرز']]),
    ]),
    cat('mood', [param('moods', 'multi', 'حال', [['calm', 'آرام'], ['anxious', 'مضطرب']])]),
  ];
  const labels = { extraLabels: { custom_3: 'سوزش معده' }, unknown: '؟' };
  const yes = { level: 'yes', score: null };

  it('counts every logged symptom across params, skipping «ندارم»', () => {
    const s = summarizeTodayLog(
      {
        bleeding: { flow: 'medium' },
        symptoms: {
          digestive: { diarrhea: yes, constipation: yes, nausea: { level: 'no', score: null } },
          general: { fatigue: yes, dizziness: { level: 'mild', score: null }, chills: yes, custom_3: yes },
        },
        mood: { moods: ['anxious', 'calm'] },
      },
      taxonomy,
      labels,
    );
    expect(s.bleeding).toBe('شدت: متوسط');
    expect(s.symptoms).toEqual(['اسهال', 'یبوست', 'خستگی', 'سرگیجه', 'لرز', 'سوزش معده']);
    expect(s.moods).toEqual(['مضطرب', 'آرام']);
  });

  it('falls back to spotting and handles an empty day', () => {
    expect(summarizeTodayLog({ bleeding: { spotting: true } }, taxonomy, labels).bleeding).toBe('لکه‌بینی');
    expect(summarizeTodayLog(null, taxonomy, labels)).toEqual({ bleeding: null, symptoms: [], moods: [] });
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

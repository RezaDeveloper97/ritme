import { describe, expect, it } from 'vitest';

import type { IvfCycle } from '@/entities/ivf';

import {
  clockOf,
  dayOf,
  hasAmount,
  stageCopy,
  stageDayChip,
  stepMeta,
  suggestedSiteFor,
  unitKey,
  whenKey,
} from './home';

const CYCLE: IvfCycle = {
  id: 1,
  number: 1,
  protocol: null,
  stage: 'stim',
  status: 'open',
  startedOn: '2026-09-10',
  stimStartedOn: '2026-09-17',
  retrievalAt: null,
  transferAt: null,
  nextScanAt: null,
  betaOn: null,
  notifyCompanion: false,
  stageDay: 7,
  daysToBeta: null,
  timeline: [],
};

describe('IVF home helpers', () => {
  it('maps known units and keeps the rest as typed', () => {
    expect(unitKey('iu')).toBe('iu');
    expect(unitKey(' IU ')).toBe('iu');
    expect(unitKey('pen')).toBe('other');
    expect(unitKey(null)).toBe('other');
    expect(hasAmount({ dose: '150' })).toBe(true);
    expect(hasAmount({ dose: ' ' })).toBe(false);
    expect(hasAmount({ dose: null })).toBe(false);
  });

  it('prefers the catalog copy of a stage', () => {
    expect(stageCopy('prep', [{ code: 'prep', title: 'A', body: 'B' }])).toEqual({ title: 'A', hint: 'B' });
    expect(stageCopy('stim', [{ code: 'prep', title: 'A', body: 'B' }])).toEqual({ title: null, hint: null });
    expect(stageCopy('stim', undefined)).toEqual({ title: null, hint: null });
  });

  it('shows hints on done/current steps and dates on upcoming ones', () => {
    expect(stepMeta({ stage: 'prep', status: 'done', date: '2026-09-10' })).toBe('hint');
    expect(stepMeta({ stage: 'stim', status: 'current', date: null })).toBe('hint');
    expect(stepMeta({ stage: 'test', status: 'todo', date: '2026-10-10' })).toBe('date');
    expect(stepMeta({ stage: 'tww', status: 'todo', date: null })).toBeNull();
  });

  it('builds the stage-day chip only for an open cycle with a day', () => {
    expect(stageDayChip(CYCLE)).toEqual({ day: 7, stage: 'stim' });
    expect(stageDayChip({ ...CYCLE, stageDay: null })).toBeNull();
    expect(stageDayChip({ ...CYCLE, status: 'closed' })).toBeNull();
  });

  it('reads clocks and days of wall-clock values', () => {
    expect(clockOf('2026-09-24 09:00:00')).toBe('09:00');
    expect(clockOf('20:00')).toBe('20:00');
    expect(clockOf('bad')).toBe('');
    expect(dayOf('2026-09-24 09:00:00')).toBe('2026-09-24');
    expect(whenKey(0)).toBe('today');
    expect(whenKey(1)).toBe('tomorrow');
    expect(whenKey(5)).toBe('date');
    expect(whenKey(null)).toBe('date');
  });

  it('logs a home injection with the suggested site (CB-IVF-06b)', () => {
    const sites = { codes: ['abdomen_upper_right', 'thigh_left'], last: null, suggested: 'thigh_left' };
    expect(suggestedSiteFor({ route: 'subcutaneous' }, sites)).toBe('thigh_left');
    expect(suggestedSiteFor({ route: 'intramuscular' }, { ...sites, suggested: 'gone' })).toBe('abdomen_upper_right');
    expect(suggestedSiteFor({ route: 'vaginal' }, sites)).toBeNull();
    expect(suggestedSiteFor({ route: 'subcutaneous' }, undefined)).toBeNull();
    expect(suggestedSiteFor({ route: 'subcutaneous' }, { codes: [], last: null, suggested: null })).toBeNull();
  });
});

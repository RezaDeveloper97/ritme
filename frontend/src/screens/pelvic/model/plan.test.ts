import { describe, expect, it } from 'vitest';

import { PELVIC_UTI_SYMPTOMS } from '@/entities/pelvic';

import { nextLeak, sessionMinutes, toggleSymptom, weekDotStates } from './plan';

// 2026-09-23 is a Wednesday; its Saturday-start week is 09-19 … 09-25 (the CB-PELV-01 golden).
const WEEK = ['19', '20', '21', '22', '23', '24', '25'].map((d) => ({ date: `2026-09-${d}`, done: d === '20' || d === '23' }));

describe('weekDotStates', () => {
  it('maps the API week onto Saturday-first columns', () => {
    expect(weekDotStates(WEEK, '2026-09-23', 'fa')).toEqual({
      states: ['missed', 'done', 'missed', 'missed', 'done', 'future', 'future'],
      todayIndex: 4,
    });
  });

  it('shifts to Sunday-first columns', () => {
    const { states, todayIndex } = weekDotStates(WEEK, '2026-09-23', 'en');
    expect(todayIndex).toBe(3);
    expect(states).toEqual(['done', 'missed', 'missed', 'done', 'future', 'future', 'future']);
  });

  it('keeps an untrained today as future', () => {
    expect(weekDotStates([], '2026-09-19', 'fa').states[0]).toBe('future');
  });
});

describe('diary helpers', () => {
  it('toggles the leak answer', () => {
    expect(nextLeak(null, 'cough')).toBe('cough');
    expect(nextLeak('cough', 'cough')).toBeNull();
    expect(nextLeak('cough', 'none')).toBe('none');
  });

  it('toggles symptoms in API order', () => {
    expect(toggleSymptom(['cloudy_odor'], 'burning', PELVIC_UTI_SYMPTOMS)).toEqual(['burning', 'cloudy_odor']);
    expect(toggleSymptom(['burning'], 'burning', PELVIC_UTI_SYMPTOMS)).toEqual([]);
  });

  it('rounds the session to minutes', () => {
    expect(sessionMinutes(300)).toBe(5);
    expect(sessionMinutes(20)).toBe(1);
  });
});

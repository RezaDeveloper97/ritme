import { describe, expect, it } from 'vitest';

import type { MenopauseFlash } from '@/entities/menopause';

import { detailsOf, flashElapsedSeconds, flashLength, severityIndex, startClock, toggleTrigger } from './flash';

describe('hot-flash timer helpers (CB-MENO-07)', () => {
  it('adds local ticks to the server elapsed and caps at an hour', () => {
    expect(flashElapsedSeconds(100, 2500)).toBe(102);
    expect(flashElapsedSeconds(3590, 60_000)).toBe(3600);
    expect(flashElapsedSeconds(0, -500)).toBe(0);
  });

  it('keeps «unknown» exclusive', () => {
    expect(toggleTrigger(['caffeine'], 'stress')).toEqual(['caffeine', 'stress']);
    expect(toggleTrigger(['caffeine', 'stress'], 'unknown')).toEqual(['unknown']);
    expect(toggleTrigger(['unknown'], 'warm_room')).toEqual(['warm_room']);
    expect(toggleTrigger(['caffeine', 'stress'], 'caffeine')).toEqual(['stress']);
  });

  it('reads the start clock from the API string, not the device zone', () => {
    expect(startClock('2026-09-23T03:20:00+03:30')).toBe('03:20');
    expect(startClock('nonsense')).toBeNull();
  });

  it('shows minutes from a minute up, else seconds', () => {
    expect(flashLength(102)).toEqual({ unit: 'minutes', value: 2 });
    expect(flashLength(150)).toEqual({ unit: 'minutes', value: 3 });
    expect(flashLength(42)).toEqual({ unit: 'seconds', value: 42 });
  });

  it('orders severities on the 4-step scale', () => {
    expect(severityIndex('mild')).toBe(0);
    expect(severityIndex('very_severe')).toBe(3);
    expect(severityIndex(null)).toBe(-1);
  });

  it('copies the details of a flash', () => {
    const flash: MenopauseFlash = {
      id: 1,
      startedAt: '2026-09-23T10:00:00+03:30',
      running: true,
      durationS: null,
      elapsedS: 5,
      severity: 'severe',
      night: false,
      sweat: true,
      triggers: ['caffeine'],
    };
    const details = detailsOf(flash);
    expect(details).toEqual({ severity: 'severe', sweat: true, triggers: ['caffeine'] });
    expect(details.triggers).not.toBe(flash.triggers);
  });
});

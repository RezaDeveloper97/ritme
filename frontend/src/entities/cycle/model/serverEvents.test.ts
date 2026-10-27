import { describe, expect, it } from 'vitest';

import type { CycleMonthDay } from './types';

import { serverFertileEvents } from './serverEvents';

const day = (date: string, over: Partial<CycleMonthDay> = {}): CycleMonthDay => ({
  date, mainPhase: 'follicular', fertilityLevel: null, cycleDay: null, estimatedOvulationDay: null,
  phase: null, isPredicted: false, isPeriod: false, periodKind: null, daysToPeriod: null, daysLate: null, ...over,
});

describe('serverFertileEvents', () => {
  it('groups server fertile days and takes the ovulation day from the day fields', () => {
    const days = [
      day('2026-10-10', { mainPhase: 'fertile', cycleDay: 12, estimatedOvulationDay: 14 }),
      day('2026-10-11', { mainPhase: 'fertile', cycleDay: 13, estimatedOvulationDay: 14 }),
      day('2026-10-12', { mainPhase: 'fertile', cycleDay: 14, estimatedOvulationDay: 14 }),
    ];
    expect(serverFertileEvents(null, days, [], '2026-10-11', false)).toEqual({
      ovulation: '2026-10-12',
      fertile: { start: '2026-10-10', end: '2026-10-12' },
    });
  });

  it('moves on to future_cycles once the current window is behind', () => {
    const days = [day('2026-10-10', { mainPhase: 'fertile' })];
    const future = [{ k: 2, periodStart: '2026-11-20', ovulation: '2026-11-06', fertileStart: '2026-11-01', fertileEnd: '2026-11-06' }];
    expect(serverFertileEvents('2026-10-12', days, future, '2026-10-20', false)).toEqual({
      ovulation: '2026-11-06',
      fertile: { start: '2026-11-01', end: '2026-11-06' },
    });
  });

  it('is empty under suppression', () => {
    expect(serverFertileEvents('2026-10-12', [day('2026-10-10', { mainPhase: 'fertile' })], [], '2026-10-01', true)).toEqual({
      ovulation: null, fertile: null,
    });
  });
});

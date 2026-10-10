import { describe, expect, it } from 'vitest';

import { PROGRESS_EVERY_MS, SLEEP_MINUTES, SPEEDS, nextInCycle, saveDue } from './use-media';

describe('lesson media helpers', () => {
  it('cycles speed and sleep options', () => {
    expect(nextInCycle(SPEEDS, 1)).toBe(1.25);
    expect(nextInCycle(SPEEDS, 0.75)).toBe(1);
    expect(nextInCycle(SLEEP_MINUTES, 60)).toBe(0);
  });

  it('throttles progress saves', () => {
    expect(saveDue(PROGRESS_EVERY_MS - 1, 0)).toBe(false);
    expect(saveDue(PROGRESS_EVERY_MS, 0)).toBe(true);
  });
});

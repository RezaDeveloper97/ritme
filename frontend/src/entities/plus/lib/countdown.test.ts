import { describe, expect, it } from 'vitest';

import { secondsRemaining, splitSeconds } from './countdown';

describe('trial countdown', () => {
  it('splits seconds like the backend countdown', () => {
    expect(splitSeconds(5 * 86_400 + 14 * 3_600 + 22 * 60 + 59)).toEqual({ days: 5, hours: 14, minutes: 22 });
    expect(splitSeconds(-3)).toEqual({ days: 0, hours: 0, minutes: 0 });
  });

  it('counts down from the server value by elapsed time only', () => {
    expect(secondsRemaining(600, 1_000, 1_000)).toBe(600);
    expect(secondsRemaining(600, 1_000, 61_500)).toBe(540);
    expect(secondsRemaining(600, 1_000, 10_000_000)).toBe(0);
    // A device clock behind the response time never adds time.
    expect(secondsRemaining(600, 5_000, 1_000)).toBe(600);
  });
});

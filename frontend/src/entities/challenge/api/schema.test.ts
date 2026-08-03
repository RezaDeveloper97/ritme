import { describe, expect, it } from 'vitest';

import { todayChallengeSchema } from './schema';

/**
 * Boundary contract for «چالش امروز». The cycle-day fields arrived after the
 * card shipped, so the parser has to keep working against a backend that
 * predates them — a missing range must read as "any day", never as a crash.
 */
describe('todayChallengeSchema — cycle-day targeting', () => {
  const base = {
    id: 7,
    title: 'کشش ملایم',
    description: 'توضیح',
    category: 'exercise',
    is_completed: false,
  };

  it('maps the day the pick was made for and the range it was authored for', () => {
    const challenge = todayChallengeSchema.parse({
      ...base,
      cycle_day: 7,
      cycle_day_from: 6,
      cycle_day_to: 12,
    });

    expect(challenge.cycleDay).toBe(7);
    expect(challenge.cycleDayRange).toEqual({ from: 6, to: 12 });
  });

  it('keeps open-ended ranges open on the side the backend left null', () => {
    expect(
      todayChallengeSchema.parse({ ...base, cycle_day: 30, cycle_day_from: 20, cycle_day_to: null })
        .cycleDayRange,
    ).toEqual({ from: 20, to: null });
  });

  it('reads an untargeted challenge as "any day"', () => {
    const challenge = todayChallengeSchema.parse({ ...base, cycle_day: 3 });

    expect(challenge.cycleDayRange).toEqual({ from: null, to: null });
  });

  it('survives a user with no cycle data and a backend that omits the fields', () => {
    const challenge = todayChallengeSchema.parse(base);

    expect(challenge.cycleDay).toBeNull();
    expect(challenge.cycleDayRange).toEqual({ from: null, to: null });
  });
});

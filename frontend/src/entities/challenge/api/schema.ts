import { z } from 'zod';

import type { ChallengeToggleResult, TodayChallenge } from '../model/types';

/**
 * `data.section.data` of `GET /home/sections/challenge` — validated at the API
 * boundary (CLAUDE.md §10) and mapped onto our camelCase domain shape.
 */
export const todayChallengeSchema = z
  .object({
    id: z.number(),
    title: z.string(),
    description: z.string().nullable().default(null),
    category: z.string().nullable().default(null),
    cycle_day: z.number().nullable().default(null),
    cycle_day_from: z.number().nullable().default(null),
    cycle_day_to: z.number().nullable().default(null),
    is_completed: z.boolean().default(false),
  })
  .transform(
    (c): TodayChallenge => ({
      id: c.id,
      title: c.title,
      description: c.description,
      category: c.category,
      cycleDay: c.cycle_day,
      cycleDayRange: { from: c.cycle_day_from, to: c.cycle_day_to },
      isCompleted: c.is_completed,
    }),
  );

/**
 * The section envelope. The backend omits the section entirely when no
 * challenge is available, so a missing/null section maps to `null`.
 */
export const challengeSectionSchema = z
  .object({
    section: z
      .object({ data: todayChallengeSchema })
      .nullable()
      .default(null),
  })
  .transform((e): TodayChallenge | null => e.section?.data ?? null);

/** `POST /home/challenges/{id}/toggle`. */
export const challengeToggleSchema = z
  .object({
    challenge_id: z.number(),
    is_completed: z.boolean(),
  })
  .transform(
    (r): ChallengeToggleResult => ({
      challengeId: r.challenge_id,
      isCompleted: r.is_completed,
    }),
  );

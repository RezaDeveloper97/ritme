import { z } from 'zod';

import type {
  ChallengeDay,
  ChallengeDifficulty,
  ChallengeToggleResult,
  TodayChallenge,
} from '../model/types';

const difficultySchema = z
  .enum(['easy', 'medium', 'hard'])
  .nullable()
  .catch(null)
  .default(null);

const daySchema = z
  .object({
    date: z.string(),
    is_completed: z.boolean().default(false),
    is_today: z.boolean().default(false),
  })
  .transform(
    (d): ChallengeDay => ({
      date: d.date,
      isCompleted: d.is_completed,
      isToday: d.is_today,
    }),
  );

/** Fields shared by the section payload and the toggle response. */
const progressShape = {
  streak: z.number().default(0),
  longest_streak: z.number().default(0),
  week_days: z.array(daySchema).default([]),
  status_message: z.string().nullable().default(null),
};

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
    difficulty: difficultySchema,
    is_completed: z.boolean().default(false),
    ...progressShape,
  })
  .transform(
    (c): TodayChallenge => ({
      id: c.id,
      title: c.title,
      description: c.description,
      category: c.category,
      difficulty: c.difficulty as ChallengeDifficulty | null,
      isCompleted: c.is_completed,
      streak: c.streak,
      longestStreak: c.longest_streak,
      weekDays: c.week_days,
      statusMessage: c.status_message,
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
    ...progressShape,
  })
  .transform(
    (r): ChallengeToggleResult => ({
      challengeId: r.challenge_id,
      isCompleted: r.is_completed,
      streak: r.streak,
      longestStreak: r.longest_streak,
      weekDays: r.week_days,
      statusMessage: r.status_message,
    }),
  );

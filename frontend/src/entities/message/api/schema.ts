import { z } from 'zod';

import type { AppMode, DailyMessage, UserMode } from '../model/types';

const appModeSchema = z
  .string()
  .transform((m): AppMode => (m === 'pregnancy' ? 'pregnancy' : 'cycle'));

/** Boundary parser for GET /messages/mode (CLAUDE.md §10). */
export const userModeSchema = z
  .object({
    mode: appModeSchema,
    mode_label: z.string().default(''),
    user_goal: z.string().default(''),
    is_ttc: z.boolean().default(false),
    is_premium: z.boolean().default(false),
  })
  .transform(
    (m): UserMode => ({
      mode: m.mode,
      modeLabel: m.mode_label,
      userGoal: m.user_goal,
      isTtc: m.is_ttc,
      isPremium: m.is_premium,
    }),
  );

const primaryMessageSchema = z
  .object({
    short_message: z.string().default(''),
    long_message: z.string().default(''),
    action_suggestion: z.string().default(''),
    dos: z.array(z.string()).default([]),
    donts: z.array(z.string()).default([]),
  })
  .default({});

const contextInfoSchema = z
  .object({
    phase: z.string().nullable().default(null),
    phase_label: z.string().nullable().default(null),
    cycle_day: z.number().nullable().default(null),
  })
  .default({});

/** Boundary parser for GET /messages/daily (CLAUDE.md §10). */
export const dailyMessageSchema = z
  .object({
    mode: appModeSchema,
    date: z.string().default(''),
    context_info: contextInfoSchema,
    primary_message: primaryMessageSchema,
  })
  .transform(
    (d): DailyMessage => ({
      mode: d.mode,
      date: d.date,
      phase: d.context_info.phase,
      phaseLabel: d.context_info.phase_label,
      cycleDay: d.context_info.cycle_day,
      primary: {
        shortMessage: d.primary_message.short_message,
        longMessage: d.primary_message.long_message,
        actionSuggestion: d.primary_message.action_suggestion,
        dos: d.primary_message.dos,
        donts: d.primary_message.donts,
      },
    }),
  );

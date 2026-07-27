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

/**
 * PHP serializes an empty associative array as `[]`, so any object-shaped field
 * of the message payload can arrive as an empty list. Normalize it to `{}` so a
 * message-less day parses to empty strings instead of throwing at the boundary.
 */
const objectish = <T extends z.ZodTypeAny>(shape: T) =>
  z.preprocess(v => (Array.isArray(v) && v.length === 0 ? {} : v), shape);

/** Same idea in reverse: an empty object stands in for an empty list. */
const listish = <T extends z.ZodTypeAny>(shape: T) =>
  z.preprocess(
    v => (v && !Array.isArray(v) && typeof v === 'object' && Object.keys(v).length === 0 ? [] : v),
    shape,
  );

const stringList = listish(z.array(z.string()).default([])).pipe(z.array(z.string()));

const primaryMessageSchema = objectish(
  z
    .object({
      short_message: z.string().default(''),
      long_message: z.string().default(''),
      action_suggestion: z.string().default(''),
      dos: stringList.default([]),
      donts: stringList.default([]),
    })
    .default({}),
);

const contextInfoSchema = objectish(
  z
    .object({
      phase: z.string().nullable().default(null),
      phase_label: z.string().nullable().default(null),
      cycle_day: z.number().nullable().default(null),
    })
    .default({}),
);

const correlationSchema = z.object({
  type: z.string().default(''),
  insight_message: z.string().default(''),
  action: z.string().default(''),
  is_premium_only: z.boolean().default(false),
});

const patternSchema = z.object({
  pattern_type: z.string().default(''),
  alert_level: z.string().default('info'),
  message: z.string().default(''),
  recommendation: z.string().default(''),
});

/**
 * The supplement modules answer with a different shape per mode (cycle vs.
 * pregnancy vs. TTC), so only the fields the UI actually renders are picked;
 * everything else is ignored rather than rejected.
 */
const supplementsSchema = objectish(
  z
    .object({
      nutrition: objectish(
        z
          .object({ focus: z.string().default(''), tip: z.string().default('') })
          .partial()
          .passthrough()
          .default({}),
      ),
      sleep: objectish(
        z
          .object({ quality_focus: z.string().default(''), tips: stringList.default([]) })
          .partial()
          .passthrough()
          .default({}),
      ),
      exercise: objectish(
        z.object({ tip: z.string().default('') }).partial().passthrough().default({}),
      ),
    })
    .partial()
    .default({}),
);

/** Boundary parser for GET /messages/daily (CLAUDE.md §10). */
export const dailyMessageSchema = z
  .object({
    mode: appModeSchema,
    date: z.string().default(''),
    context_info: contextInfoSchema,
    primary_message: primaryMessageSchema,
    correlations: listish(z.array(correlationSchema).default([])).pipe(z.array(correlationSchema)),
    patterns: listish(z.array(patternSchema).default([])).pipe(z.array(patternSchema)),
    supplements: supplementsSchema,
    tips: stringList.default([]),
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
      correlations: d.correlations.map(c => ({
        type: c.type,
        insightMessage: c.insight_message,
        action: c.action,
        isPremiumOnly: c.is_premium_only,
      })),
      patterns: d.patterns.map(p => ({
        patternType: p.pattern_type,
        alertLevel: p.alert_level,
        message: p.message,
        recommendation: p.recommendation,
      })),
      supplements: {
        nutritionFocus: d.supplements.nutrition?.focus ?? '',
        nutritionTip: d.supplements.nutrition?.tip ?? '',
        sleepFocus: d.supplements.sleep?.quality_focus ?? '',
        sleepTips: d.supplements.sleep?.tips ?? [],
        exerciseTip: d.supplements.exercise?.tip ?? '',
      },
      tips: d.tips,
    }),
  );

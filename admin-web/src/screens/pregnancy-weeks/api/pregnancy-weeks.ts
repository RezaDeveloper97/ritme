'use client';

import { z } from 'zod';

import { createResource } from '@/shared/api';

/** GET /pregnancy-weeks: every week 1…40 (+ stored weeks above) with its row id or null. */
const mapSchema = z.object({
  items: z.array(z.object({ week: z.number(), id: z.number().nullable() })),
  fields: z.array(z.string()),
});
export type WeekMap = z.infer<typeof mapSchema>;

/** `PregnancyWeek`: id, week_number and the ten translatable modules named by `fields`. */
const weekSchema = z.object({ id: z.number(), week_number: z.number() }).catchall(z.unknown());
export type PregnancyWeek = z.infer<typeof weekSchema>;

export const pregnancyWeeksApi = createResource({
  key: 'pregnancy-weeks',
  path: '/pregnancy-weeks',
  list: mapSchema,
  detail: z.object({ pregnancy_week: weekSchema }),
});

'use client';

import { z } from 'zod';

import { createResource, optionSchema, pagedSchema } from '@/shared/api';

/** GET /users row and GET /users/:id (admin-api.md §6). */
export const userRowSchema = z.object({
  id: z.number(),
  name: z.string().nullable(),
  mobile: z.string().nullable(),
  email: z.string().nullable(),
  is_blocked: z.boolean(),
  blocked_at: z.string().nullable(),
  subscription_type: z.string(),
  user_goal: z.string(),
  created_at: z.string().nullable(),
});
export type UserRow = z.infer<typeof userRowSchema>;

const userDetailSchema = z.object({
  user: userRowSchema.extend({
    mobile_verified_at: z.string().nullable().optional(),
    updated_at: z.string().nullable().optional(),
  }),
  profile: z
    .object({
      birthday: z.string().nullable(),
      last_period_start: z.string().nullable(),
      cycle_duration: z.number().nullable(),
      period_duration: z.number().nullable(),
      subscription_type: z.string().nullable(),
      user_goal: z.string().nullable(),
      pregnancy_intention: z.string().nullable().optional(),
    })
    .nullable(),
  stats: z.object({ health_logs: z.number(), reminders: z.number(), notifications: z.number() }),
  options: z.object({ subscription_types: z.array(optionSchema), user_goals: z.array(optionSchema) }),
});
export type UserDetail = z.infer<typeof userDetailSchema>;

export const usersApi = createResource({
  key: 'users',
  path: '/users',
  list: pagedSchema(userRowSchema, {
    filters: z.object({ q: z.string(), status: z.string() }).partial().optional(),
  }),
  detail: userDetailSchema,
  alsoInvalidate: [['dashboard']],
});

export interface UserUpdate {
  name: string | null;
  subscription_type: string;
  user_goal: string;
}

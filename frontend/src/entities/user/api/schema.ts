import { z } from 'zod';

import type { AuthUser, UserProfile } from '../model/types';

/**
 * Validate the API `user` object at the boundary (CLAUDE.md §10) and map its
 * snake_case fields onto our camelCase domain type. Extra fields the endpoint
 * sends (`created_at`, …) are ignored — we keep only account identity.
 */
export const authUserSchema = z
  .object({
    id: z.number(),
    name: z.string().nullable().default(null),
    mobile: z.string(),
    mobile_verified_at: z.string().nullable().default(null),
  })
  .transform(
    (u): AuthUser => ({
      id: u.id,
      name: u.name,
      mobile: u.mobile,
      mobileVerifiedAt: u.mobile_verified_at,
    }),
  );

/**
 * Validate the `GET /profile` payload at the boundary (CLAUDE.md §10) and map
 * its snake_case fields onto the camelCase {@link UserProfile}. The `profile`
 * object — and each field inside it — is nullable: new accounts have no health
 * profile yet, so we default to `null` rather than throwing. This is sensitive
 * data (§11); it is validated here but never logged.
 */
export const userProfileSchema = z
  .object({
    user: z.object({
      id: z.number(),
      name: z.string().nullable().default(null),
      mobile: z.string(),
    }),
    profile: z
      .object({
        birthday: z.string().nullable().default(null),
        weight: z.number().nullable().default(null),
        height: z.number().nullable().default(null),
        period_duration: z.number().nullable().default(null),
        cycle_duration: z.number().nullable().default(null),
        last_period_start: z.string().nullable().default(null),
      })
      .nullable()
      .default(null),
  })
  .transform(
    (r): UserProfile => ({
      id: r.user.id,
      name: r.user.name,
      mobile: r.user.mobile,
      health: r.profile
        ? {
            birthday: r.profile.birthday,
            weight: r.profile.weight,
            height: r.profile.height,
            periodDuration: r.profile.period_duration,
            cycleDuration: r.profile.cycle_duration,
            lastPeriodStart: r.profile.last_period_start,
          }
        : null,
    }),
  );

import { z } from 'zod';

/**
 * Validate environment configuration once, at the boundary (CLAUDE.md §10).
 * Only `NEXT_PUBLIC_*` values are referenced by literal name so Next can inline
 * them for both server and client.
 *
 * Privacy: never place secrets or any health data in public env — see §11.
 */
const envSchema = z.object({
  apiBaseUrl: z.string().url().default('https://api.ritme.app/api/v1'),
});

export const env = envSchema.parse({
  apiBaseUrl: process.env.NEXT_PUBLIC_API_BASE_URL,
});

export type Env = typeof env;

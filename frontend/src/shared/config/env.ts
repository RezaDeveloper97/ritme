import { z } from 'zod';

import { resolveNeshanKey } from './map';

/**
 * Validate environment configuration once, at the boundary (CLAUDE.md §10).
 * Only `NEXT_PUBLIC_*` values are referenced by literal name so Next can inline
 * them for both server and client.
 *
 * Privacy: never place secrets or any health data in public env — see §11.
 * `NEXT_PUBLIC_NESHAN_KEY` is a browser map key by nature (Neshan's web SDK
 * needs it client-side); it is domain-restricted in the Neshan panel and set on
 * the server at build time only — never in the repo or an `.env` file
 * (roadmap/DECISIONS.md #18, docs/canvas-build/map.md).
 */
const envSchema = z.object({
  apiBaseUrl: z.string().url().default('https://api.ritme.app/api/v1'),
  neshanKey: z.string().optional().transform(resolveNeshanKey),
});

export const env = envSchema.parse({
  apiBaseUrl: process.env.NEXT_PUBLIC_API_BASE_URL,
  neshanKey: process.env.NEXT_PUBLIC_NESHAN_KEY,
});

export type Env = typeof env;

import { z } from 'zod';

/**
 * Validate environment configuration once, at the boundary (CLAUDE.md §10).
 * Only `NEXT_PUBLIC_*` values are referenced by literal name so Next can inline
 * them for both server and client.
 *
 * Privacy: never place secrets or any health data in public env — see §11.
 */
const envSchema = z.object({
  apiBaseUrl: z.string().url().default('https://ritmeapp.ir/api/v1'),
  // Test mode asks the backend to skip real SMS and accept `1111` as the OTP
  // (see the Auth group in the OpenAPI spec). Defaults to ON while we are not in
  // production so the login flow is exercisable without a live SMS gateway; set
  // NEXT_PUBLIC_OTP_TEST_MODE=false in production to send real OTPs.
  otpTestMode: z
    .string()
    .optional()
    .transform((v) => v !== 'false' && v !== '0'),
});

export const env = envSchema.parse({
  apiBaseUrl: process.env.NEXT_PUBLIC_API_BASE_URL,
  otpTestMode: process.env.NEXT_PUBLIC_OTP_TEST_MODE,
});

export type Env = typeof env;

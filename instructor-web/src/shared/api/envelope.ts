import { z } from 'zod';

/** `{success, message?, data}` — the backend-go httpx envelope. */
export function envelopeSchema<T extends z.ZodTypeAny>(data: T) {
  return z.object({
    success: z.literal(true),
    message: z.string().optional(),
    data,
  });
}

/**
 * Failure body. `retry_after` sits top-level on throttle responses and under
 * `data` on POST /auth/send-otp's 429 (Laravel parity).
 */
export const errorBodySchema = z.object({
  success: z.literal(false).optional(),
  message: z.string().optional(),
  error_code: z.string().optional(),
  errors: z.record(z.string(), z.array(z.string())).optional(),
  retry_after: z.number().optional(),
  data: z.object({ retry_after: z.number().optional() }).passthrough().nullable().optional(),
});

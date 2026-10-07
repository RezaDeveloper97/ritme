import { z } from 'zod';

import { api } from '@/shared/api';
import { setAuthToken } from '@/shared/session';

/** POST /api/v1/auth/send-otp — the same endpoint as the user app. */
export function sendOtp(mobile: string): Promise<{ expires_in: number }> {
  return api.post('/auth/send-otp', { mobile }, { api: 'public', schema: z.object({ expires_in: z.number() }) });
}

const verifySchema = z.object({ access_token: z.string().min(1) }).passthrough();

/**
 * POST /api/v1/auth/verify-otp — on success the bearer token is stored. (A
 * number with no Ritme account gets one here, exactly as in the app.)
 */
export async function verifyOtp(mobile: string, code: string): Promise<void> {
  const data = await api.post('/auth/verify-otp', { mobile, code }, { api: 'public', schema: verifySchema });
  setAuthToken(data.access_token);
}

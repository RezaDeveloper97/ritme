import { apiClient } from './apiClient';
import { type ApiEnvelope, getApiErrorCode, getApiErrorStatus } from './envelope';

/**
 * Step-up (T-M12-06, D-213): the full data export and the account deletion
 * need a fresh SMS code on top of the session. The code goes to the signed-in
 * user's own number (the API takes it from the token, never from us), and a
 * matching code returns a short-lived, single-use token that the protected
 * call sends as `X-Step-Up`. The token lives only in the caller's memory for
 * that one call: never stored, logged or put in a URL.
 */

export const STEP_UP_HEADER = 'X-Step-Up';

/** The 403 `error_code` of a protected call without a valid step-up token. */
export const STEP_UP_REQUIRED = 'step_up_required';

/** POST /auth/send-otp `purpose: step_up` — texts a code to the user's own number. */
export async function requestStepUpCode(): Promise<void> {
  await apiClient.post('/auth/send-otp', { purpose: 'step_up' });
}

/** POST /auth/verify-otp `purpose: step_up` — exchanges the code for a single-use step-up token. */
export async function verifyStepUpCode(code: string): Promise<string> {
  const { data } = await apiClient.post<ApiEnvelope<{ step_up_token?: unknown }>>('/auth/verify-otp', {
    purpose: 'step_up',
    code,
  });
  const token = data.data?.step_up_token;
  if (typeof token !== 'string' || !token) throw new Error('step-up: no token');
  return token;
}

/** Request headers that carry a step-up token. */
export function stepUpHeaders(token: string): Record<string, string> {
  return { [STEP_UP_HEADER]: token };
}

/** The API refused the call for a missing / expired / spent step-up (ask for a new code). */
export function isStepUpRequired(error: unknown): boolean {
  return getApiErrorStatus(error) === 403 && getApiErrorCode(error) === STEP_UP_REQUIRED;
}

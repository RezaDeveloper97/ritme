import { getApiErrorStatus } from '@/shared/api';

export type StepUpErrorKey = 'wrongCode' | 'expired' | 'wait' | 'tooMany' | 'error';

/**
 * Maps a step-up send / verify failure to its `profile.stepUp.*` copy key. The
 * OTP endpoints answer in English (Laravel parity), so the sheet shows its own
 * localized line instead of the server message.
 */
export function stepUpErrorKey(error: unknown, stage: 'send' | 'verify'): StepUpErrorKey {
  const status = getApiErrorStatus(error);
  if (stage === 'send') return status === 429 ? 'wait' : 'error';
  if (status === 422) return 'wrongCode';
  if (status === 400) return 'expired';
  if (status === 429) return 'tooMany';
  return 'error';
}

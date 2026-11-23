'use client';

import { useAiGate } from '@/features/ai-gate';

/**
 * Whether the voice entry should wear «پلاس»: the user has no running subscription. Unknown (loading, a failed
 * read) is not locked — the gate fails open and the server decides. Lives apart from the recorder slot so a
 * screen can ask without pulling the recorder (T-M12-27b).
 */
export function useVoiceLocked(): boolean {
  return useAiGate('voice_log').status === 'no_subscription';
}

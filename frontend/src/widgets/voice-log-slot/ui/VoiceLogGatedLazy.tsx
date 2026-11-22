'use client';

import dynamic from 'next/dynamic';
import { type VoiceLogSlot } from '@/features/log-day';
import { Skeleton } from '@/shared/ui';

/**
 * The recorder + AI-gate wiring behind `next/dynamic`, loaded when the voice tab first renders. Exported by the
 * barrel instead of the real component so a screen that imports `useVoiceLocked` does not pull the recorder (T-M12-27b).
 */
export const VoiceLogGated = dynamic(() => import('./VoiceLogGated').then((m) => m.VoiceLogGated as VoiceLogSlot), {
  loading: () => <Skeleton shape="card" />,
});

'use client';

import { useMemo } from 'react';

import { useLogTaxonomy } from '@/entities/health-log';
import { useAiGate } from '@/features/ai-gate';
import { logTargetsOf } from '@/features/log-day';
import { logSheetManifest, VoiceLogPanel, type VoiceLogPanelProps } from '@/features/voice-log';

type SlotProps = Omit<VoiceLogPanelProps, 'ai' | 'manifest'> & { mode: string | null };

/**
 * The log sheet's voice slot, wired to the AI gate and to the sheet's autopilot vocabulary: the recorder
 * (features/voice-log), the gate (features/ai-gate) and the sheet's target builder (features/log-day) are
 * sibling features, so this widget composes them. Screens pass it as `VoiceLog={…}`.
 *
 * The manifest is the log sheet's fields only, in the user's language and mode — what the model is told
 * exists. It holds nothing she logged (§11); the request that carries it goes through the gate.
 */
export function VoiceLogGated({ mode, ...props }: SlotProps) {
  const ai = useAiGate('voice_log');
  const taxonomy = useLogTaxonomy(mode ?? undefined);
  const manifest = useMemo(() => (taxonomy.data ? logSheetManifest(logTargetsOf(taxonomy.data)) : null), [taxonomy.data]);
  return <VoiceLogPanel {...props} ai={ai} manifest={manifest} />;
}

/**
 * Whether the voice entry should wear «پلاس»: the user has no running subscription. Unknown (loading, a failed
 * read) is not locked — the gate fails open and the server decides.
 */
export function useVoiceLocked(): boolean {
  return useAiGate('voice_log').status === 'no_subscription';
}

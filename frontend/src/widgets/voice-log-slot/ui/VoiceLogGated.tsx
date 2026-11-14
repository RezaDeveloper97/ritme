'use client';

import { useAiGate } from '@/features/ai-gate';
import { VoiceLogPanel, type VoiceLogPanelProps } from '@/features/voice-log';

/**
 * The log sheet's voice slot, wired to the AI gate: the recorder (features/voice-log) and the gate
 * (features/ai-gate) are sibling features, so this widget composes them. Screens pass it as `VoiceLog={…}`.
 */
export function VoiceLogGated(props: Omit<VoiceLogPanelProps, 'ai'>) {
  const ai = useAiGate('voice_log');
  return <VoiceLogPanel {...props} ai={ai} />;
}

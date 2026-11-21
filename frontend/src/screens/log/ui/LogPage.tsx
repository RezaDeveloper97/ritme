'use client';

import dynamic from 'next/dynamic';
import { useSearchParams } from 'next/navigation';

import { LogDay, type VoiceLogSlot } from '@/features/log-day';
import { useAiGate } from '@/features/ai-gate';
import { useRouter } from '@/shared/i18n';
import { Skeleton, SkyLayer } from '@/shared/ui';
import { BodyMap } from '@/widgets/body-map';
import '@/shared/ui/styles/lday.css';

// T-M12-28: the recorder + AI gate wiring load when the voice tab first renders (`useVoiceLocked` shares the slot's module, so ask the gate directly).
const VoiceLogGated = dynamic(() => import('@/widgets/voice-log-slot').then((m) => m.VoiceLogGated as VoiceLogSlot), { loading: () => <Skeleton shape="card" /> });

/**
 * `/log[?date=Y-m-d]` — the log sheet v2 as a full screen (B-N3-03). Kept so existing links (week summary,
 * menopause home tiles, bookmarks) keep working; new entry points open the `?sheet=log` panel instead.
 */
export function LogPage() {
  const router = useRouter();
  const date = useSearchParams().get('date');
  const voiceLocked = useAiGate('voice_log').status === 'no_subscription';
  return (
    <div className="view lday-page">
      <SkyLayer />
      <div className="scroll lday-page-scroll">
        <LogDay variant="page" initialDate={date} BodyMap={BodyMap} VoiceLog={VoiceLogGated} voiceLocked={voiceLocked} onClose={() => router.push('/home')} />
      </div>
    </div>
  );
}

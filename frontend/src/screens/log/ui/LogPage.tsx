'use client';

import { useSearchParams } from 'next/navigation';

import { LogDay } from '@/features/log-day';
import { useVoiceLocked, VoiceLogGated } from '@/widgets/voice-log-slot';
import { useRouter } from '@/shared/i18n';
import { SkyLayer } from '@/shared/ui';
import { BodyMap } from '@/widgets/body-map';
import '@/shared/ui/styles/lday.css';

/**
 * `/log[?date=Y-m-d]` — the log sheet v2 as a full screen (B-N3-03). Kept so existing links (week summary,
 * menopause home tiles, bookmarks) keep working; new entry points open the `?sheet=log` panel instead.
 */
export function LogPage() {
  const router = useRouter();
  const date = useSearchParams().get('date');
  const voiceLocked = useVoiceLocked();
  return (
    <div className="view lday-page">
      <SkyLayer />
      <div className="scroll lday-page-scroll">
        <LogDay variant="page" initialDate={date} BodyMap={BodyMap} VoiceLog={VoiceLogGated} voiceLocked={voiceLocked} onClose={() => router.push('/home')} />
      </div>
    </div>
  );
}

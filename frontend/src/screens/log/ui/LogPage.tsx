'use client';

import { useSearchParams } from 'next/navigation';

import { LogDay } from '@/features/log-day';
import { VoiceLogPanel } from '@/features/voice-log';
import { useRouter } from '@/shared/i18n';
import { SkyLayer } from '@/shared/ui';
import { BodyMap } from '@/widgets/body-map';

/**
 * `/log[?date=Y-m-d]` — the log sheet v2 as a full screen (B-N3-03). Kept so existing links (week summary,
 * menopause home tiles, bookmarks) keep working; new entry points open the `?sheet=log` panel instead.
 */
export function LogPage() {
  const router = useRouter();
  const date = useSearchParams().get('date');
  return (
    <div className="view lday-page">
      <SkyLayer />
      <div className="scroll lday-page-scroll">
        <LogDay variant="page" initialDate={date} BodyMap={BodyMap} VoiceLog={VoiceLogPanel} onClose={() => router.push('/home')} />
      </div>
    </div>
  );
}

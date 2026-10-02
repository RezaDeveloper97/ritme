'use client';

import { LogDay } from '@/features/log-day';
import { VoiceLogPanel } from '@/features/voice-log';
import { useRouter } from '@/shared/i18n';
import { SkyLayer } from '@/shared/ui';
import { BodyMap } from '@/widgets/body-map';

/**
 * `/pregnancy/log[?date=]` — the log sheet v2 on its pregnancy preset as a full page (B-N3-06,
 * nbl_/nbd_Log_Sheet_Preg): kicks, contractions, symptom groups, weight, BP, sleep, meds. Saves through
 * `/logs/days`. The pregnancy-endpoint forms stay on their tabs (`?tab=day|symptoms|weekly|movement`).
 */
export function PregnancySheetPage({ date }: { date?: string }) {
  const router = useRouter();
  const close = () => router.push('/pregnancy');
  return (
    <div className="view lday-page">
      <SkyLayer />
      <div className="scroll lday-page-scroll">
        <LogDay
          variant="page"
          mode="pregnancy"
          initialDate={date}
          BodyMap={BodyMap}
          VoiceLog={VoiceLogPanel}
          onClose={close}
          onSaved={close}
        />
      </div>
    </div>
  );
}

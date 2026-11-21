'use client';

import { LogDay, LogDaySheetTitle, type VoiceLogSlot } from '@/features/log-day';
import dynamic from 'next/dynamic';

import { useAiGate } from '@/features/ai-gate';
import { closeSheet, type SheetContentProps } from '@/shared/sheet';
import { Skeleton } from '@/shared/ui';
import { BodyMap } from '@/widgets/body-map';
import { LogSheet as CompanionLogSheet, LogSheetTitle as CompanionLogSheetTitle, useNavMode } from '@/widgets/bottom-nav';
import { LogLater } from '@/widgets/log-later';

// T-M12-28: the recorder, the AI gate wiring and the voice manifest load when the voice tab first renders, not with the sheet.
// (`useVoiceLocked` lives in the same module as the slot, so the sheet asks the gate itself.)
const VoiceLogGated = dynamic(() => import('@/widgets/voice-log-slot').then((m) => m.VoiceLogGated as VoiceLogSlot), { loading: () => <Skeleton shape="card" /> });

/**
 * `?sheet=log` — the nav FAB, home «ثبت امروز» and calendar «ثبت جزئیات» (B-N3-03). `arg` is the day to
 * log (`YYYY-MM-DD`, optionally `:pregnancy_test` or `:<category>` to open on that field / section). Companions log nothing of their own, so they keep the
 * interim reminder / checkup shortcuts.
 */
export function LogSheet({ arg }: SheetContentProps) {
  // `YYYY-MM-DD`, optionally `:<focus>` — `pregnancy_test` (the overdue banner's deep link into the
  // measurement panel) or a category code (home «ثبت امروز» rows open that section).
  const [date, focusKey] = (arg ?? '').split(':');
  const focus = focusKey && /^[a-z_]{1,40}$/.test(focusKey) ? focusKey : undefined;
  const nav = useNavMode();
  const voiceLocked = useAiGate('voice_log').status === 'no_subscription';
  if (nav.mode === 'companion') return <CompanionLogSheet />;
  return (
    <LogDay
      variant="sheet"
      initialDate={date || undefined}
      focus={focus}
      BodyMap={BodyMap}
      VoiceLog={VoiceLogGated} voiceLocked={voiceLocked}
      onSaved={closeSheet}
      // CB-NAV-03 (nbd_Nav_Plus): «برای بعد» + «ثبت برای <child>» (the postpartum preset already has the baby rows).
      extra={<LogLater childRow={nav.mode !== 'postpartum'} />}
    />
  );
}

export function LogSheetTitle() {
  const nav = useNavMode();
  if (nav.mode === 'companion') return <CompanionLogSheetTitle />;
  return <LogDaySheetTitle />;
}

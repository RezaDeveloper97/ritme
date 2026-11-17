'use client';

import { LogDay, LogDaySheetTitle } from '@/features/log-day';
import { useVoiceLocked, VoiceLogGated } from '@/widgets/voice-log-slot';
import { closeSheet, type SheetContentProps } from '@/shared/sheet';
import { BodyMap } from '@/widgets/body-map';
import { LogSheet as CompanionLogSheet, LogSheetTitle as CompanionLogSheetTitle, useNavMode } from '@/widgets/bottom-nav';
import { LogLater } from '@/widgets/log-later';

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
  const voiceLocked = useVoiceLocked();
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

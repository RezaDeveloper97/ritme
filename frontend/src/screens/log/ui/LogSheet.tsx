'use client';

import { LogDay, LogDaySheetTitle } from '@/features/log-day';
import { VoiceLogPanel } from '@/features/voice-log';
import { closeSheet, type SheetContentProps } from '@/shared/sheet';
import { BodyMap } from '@/widgets/body-map';
import { LogSheet as CompanionLogSheet, LogSheetTitle as CompanionLogSheetTitle, useNavMode } from '@/widgets/bottom-nav';
import { LogLater } from '@/widgets/log-later';

/**
 * `?sheet=log` — the nav FAB, home «ثبت امروز» and calendar «ثبت جزئیات» (B-N3-03). `arg` is the day to
 * log (`YYYY-MM-DD`, optionally `:pregnancy_test` to open on the test field). Companions log nothing of their own, so they keep the
 * interim reminder / checkup shortcuts.
 */
export function LogSheet({ arg }: SheetContentProps) {
  // `YYYY-MM-DD` or `YYYY-MM-DD:pregnancy_test` (the overdue banner's deep link into the measurement panel).
  const [date, focusKey] = (arg ?? '').split(':');
  const focus = focusKey === 'pregnancy_test' ? focusKey : undefined;
  const nav = useNavMode();
  if (nav.mode === 'companion') return <CompanionLogSheet />;
  return (
    <LogDay
      variant="sheet"
      initialDate={date || undefined}
      focus={focus}
      BodyMap={BodyMap}
      VoiceLog={VoiceLogPanel}
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

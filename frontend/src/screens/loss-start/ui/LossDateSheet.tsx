'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useState } from 'react';

import type { Locale } from '@/shared/i18n';
import { type DateParts, diffInDays, fromApiDate, partsToDate, toApiDate, toParts, today } from '@/shared/lib/date';
import { AppSheet } from '@/shared/sheet';
import { CalendarPicker, PrimaryButton, SecondaryButton } from '@/shared/ui';

/** How far back the approximate date may go (CB-LOSS-01 `MaxPastDays`). */
const MAX_PAST_DAYS = 365;

/**
 * «تاریخ تقریبی» — a day in the locale's calendar (Jalali in fa), never in the
 * future and within the past year. «بدون تاریخ» clears it: the date is optional.
 */
export function LossDateSheet({
  value,
  onClose,
  onPick,
}: {
  value: string | null;
  onClose: () => void;
  onPick: (date: string | null) => void;
}) {
  const t = useTranslations('loss.start');
  const tc = useTranslations('loss.common');
  const locale = useLocale() as Locale;
  const [parts, setParts] = useState<DateParts | null>(value ? toParts(fromApiDate(value), locale) : null);

  const picked = parts ? partsToDate(parts, locale) : null;
  const ago = picked ? diffInDays(today(), picked) : 0;
  const error = picked ? (ago < 0 ? t('dateFuture') : ago > MAX_PAST_DAYS ? t('dateTooOld') : null) : null;

  return (
    <AppSheet
      open
      onClose={onClose}
      size="half"
      title={t('dateSheet')}
      footer={
        <div className="lst-sheet-btns">
          <SecondaryButton block={false} onClick={() => onPick(null)}>
            {t('dateClear')}
          </SecondaryButton>
          <PrimaryButton block={false} disabled={!picked || !!error} onClick={() => picked && onPick(toApiDate(picked))}>
            {tc('save')}
          </PrimaryButton>
        </div>
      }
    >
      <CalendarPicker value={parts} onSelect={setParts} />
      {error ? (
        <p className="lst-error" role="alert">
          {error}
        </p>
      ) : null}
    </AppSheet>
  );
}

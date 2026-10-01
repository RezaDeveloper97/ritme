'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useState } from 'react';

import type { Locale } from '@/shared/i18n';
import { type DateParts, fromApiDate, partsToDate, toApiDate, toParts } from '@/shared/lib/date';
import { AppSheet } from '@/shared/sheet';
import { CalendarPicker, PrimaryButton, SecondaryButton } from '@/shared/ui';

/**
 * A date in the locale's calendar (Jalali in fa) → `Y-m-d`, for the implant's
 * removal / replacement date. Mounted only while open; `busy` while saving.
 */
export function DateSheet({
  title,
  value,
  busy,
  error,
  onClose,
  onPick,
}: {
  title: string;
  value: string | null;
  busy: boolean;
  error: string | null;
  onClose: () => void;
  onPick: (apiDate: string) => void;
}) {
  const t = useTranslations('contraception.setup');
  const locale = useLocale() as Locale;
  const [parts, setParts] = useState<DateParts | null>(value ? toParts(fromApiDate(value), locale) : null);
  return (
    <AppSheet
      open
      onClose={onClose}
      size="half"
      title={title}
      footer={
        <div className="ctr-sheet-btns">
          <SecondaryButton block={false} onClick={onClose} disabled={busy}>
            {t('cancel')}
          </SecondaryButton>
          <PrimaryButton
            block={false}
            loading={busy}
            disabled={!parts}
            onClick={() => parts && onPick(toApiDate(partsToDate(parts, locale)))}
          >
            {t('save')}
          </PrimaryButton>
        </div>
      }
    >
      <CalendarPicker value={parts} onSelect={setParts} />
      {error ? (
        <p className="ctr-error is-inline" role="alert">
          {error}
        </p>
      ) : null}
    </AppSheet>
  );
}

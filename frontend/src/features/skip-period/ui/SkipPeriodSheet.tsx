'use client';

import { useTranslations } from 'next-intl';

import { AppSheet } from '@/shared/sheet';
import { toApiDate, today } from '@/shared/lib/date';

import { skipErrorKind, useSkipPeriod, type SkipErrorKind } from '../api/mutations';
import { useSkipStore } from '../model/store';
import '@/shared/ui/styles/btn.css';
import '@/shared/ui/styles/ch.css';

const ERROR_KEY = {
  not_expected_yet: 'errors.notExpectedYet',
  period_started: 'errors.periodStarted',
  prediction_mismatch: 'errors.predictionMismatch',
  network: 'errors.network',
  other: 'errors.generic',
} as const satisfies Record<SkipErrorKind, string>;

export interface SkipPeriodSheetProps {
  open: boolean;
  onClose: () => void;
  /** The predicted start the user says did not come (Y-m-d). */
  expectedStart: string;
}

/** Confirm sheet for «پریود نداشتم»: one short paragraph, then a single decision (§4.1: a decision is a sheet). */
export function SkipPeriodSheet({ open, onClose, expectedStart }: SkipPeriodSheetProps) {
  const t = useTranslations('home.skip');
  const skip = useSkipPeriod();
  const record = useSkipStore((s) => s.record);

  const close = () => {
    skip.reset();
    onClose();
  };
  const confirm = () =>
    skip.mutate(expectedStart, {
      onSuccess: (expected) => {
        record(expected, toApiDate(today()));
        close();
      },
    });

  return (
    <AppSheet open={open} onClose={close} size="half" title={t('sheet.title')}>
      <div className="flex flex-col gap-4 px-5 pb-6">
        <p className="m-0 text-[14px] leading-7 text-(--text-2)">{t('sheet.body')}</p>
        {skip.isError && (
          <p role="alert" className="m-0 text-[13px] font-semibold text-(--danger)">
            {t(ERROR_KEY[skipErrorKind(skip.error)])}
          </p>
        )}
        <div className="flex flex-col gap-2">
          <button type="button" className="btn btn-primary" onClick={confirm} disabled={skip.isPending}>
            {t('sheet.confirm')}
          </button>
          <button type="button" className="ch-link self-center" onClick={close}>
            {t('sheet.cancel')}
          </button>
        </div>
      </div>
    </AppSheet>
  );
}

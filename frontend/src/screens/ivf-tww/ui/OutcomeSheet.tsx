'use client';

import { useTranslations } from 'next-intl';

import { type IvfOutcome, type IvfOutcomeResult, useRecordIvfOutcome } from '@/entities/ivf';
import { AppSheet } from '@/shared/sheet';
import { PrimaryButton, SecondaryButton } from '@/shared/ui';

/**
 * Confirm before the outcome closes the cycle (`POST /ivf/cycles/current/outcome`):
 * it ends the cycle and, unless positive, turns its medicine reminders off — the
 * copy says so and sends her to her doctor before stopping any medicine.
 */
export function OutcomeSheet({
  result,
  onClose,
  onSaved,
}: {
  result: IvfOutcomeResult;
  onClose: () => void;
  onSaved: (outcome: IvfOutcome) => void;
}) {
  const t = useTranslations('ivf');
  const record = useRecordIvfOutcome();
  return (
    <AppSheet
      open
      onClose={record.isPending ? () => undefined : onClose}
      size="half"
      title={t(`tww.confirm.title.${result}`)}
      footer={
        <div className="tww-sheet-actions">
          <PrimaryButton loading={record.isPending} onClick={() => record.mutate(result, { onSuccess: onSaved })}>
            {t('tww.confirm.save')}
          </PrimaryButton>
          <SecondaryButton variant="text" disabled={record.isPending} onClick={onClose}>
            {t('tww.confirm.cancel')}
          </SecondaryButton>
        </div>
      }
    >
      <p className="tww-sheet-body">{t(`tww.confirm.body.${result}`)}</p>
      {record.isError ? (
        <p className="ivf-error" role="alert">
          {t('tww.confirm.error')}
        </p>
      ) : null}
    </AppSheet>
  );
}

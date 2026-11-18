'use client';

import { useTranslations } from 'next-intl';
import { useState } from 'react';

import { getApiErrorMessage, getApiErrorStatus, isStepUpRequired } from '@/shared/api';
import { useRouter } from '@/shared/i18n';
import { Icon } from '@/shared/ui';

import { useDeleteAccount } from '../api/mutations';
import { StepUpPanel } from './StepUpPanel';

interface DeleteAccountConfirmProps {
  open: boolean;
  onClose: () => void;
}

/**
 * Confirmation sheet for permanent account deletion. Deliberately explicit and
 * calm (§11): it spells out that everything is erased and cannot be undone,
 * and the safe action (cancel) is the visually primary one. Confirming moves to
 * the step-up (T-M12-06): a fresh SMS code, and only then the deletion.
 */
export function DeleteAccountConfirm({ open, onClose }: DeleteAccountConfirmProps) {
  const t = useTranslations('account');
  const ts = useTranslations('profile.stepUp');
  const router = useRouter();
  const deleteAccount = useDeleteAccount();
  const [step, setStep] = useState<'confirm' | 'code'>('confirm');

  if (!open) return null;

  const handleVerified = (token: string) => {
    if (deleteAccount.isPending) return;
    deleteAccount.mutate(token, {
      onSuccess: () => router.replace('/signup'),
    });
  };

  const handleClose = () => {
    if (deleteAccount.isPending) return;
    deleteAccount.reset();
    setStep('confirm');
    onClose();
  };

  // A refused deletion in the code step: an expired / spent step-up asks for a new code, the hourly cap says so,
  // anything else is the server's message or the generic error.
  const actionError = !deleteAccount.isError
    ? null
    : isStepUpRequired(deleteAccount.error)
      ? ts('again')
      : getApiErrorStatus(deleteAccount.error) === 429
        ? ts('throttled')
        : getApiErrorMessage(deleteAccount.error) ?? t('delete.error');

  return (
    <div
      role="dialog"
      aria-modal="true"
      aria-label={t('delete.title')}
      className="del-backdrop"
      onClick={handleClose}
    >
      <div
        className="card del-card"
        onClick={(event) => event.stopPropagation()}
      >
        <span className="del-icon" aria-hidden>
          <Icon name="trash" size={22} />
        </span>

        {step === 'code' ? (
          <StepUpPanel
            body={ts('deleteBody')}
            onVerified={handleVerified}
            onCancel={handleClose}
            busy={deleteAccount.isPending}
            actionError={actionError}
          />
        ) : (
          <>
            <div className="del-title">{t('delete.title')}</div>
            <p className="del-body">{t('delete.warning')}</p>

            <div className="del-btns">
              <button type="button" className="del-confirm" onClick={() => setStep('code')}>
                {t('delete.confirm')}
              </button>
              <button type="button" className="del-cancel" onClick={handleClose}>
                {t('delete.cancel')}
              </button>
            </div>
          </>
        )}
      </div>
    </div>
  );
}

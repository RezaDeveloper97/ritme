'use client';

import { useTranslations } from 'next-intl';

import { getApiErrorMessage } from '@/shared/api';
import { useRouter } from '@/shared/i18n';
import { Icon } from '@/shared/ui';

import { useDeleteAccount } from '../api/mutations';

interface DeleteAccountConfirmProps {
  open: boolean;
  onClose: () => void;
}

/**
 * Confirmation sheet for permanent account deletion. Deliberately explicit and
 * calm (§11): it spells out that everything is erased and cannot be undone,
 * and the safe action (cancel) is the visually primary one.
 */
export function DeleteAccountConfirm({ open, onClose }: DeleteAccountConfirmProps) {
  const t = useTranslations('account');
  const router = useRouter();
  const deleteAccount = useDeleteAccount();

  if (!open) return null;

  const handleConfirm = () => {
    if (deleteAccount.isPending) return;
    deleteAccount.mutate(undefined, {
      onSuccess: () => router.replace('/signup'),
    });
  };

  const handleClose = () => {
    if (deleteAccount.isPending) return;
    deleteAccount.reset();
    onClose();
  };

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

        <div className="del-title">
          {t('delete.title')}
        </div>
        <p className="del-body">
          {t('delete.warning')}
        </p>

        {deleteAccount.isError ? (
          <p className="del-error" role="alert">
            {getApiErrorMessage(deleteAccount.error) ?? t('delete.error')}
          </p>
        ) : null}

        <div className="del-btns">
          <button
            type="button"
            className="del-confirm"
            onClick={handleConfirm}
            disabled={deleteAccount.isPending}
          >
            {deleteAccount.isPending ? t('delete.deleting') : t('delete.confirm')}
          </button>
          <button type="button" className="del-cancel" onClick={handleClose} disabled={deleteAccount.isPending}>
            {t('delete.cancel')}
          </button>
        </div>
      </div>
    </div>
  );
}

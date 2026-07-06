'use client';

import { useTranslations } from 'next-intl';

import { getApiErrorMessage } from '@/shared/api';
import { useRouter } from '@/shared/i18n';
import { Icon } from '@/shared/ui';

import { useDeleteAccount } from '../api/mutations';

const DANGER = '#E5484D';

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
      style={{
        position: 'fixed',
        inset: 0,
        zIndex: 60,
        display: 'flex',
        alignItems: 'flex-end',
        justifyContent: 'center',
        background: 'rgba(17,24,28,.45)',
        padding: 16,
      }}
      onClick={handleClose}
    >
      <div
        className="card"
        style={{ width: '100%', maxWidth: 420, padding: 22, textAlign: 'start' }}
        onClick={(event) => event.stopPropagation()}
      >
        <span
          style={{
            width: 46,
            height: 46,
            borderRadius: 14,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            background: 'rgba(229,72,77,.1)',
            color: DANGER,
          }}
        >
          <Icon name="trash" size={22} />
        </span>

        <div style={{ fontSize: 17, fontWeight: 800, color: 'var(--ink)', marginTop: 14 }}>
          {t('delete.title')}
        </div>
        <p style={{ fontSize: 13.5, lineHeight: 1.9, color: 'var(--muted)', margin: '8px 0 0' }}>
          {t('delete.warning')}
        </p>

        {deleteAccount.isError ? (
          <p style={{ fontSize: 13, color: DANGER, margin: '10px 0 0' }}>
            {getApiErrorMessage(deleteAccount.error) ?? t('delete.error')}
          </p>
        ) : null}

        <div style={{ display: 'flex', flexDirection: 'column', gap: 10, marginTop: 18 }}>
          <button
            type="button"
            onClick={handleConfirm}
            disabled={deleteAccount.isPending}
            style={{
              width: '100%',
              padding: '13px 16px',
              borderRadius: 14,
              border: 0,
              font: 'inherit',
              fontSize: 14,
              fontWeight: 700,
              cursor: deleteAccount.isPending ? 'default' : 'pointer',
              background: DANGER,
              color: '#fff',
              opacity: deleteAccount.isPending ? 0.65 : 1,
            }}
          >
            {deleteAccount.isPending ? t('delete.deleting') : t('delete.confirm')}
          </button>
          <button
            type="button"
            onClick={handleClose}
            disabled={deleteAccount.isPending}
            style={{
              width: '100%',
              padding: '13px 16px',
              borderRadius: 14,
              border: '1px solid var(--line)',
              font: 'inherit',
              fontSize: 14,
              fontWeight: 700,
              cursor: deleteAccount.isPending ? 'default' : 'pointer',
              background: 'transparent',
              color: 'var(--ink)',
            }}
          >
            {t('delete.cancel')}
          </button>
        </div>
      </div>
    </div>
  );
}

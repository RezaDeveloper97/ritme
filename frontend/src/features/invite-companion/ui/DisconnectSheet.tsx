'use client';

import { useTranslations } from 'next-intl';
import { useState } from 'react';

import { useRevokeCompanion } from '@/entities/companion';
import { getApiErrorMessage, getApiErrorStatus } from '@/shared/api';
import { AppSheet } from '@/shared/sheet';
import { Icon, SecondaryButton } from '@/shared/ui';
import '@/shared/ui/styles/cmp.css';

interface DisconnectSheetProps {
  /** The companion to cut off; `null` keeps the sheet closed. */
  companionId: number | null;
  name: string;
  onClose: () => void;
  /** DELETE /companions/{id} answered — access has ended. */
  onDisconnected: () => void;
}

/** A 422/429/503 carries a localized API message; anything else gets the generic copy. */
function errorText(error: unknown, fallback: string): string {
  const status = getApiErrorStatus(error);
  if (status === 422 || status === 429 || status === 503) return getApiErrorMessage(error) ?? fallback;
  return fallback;
}

/**
 * «وقتی قطع کنی چه می‌شود؟» (companion-v2 `sheetMarkup`): what stops, that she
 * can invite again, «بله، قطع کن» and the calm way out. Inline half sheet
 * (CLAUDE.md §4.1) — the caller owns which companion it is for.
 */
export function DisconnectSheet({ companionId, name, onClose, onDisconnected }: DisconnectSheetProps) {
  const t = useTranslations('companions.disconnect');
  // Keep the last target while the sheet plays its exit; a never-opened sheet holds 0 and never fires.
  const [target, setTarget] = useState(companionId);
  if (companionId !== null && companionId !== target) setTarget(companionId);
  const revoke = useRevokeCompanion(target ?? 0);
  const close = () => (revoke.isPending ? undefined : onClose());

  return (
    <AppSheet
      open={companionId !== null}
      onClose={close}
      size="half"
      title={t('title')}
      footer={
        <div className="cmp2-sheet-btns">
          <button
            type="button"
            className="cmp2-danger-btn"
            disabled={revoke.isPending}
            aria-busy={revoke.isPending || undefined}
            onClick={() => {
              if (companionId === null || revoke.isPending) return;
              revoke.mutate(undefined, { onSuccess: onDisconnected });
            }}
          >
            {t('confirm')}
          </button>
          <SecondaryButton variant="text" disabled={revoke.isPending} onClick={onClose}>
            {t('keep')}
          </SecondaryButton>
        </div>
      }
    >
      <ul className="cmp2-warns">
        <li className="cmp2-warn">
          <span className="cmp2-warn-icon" aria-hidden>
            <Icon name="x" size={14} strokeWidth={2.4} />
          </span>
          {t('noData', { name })}
        </li>
        <li className="cmp2-warn">
          <span className="cmp2-warn-icon" aria-hidden>
            <Icon name="x" size={14} strokeWidth={2.4} />
          </span>
          {t('noAlerts')}
        </li>
      </ul>
      <p className="cmp-sheet-body">{t('again')}</p>
      {revoke.isError ? (
        <p className="cmp-error" role="alert">
          {errorText(revoke.error, t('error'))}
        </p>
      ) : null}
    </AppSheet>
  );
}

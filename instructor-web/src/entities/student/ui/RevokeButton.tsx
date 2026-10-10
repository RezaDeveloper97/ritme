'use client';

import { useTranslations } from 'next-intl';
import { useState } from 'react';

import { useErrorMessage } from '@/shared/i18n';

import { useRevokeGrant } from '../api/queries';

/** Two-step «لغو دسترسی»: first tap asks, second confirms. */
export function RevokeButton({ grantId }: { grantId: number }) {
  const t = useTranslations('grant');
  const errorMessage = useErrorMessage();
  const revoke = useRevokeGrant();
  const [asking, setAsking] = useState(false);

  if (!asking) {
    return (
      <button type="button" className="pill-btn pill-btn--danger" onClick={() => setAsking(true)}>
        {t('revoke')}
      </button>
    );
  }
  return (
    <span className="revoke-confirm" role="group" aria-label={t('revokeAsk')}>
      <button
        type="button"
        className="pill-btn pill-btn--danger"
        disabled={revoke.isPending}
        onClick={() => revoke.mutate(grantId, { onSuccess: () => setAsking(false) })}
      >
        {t('revokeYes')}
      </button>
      <button type="button" className="pill-btn" onClick={() => setAsking(false)}>
        {t('revokeNo')}
      </button>
      {revoke.isError ? <span className="form-error">{errorMessage(revoke.error)}</span> : null}
    </span>
  );
}

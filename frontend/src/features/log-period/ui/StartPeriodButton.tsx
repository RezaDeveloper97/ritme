'use client';

import { useTranslations } from 'next-intl';
import { useState } from 'react';

import { Icon } from '@/shared/ui';

import { useStartPeriod } from '../api/mutations';

/**
 * "Start period" action on the home screen. Logging a period re-anchors the
 * whole cycle, so an accidental tap is costly — the button asks for a second
 * confirming tap before it commits (a lightweight guard that needs no modal).
 * While the request is in flight it's disabled; failures surface an inline
 * retry hint. All copy is i18n'd and RTL-safe (CLAUDE.md §6, §12).
 */
export function StartPeriodButton() {
  const t = useTranslations('logPeriod');
  const { mutate, isPending, isError } = useStartPeriod();
  const [confirming, setConfirming] = useState(false);

  const commit = () => {
    if (isPending) return;
    if (!confirming) {
      setConfirming(true);
      return;
    }
    mutate(undefined, { onSettled: () => setConfirming(false) });
  };

  const label = isPending
    ? t('pending')
    : confirming
      ? t('confirm')
      : t('start');

  return (
    <div style={{ padding: '16px 16px 0' }}>
      <button
        className="btn"
        onClick={commit}
        disabled={isPending}
        aria-busy={isPending}
        style={{
          height: 40,
          borderRadius: 14,
          gap: 8,
          fontSize: 14,
          background: confirming ? 'var(--brand)' : '#fff',
          border: '1px solid #FCE7F3',
          color: confirming ? '#fff' : 'var(--brand)',
          fontWeight: 600,
          cursor: isPending ? 'default' : 'pointer',
          opacity: isPending ? 0.75 : 1,
          transition: 'background .2s, color .2s',
        }}
      >
        <Icon
          name="drop"
          size={16}
          fill={confirming ? '#fff' : 'var(--brand)'}
          strokeWidth={0}
        />
        {label}
      </button>
      {isError && (
        <p
          role="alert"
          style={{ margin: '8px 2px 0', fontSize: 12, fontWeight: 600, color: 'var(--brand)', textAlign: 'start' }}
        >
          {t('error')}
        </p>
      )}
    </div>
  );
}

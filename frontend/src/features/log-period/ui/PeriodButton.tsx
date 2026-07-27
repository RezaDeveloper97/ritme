'use client';

import { useTranslations } from 'next-intl';
import { useState } from 'react';

import { Icon } from '@/shared/ui';

import { useEndPeriod, usePeriodStatus, useStartPeriod } from '../api/mutations';

/**
 * Period action on the home screen. It reads the current period state and
 * flips between "Start period" and "End period": when a period is ongoing it
 * offers to end it, otherwise to start one. Logging a period re-anchors the
 * whole cycle, so an accidental tap is costly — the button asks for a second
 * confirming tap before it commits (a lightweight guard that needs no modal).
 * While the request is in flight it's disabled; failures surface an inline
 * retry hint. All copy is i18n'd and RTL-safe (CLAUDE.md §6, §12).
 */
export function PeriodButton() {
  const t = useTranslations('logPeriod');
  const { data: status } = usePeriodStatus();
  const start = useStartPeriod();
  const end = useEndPeriod();
  const [confirming, setConfirming] = useState(false);

  const active = status?.active ?? false;
  const action = active ? end : start;
  const { isPending, isError } = action;

  const commit = () => {
    if (isPending) return;
    if (!confirming) {
      setConfirming(true);
      return;
    }
    action.mutate(undefined, { onSettled: () => setConfirming(false) });
  };

  const label = isPending
    ? t('pending')
    : confirming
      ? t('confirm')
      : active
        ? t('end')
        : t('start');

  return (
    <div className="sec">
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
          background: confirming ? 'var(--brand)' : 'var(--surface)',
          border: '1px solid var(--pink-bg)',
          color: confirming ? 'var(--on-accent)' : 'var(--brand)',
          fontWeight: 600,
          cursor: isPending ? 'default' : 'pointer',
          opacity: isPending ? 0.75 : 1,
          transition: 'background .2s, color .2s',
        }}
      >
        <Icon
          name="drop"
          size={16}
          fill={confirming ? 'var(--on-accent)' : 'var(--brand)'}
          strokeWidth={0}
        />
        {label}
      </button>
      {isError && (
        <p
          role="alert"
          className="ped-note is-error"
        >
          {t('error')}
        </p>
      )}
    </div>
  );
}

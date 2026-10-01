'use client';

import { useQueryClient } from '@tanstack/react-query';
import { useLocale, useTranslations } from 'next-intl';
import { useCallback } from 'react';

import { plusKeys, usePlusStatus, useServerCountdown, type PlusTrialOffer } from '@/entities/plus';
import type { Locale } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib/date';
import { Icon } from '@/shared/ui';

const pad = (n: number) => String(n).padStart(2, '0');

function Banner({ offer, since, onOpen }: { offer: PlusTrialOffer; since: number; onOpen: () => void }) {
  const t = useTranslations('plus.trialBanner');
  const loc = useLocale() as Locale;
  const queryClient = useQueryClient();
  // At zero the offer is over server-side too: refetch so the banner leaves.
  const onExpire = useCallback(() => void queryClient.invalidateQueries({ queryKey: plusKeys.all }), [queryClient]);
  const left = useServerCountdown(offer.secondsLeft, since, onExpire);
  if (left.seconds === 0) return null;

  const cells = [
    { value: left.days, label: t('days') },
    { value: left.hours, label: t('hours') },
    { value: left.minutes, label: t('minutes') },
  ];
  return (
    <button
      type="button"
      className="plus-tbanner"
      onClick={onOpen}
      aria-haspopup="dialog"
      aria-label={t('aria', { days: formatNumber(left.days, loc) })}
    >
      <span className="plus-tbanner-icon" aria-hidden>
        <Icon name="crown" size={19} strokeWidth={1.8} />
      </span>
      <span className="plus-tbanner-text">
        <b className="plus-tbanner-title">{t('title')}</b>
        <span className="plus-tbanner-sub">
          {t.rich('offer', {
            percent: formatNumber(offer.percent, loc),
            hl: (chunks) => <span className="plus-tbanner-hl">{chunks}</span>,
          })}
        </span>
      </span>
      <span className="plus-tbanner-clock" aria-hidden>
        {cells.map((c) => (
          <span key={c.label} className="plus-tbanner-cell">
            <b>{formatNumber(pad(c.value), loc)}</b>
            <span>{c.label}</span>
          </span>
        ))}
      </span>
    </button>
  );
}

/**
 * «ریتمی پلاس برای تو باز است» (B-N2-08, nbl_/nbd_Prem_TrialHome): the
 * floating trial-offer card above the bottom nav with a live countdown from
 * the server's `seconds_left`. Renders nothing unless an offer runs (no trial,
 * subscribed, offer off — `trial_offer: null`), while loading or on error: it
 * is a promotion, never a reason to block the home. Mount it as a direct child
 * of `.view`, next to `<BottomNav />`; the caller decides who sees it (not teen,
 * QUESTIONS #70/#72) and owns the sheet `onOpen` opens.
 */
export function PlusTrialBanner({ onOpen }: { onOpen: () => void }) {
  const status = usePlusStatus();
  const offer = status.data?.trialOffer ?? null;
  if (!offer || offer.secondsLeft <= 0) return null;
  return <Banner key={status.dataUpdatedAt} offer={offer} since={status.dataUpdatedAt} onOpen={onOpen} />;
}

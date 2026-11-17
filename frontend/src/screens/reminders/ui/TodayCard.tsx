'use client';

import clsx from 'clsx';
import { useLocale, useTranslations } from 'next-intl';

import { useCareToday } from '@/entities/care-reminder';
import { useLogIntake } from '@/features/log-intake';
import type { Locale } from '@/shared/i18n';
import { pilot } from '@/shared/pilot';
import { formatNumber } from '@/shared/lib/date';
import { CheckTick, Icon, Skeleton, SkeletonGroup } from '@/shared/ui';

import { type DoseCardState, doseCardState } from '../model/view';

type T = ReturnType<typeof useTranslations<'care'>>;

/** «۱ قرص · بعد از غذا» from a dose's detail parts. */
function doseLine(detail: NonNullable<DoseCardState['detail']>, t: T, locale: Locale): string {
  const unit = detail.unit ? t(`doseUnits.${detail.unit}`) : (detail.unitLabel ?? '');
  const base = t('doseLine', { amount: formatNumber(detail.amount, locale), unit });
  return detail.meal ? `${base} · ${t(`meals.${detail.meal}`)}` : base;
}


/**
 * «امروز» — today's doses as a horizontal strip of cards (time, check, name).
 * Ticking goes through `log-intake`, which updates every cached `/care/today`
 * optimistically, so the home card agrees without a refetch.
 *
 * Privacy (§11): which doses were taken is health data — rendered, never logged.
 */
export function TodayCard() {
  const t = useTranslations('care');
  const locale = useLocale() as Locale;
  const query = useCareToday();
  const logIntake = useLogIntake();

  const data = query.data;
  const pendingKey =
    logIntake.isPending && logIntake.variables
      ? `${logIntake.variables.reminderId}-${logIntake.variables.slot}`
      : null;

  return (
    <section className="rmd-today" aria-labelledby="rmd-today-title">
      <div className="rmd-today-head">
        <h2 id="rmd-today-title" className="rmd-today-title">
          {t('today.title')}
        </h2>
        {data && data.total > 0 && (
          <span className="rmd-badge">
            <Icon name="check" size={13} strokeWidth={2.4} />
            {t('today.progress', {
              taken: formatNumber(data.takenCount, locale),
              total: formatNumber(data.total, locale),
            })}
          </span>
        )}
      </div>

      {query.isError ? (
        <div className="rmd-state">
          <p className="rmd-empty">{t('loadError')}</p>
          <button type="button" className="rmd-retry" onClick={() => void query.refetch()}>
            {t('retry')}
          </button>
        </div>
      ) : !data ? (
        <SkeletonGroup label={t('loading')} className="rmd-doses">
          <Skeleton shape="block" className="rmd-dose-skel" />
          <Skeleton shape="block" className="rmd-dose-skel" />
        </SkeletonGroup>
      ) : data.doses.length === 0 ? (
        <p className="rmd-empty rmd-today-empty">{t('today.empty')}</p>
      ) : (
        <ul className="rmd-doses">
          {data.doses.map(doseCardState).map((card) => (
            <li key={card.key} className={clsx('rmd-dose', card.taken && 'is-taken')}>
              <div className="rmd-dose-top">
                <span className="rmd-dose-time">
                  {t('slotWithPeriod', {
                    time: formatNumber(card.clock, locale),
                    period: t(`slotPeriod.${card.period}`),
                  })}
                </span>
                <button
                  type="button"
                  className="rmd-check"
                  {...pilot(`care.dose:${card.key.replace(':', '')}`)}
                  aria-pressed={card.taken}
                  aria-label={t(card.taken ? 'today.markNotTaken' : 'today.markTaken', {
                    title: card.title,
                  })}
                  // Optimistic already; blocking only this card while its write
                  // is in flight keeps a double-tap from queueing an undo.
                  disabled={pendingKey === card.key}
                  onClick={() =>
                    logIntake.mutate({
                      reminderId: card.reminderId,
                      date: data.date,
                      slot: card.slot,
                      taken: card.nextTaken,
                    })
                  }
                >
                  <span className="rmd-check-dot" aria-hidden>
                    <CheckTick on={card.taken} />
                  </span>
                </button>
              </div>
              <b className="rmd-dose-name">{card.title}</b>
              {card.detail && <span className="rmd-dose-detail">{doseLine(card.detail, t, locale)}</span>}
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}

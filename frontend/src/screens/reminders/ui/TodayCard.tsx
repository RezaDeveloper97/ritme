'use client';

import clsx from 'clsx';
import { useLocale, useTranslations } from 'next-intl';

import { useCareToday } from '@/entities/care-reminder';
import { useLogIntake } from '@/features/log-intake';
import type { Locale } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib/date';
import { Icon } from '@/shared/ui';

import { doseCardState } from '../model/view';

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
        <div className="rmd-doses" aria-busy="true" aria-label={t('loading')}>
          <span className="skeleton-line rmd-dose-skel" />
          <span className="skeleton-line rmd-dose-skel" />
        </div>
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
                    {card.taken && <Icon name="check" size={14} strokeWidth={3} />}
                  </span>
                </button>
              </div>
              <b className="rmd-dose-name">{card.title}</b>
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}

'use client';

import Link from 'next/link';
import { useLocale, useTranslations } from 'next-intl';
import type { CSSProperties } from 'react';

import { formatNumber } from '@/shared/lib';
import { ErrorState, Panel, Skeleton } from '@/shared/ui';

import { pregnancyWeeksApi } from '../api/pregnancy-weeks';
import { useWeekDetailsList } from '../api/week-details';
import { cellFilled, cellHref, weekCells } from '../lib/cells';
import { byTrimester } from '../lib/trimesters';

/**
 * /pregnancy-weeks — the weeks as a gestation strip in three trimesters; a filled cell has
 * content (a v1 text row or structured details), an outlined one opens a new week
 * (Blade pregnancy-weeks.index).
 */
export function PregnancyWeeksScreen() {
  const t = useTranslations('pregnancyWeeks');
  const locale = useLocale();
  const query = pregnancyWeeksApi.useList();
  const details = useWeekDetailsList();
  const n = (v: number) => formatNumber(v, locale);

  if (query.error && !query.data) {
    return (
      <Panel title={t('title')}>
        <ErrorState error={query.error} onRetry={() => query.refetch()} />
      </Panel>
    );
  }

  // The details list only adds to the grid: wait for it, but a failure still shows the text rows.
  const detailsSettled = details.data !== undefined || details.error !== null;
  const items = query.data && detailsSettled ? weekCells(query.data.items, details.data?.items) : undefined;
  const filled = items?.filter(cellFilled).length ?? 0;

  return (
    <Panel
      title={t('title')}
      actions={
        items ? (
          <span className="text-[13px] text-ink-3">{t('coverage', { filled: n(filled), total: n(items.length) })}</span>
        ) : null
      }
    >
      {!items ? (
        <div className="flex flex-col gap-3" aria-busy="true">
          <Skeleton className="h-5 w-40" />
          <Skeleton className="h-28 w-full" />
          <Skeleton className="h-28 w-full" />
        </div>
      ) : (
        <div className="content-map">
          {byTrimester(items).map((band) => {
            const bandFilled = band.weeks.filter(cellFilled).length;
            const meter = { inlineSize: `${(bandFilled / band.weeks.length) * 100}%` } as CSSProperties;
            return (
              <section key={band.key} className="content-map-band" aria-label={t(`bands.${band.key}`)}>
                <div className="content-map-legend">
                  <span className="font-bold">
                    {t(`bands.${band.key}`)}
                    <span className="ms-2 font-normal text-muted">
                      {t('weekRange', { from: n(band.weeks[0]?.week ?? band.from), to: n(band.weeks.at(-1)?.week ?? band.to) })}
                    </span>
                  </span>
                  <span className="tabular-nums text-ink-3">
                    {n(bandFilled)}/{n(band.weeks.length)}
                  </span>
                </div>
                <div className="meter" aria-hidden="true">
                  <span style={meter} />
                </div>
                <div className="content-map-cells">
                  {band.weeks.map((w) => (
                    <Link
                      key={w.week}
                      href={cellHref(w)}
                      className="map-cell"
                      data-filled={cellFilled(w) || undefined}
                      aria-label={cellFilled(w) ? t('editWeek', { week: n(w.week) }) : t('addWeek', { week: n(w.week) })}
                    >
                      <span className="map-cell-num">{n(w.week)}</span>
                      <span className="map-cell-state">{cellFilled(w) ? t('hasContent') : t('add')}</span>
                    </Link>
                  ))}
                </div>
              </section>
            );
          })}
        </div>
      )}
    </Panel>
  );
}

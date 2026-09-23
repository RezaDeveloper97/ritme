'use client';

import Link from 'next/link';
import { useLocale, useTranslations } from 'next-intl';
import type { CSSProperties } from 'react';

import { formatNumber } from '@/shared/lib';
import { Badge, ErrorState, Panel, Skeleton } from '@/shared/ui';

import { phaseContentsApi } from '../api/phase-contents';

/** /phase-contents — one cell per cycle sub-phase; filled = has content (Blade phase-contents.index). */
export function PhaseContentsScreen() {
  const t = useTranslations('phaseContents');
  const tc = useTranslations('crud');
  const locale = useLocale();
  const query = phaseContentsApi.useList();
  const n = (v: number) => formatNumber(v, locale);

  if (query.error && !query.data) {
    return (
      <Panel title={t('title')}>
        <ErrorState error={query.error} onRetry={() => query.refetch()} />
      </Panel>
    );
  }
  const items = query.data?.items;
  const current = items?.filter((p) => !p.legacy) ?? [];
  const legacy = items?.filter((p) => p.legacy) ?? [];
  const filled = current.filter((p) => p.id !== null).length;
  const meter = { inlineSize: `${current.length ? (filled / current.length) * 100 : 0}%` } as CSSProperties;

  return (
    <Panel
      title={t('title')}
      actions={
        items ? <span className="text-[13px] text-ink-3">{t('coverage', { filled: n(filled), total: n(current.length) })}</span> : null
      }
    >
      {!items ? (
        <div className="flex flex-col gap-3" aria-busy="true">
          <Skeleton className="h-5 w-40" />
          <Skeleton className="h-32 w-full" />
        </div>
      ) : (
        <div className="content-map">
          <section className="content-map-band" aria-label={t('phases')}>
            <div className="meter" aria-hidden="true">
              <span style={meter} />
            </div>
            <div className="content-map-cells" data-wide="true">
              {current.map((p) => (
                <Link
                  key={p.value}
                  href={p.id !== null ? `/phase-contents/${p.id}` : `/phase-contents/new?phase=${encodeURIComponent(p.value)}`}
                  className="map-cell"
                  data-filled={p.id !== null || undefined}
                >
                  <span className="map-cell-label">{p.label}</span>
                  <span className="map-cell-state">{p.id !== null ? t('hasContent') : t('add')}</span>
                </Link>
              ))}
            </div>
          </section>
          {legacy.length ? (
            <section className="content-map-band" aria-label={t('legacyTitle')}>
              <div className="content-map-legend">
                <span className="font-bold">{t('legacyTitle')}</span>
                <Badge tone="amber">{tc('legacy')}</Badge>
              </div>
              <p className="field-hint m-0">{t('legacyHint')}</p>
              <div className="content-map-cells" data-wide="true">
                {legacy.map((p) =>
                  p.id !== null ? (
                    <Link key={p.value} href={`/phase-contents/${p.id}`} className="map-cell" data-filled="true">
                      <span className="map-cell-label">{p.label}</span>
                      <span className="map-cell-state" dir="ltr">
                        {p.value}
                      </span>
                    </Link>
                  ) : null,
                )}
              </div>
            </section>
          ) : null}
        </div>
      )}
    </Panel>
  );
}

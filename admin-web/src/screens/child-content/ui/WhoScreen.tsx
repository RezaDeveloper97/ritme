'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useState } from 'react';

import { useNumber } from '@/shared/lib';
import { Badge, ErrorState, PageHeader, Pagination, Panel, Select, Skeleton } from '@/shared/ui';

import { useWho, type WhoQuery } from '../api/child-content';
import { useAgeLabel } from './labels';
import { ChildTabs } from './parts';

const INDICATORS = ['weight', 'length', 'head'];
const SEXES = ['girl', 'boy'];
const STEPS = ['month', 'week', 'day'];

/** /children-content/who — read-only WHO Child Growth Standards: L/M/S and the derived percentiles (admin-api.md §18). */
export function WhoScreen() {
  const t = useTranslations('childContent');
  const n = useNumber();
  const ageLabel = useAgeLabel();
  const [query, setQuery] = useState<WhoQuery>({ indicator: 'weight', sex: 'girl', step: 'month', page: 1 });
  const who = useWho(query);
  const data = who.data;
  const unit = data ? t(`units.${data.unit}` as 'units.kg') : '';
  const set = (patch: Partial<WhoQuery>) => setQuery((q) => ({ ...q, ...patch, page: patch.page ?? 1 }));
  const locale = useLocale();
  const num = (v: number, digits: number) =>
    new Intl.NumberFormat(locale === 'fa' ? 'fa-IR' : locale, { maximumFractionDigits: digits, useGrouping: false }).format(v);
  const age = (a: number) => (query.step === 'month' ? ageLabel(a) : query.step === 'week' ? t('weekN', { n: n(a) }) : t('dayN', { n: n(a) }));

  return (
    <div className="flex flex-col gap-4">
      <PageHeader title={t('pages.who')} meta={<Badge tone="data">{t('readOnlyBadge')}</Badge>} />
      <ChildTabs active="who" />
      <p className="field-hint m-0">{t('whoHint')}</p>
      <Panel bodyClassName="">
        <div className="filter-bar">
          <Select
            label={t('who.indicator')}
            className="w-56"
            value={query.indicator}
            onChange={(e) => set({ indicator: e.target.value })}
            options={(data?.options.indicators ?? INDICATORS).map((v) => ({ value: v, label: t(`who.indicators.${v}` as 'who.indicators.weight') }))}
          />
          <Select
            label={t('who.sex')}
            className="w-36"
            value={query.sex}
            onChange={(e) => set({ sex: e.target.value })}
            options={(data?.options.sexes ?? SEXES).map((v) => ({ value: v, label: t(`who.sexes.${v}` as 'who.sexes.girl') }))}
          />
          <Select
            label={t('who.step')}
            className="w-36"
            value={query.step}
            onChange={(e) => set({ step: e.target.value })}
            options={(data?.options.steps ?? STEPS).map((v) => ({ value: v, label: t(`who.steps.${v}` as 'who.steps.month') }))}
          />
        </div>
        {who.error && !data ? (
          <ErrorState error={who.error} onRetry={() => who.refetch()} />
        ) : !data ? (
          <div className="flex flex-col gap-2 p-4">
            <Skeleton className="h-10 w-full" />
            <Skeleton className="h-10 w-full" />
            <Skeleton className="h-10 w-full" />
          </div>
        ) : (
          <>
            <div className="table-wrap">
              <table className="data-table">
                <caption className="sr-only">{t('pages.who')}</caption>
                <thead>
                  <tr>
                    <th scope="col">{t('who.age')}</th>
                    <th scope="col">{t('who.day')}</th>
                    <th scope="col">L</th>
                    <th scope="col">M</th>
                    <th scope="col">S</th>
                    {(['p3', 'p15', 'p50', 'p85', 'p97'] as const).map((p) => (
                      <th key={p} scope="col">
                        {t(`who.${p}`)} <span className="font-normal text-muted">({unit})</span>
                      </th>
                    ))}
                  </tr>
                </thead>
                <tbody>
                  {data.items.map((r) => (
                    <tr key={r.day}>
                      <td className="whitespace-nowrap">{age(r.age)}</td>
                      <td className="cell-num">{n(r.day)}</td>
                      <td className="cell-num" dir="ltr">
                        {num(r.l, 4)}
                      </td>
                      <td className="cell-num" dir="ltr">
                        {num(r.m, 4)}
                      </td>
                      <td className="cell-num" dir="ltr">
                        {num(r.s, 5)}
                      </td>
                      <td className="cell-num">{num(r.p3, 2)}</td>
                      <td className="cell-num">{num(r.p15, 2)}</td>
                      <td className="cell-num font-semibold">{num(r.p50, 2)}</td>
                      <td className="cell-num">{num(r.p85, 2)}</td>
                      <td className="cell-num">{num(r.p97, 2)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
            <Pagination meta={data.meta} onPage={(page) => set({ page })} />
            <p className="field-hint m-0 px-4 pb-4" dir="auto">
              {t('who.source')}: {data.source}
            </p>
          </>
        )}
      </Panel>
      <p className="field-hint m-0">{t('who.importNote')}</p>
    </div>
  );
}

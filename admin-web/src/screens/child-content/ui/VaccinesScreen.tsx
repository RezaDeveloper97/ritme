'use client';

import Link from 'next/link';
import { useTranslations } from 'next-intl';

import { useLocalized } from '@/shared/i18n';
import { excerpt, useNumber } from '@/shared/lib';
import { Badge, ErrorState, Icon, PageHeader, Panel, Skeleton } from '@/shared/ui';

import { useChildItems, type ChildItem } from '../api/child-content';
import { KINDS, approxDays, groupByVisit } from '../lib/meta';
import { ChildTabs, ItemActions, ReadOnlyNotice, ReviewBadges, itemHref, useCanWrite } from './parts';
import { useAgeLabel, useVisitLabel } from './labels';

/** /children-content/vaccines — the vaccine schedule grouped into visits (catalog order, admin-api.md §18). */
export function VaccinesScreen() {
  const t = useTranslations('childContent');
  const n = useNumber();
  const canWrite = useCanWrite();
  const query = useChildItems(KINDS.vaccines.group);
  const rows = query.data ?? [];
  const pending = rows.filter((r) => r.needs_review).length;

  return (
    <div className="flex flex-col gap-4">
      <PageHeader
        title={t('pages.vaccines')}
        meta={
          query.data ? (
            <>
              <span>{t('dosesCount', { count: n(rows.length) })}</span>
              {pending ? <Badge tone="amber">{t('pendingCount', { count: n(pending) })}</Badge> : <Badge tone="green">{t('allReviewed')}</Badge>}
            </>
          ) : null
        }
        actions={
          canWrite ? (
            <Link href={itemHref('vaccines', null)} className="btn btn-primary btn-sm">
              <Icon name="plus" size={15} />
              {t('newDose')}
            </Link>
          ) : null
        }
      />
      <ChildTabs active="vaccines" />
      <p className="field-hint m-0">{t('vaccinesHint')}</p>
      {canWrite ? null : <ReadOnlyNotice />}
      {query.error && !query.data ? (
        <Panel>
          <ErrorState error={query.error} onRetry={() => query.refetch()} />
        </Panel>
      ) : !query.data ? (
        <Panel>
          <div className="flex flex-col gap-2">
            <Skeleton className="h-10 w-full" />
            <Skeleton className="h-10 w-full" />
            <Skeleton className="h-10 w-full" />
          </div>
        </Panel>
      ) : rows.length === 0 ? (
        <Panel>
          <p className="cell-empty m-0 p-6 text-center text-muted">{t('emptyVaccines')}</p>
        </Panel>
      ) : (
        groupByVisit(rows).map((g) => <VisitPanel key={g.visit || '_'} visit={g.visit} ageMonths={g.ageMonths} rows={g.rows} />)
      )}
    </div>
  );
}

function VisitPanel({ visit, ageMonths, rows }: { visit: string; ageMonths: number | null; rows: ChildItem[] }) {
  const t = useTranslations('childContent');
  const n = useNumber();
  const localize = useLocalized();
  const canWrite = useCanWrite();
  const visitLabel = useVisitLabel();
  const ageLabel = useAgeLabel();

  return (
    <Panel
      bodyClassName=""
      title={
        <span className="flex flex-wrap items-center gap-2">
          {visit ? visitLabel(visit) : t('noVisit')}
          {visit ? (
            <span dir="ltr" className="cell-mono text-xs text-muted">
              {visit}
            </span>
          ) : null}
          {ageMonths !== null ? (
            <Badge tone="data">
              {ageLabel(ageMonths)}
              {ageMonths > 0 ? ` · ${t('approxDays', { days: n(approxDays(ageMonths)) })}` : null}
            </Badge>
          ) : null}
        </span>
      }
      actions={
        canWrite && visit ? (
          <Link href={itemHref('vaccines', null, { visit, age: ageMonths ?? 0 })} className="btn btn-sm">
            <Icon name="plus" size={14} />
            {t('addDoseToVisit')}
          </Link>
        ) : null
      }
    >
      <div className="table-wrap">
        <table className="data-table">
          <caption className="sr-only">{visit ? visitLabel(visit) : t('noVisit')}</caption>
          <thead>
            <tr>
              <th scope="col">{t('dose')}</th>
              <th scope="col">{t('protectsAgainst')}</th>
              <th scope="col">{t('review')}</th>
              <th scope="col">
                <span className="sr-only">{t('actions')}</span>
              </th>
            </tr>
          </thead>
          <tbody>
            {rows.map((r) => (
              <tr key={r.id}>
                <td className="cell-wrap">
                  <span className="flex flex-col">
                    <strong>{localize(r.title) || r.code}</strong>
                    <span dir="ltr" className="cell-mono text-xs text-muted">
                      {r.code}
                    </span>
                  </span>
                </td>
                <td className="cell-wrap">{excerpt(localize(r.body), 80) || '—'}</td>
                <td>
                  <ReviewBadges kind="vaccines" row={r} />
                </td>
                <td className="cell-actions">
                  <ItemActions kind="vaccines" row={r} editHref={itemHref('vaccines', r.id)} />
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </Panel>
  );
}

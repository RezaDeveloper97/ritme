'use client';

import Link from 'next/link';
import { useTranslations } from 'next-intl';
import { usePathname, useRouter, useSearchParams } from 'next/navigation';

import { useLocalized } from '@/shared/i18n';
import { cn, excerpt, useNumber } from '@/shared/lib';
import { Badge, ErrorState, Icon, PageHeader, Panel, Skeleton } from '@/shared/ui';

import { useChildItems, type ChildItem } from '../api/child-content';
import { KINDS, MONTH_KINDS, ageOf, bandsOf } from '../lib/meta';
import { useAgeLabel, useCodeLabels } from './labels';
import { ChildTabs, ItemActions, ReadOnlyNotice, ReviewBadges, itemHref, useCanWrite } from './parts';

type MonthKind = (typeof MONTH_KINDS)[number];

/**
 * /children-content/milestones?month= — one age band at a time: the milestones (by domain), «بازی‌های این ماه»,
 * «کی به پزشک بگم» and the «این هفته …» age note (admin-api.md §18).
 */
export function MilestonesScreen() {
  const t = useTranslations('childContent');
  const n = useNumber();
  const canWrite = useCanWrite();
  const ageLabel = useAgeLabel();
  const router = useRouter();
  const pathname = usePathname();
  const search = useSearchParams();
  const queries = {
    milestones: useChildItems(KINDS.milestones.group),
    activities: useChildItems(KINDS.activities.group),
    notes: useChildItems(KINDS.notes.group),
    'age-notes': useChildItems(KINDS['age-notes'].group),
  } satisfies Record<MonthKind, ReturnType<typeof useChildItems>>;
  const all = MONTH_KINDS.map((k) => queries[k]);
  const failed = all.find((q) => q.error && !q.data);
  const ready = all.every((q) => q.data);

  const bands = bandsOf(all.flatMap((q) => q.data ?? []));
  const raw = Number(search.get('month'));
  const month = search.get('month') !== null && bands.includes(raw) ? raw : (bands[0] ?? 0);
  const pending = all.flatMap((q) => q.data ?? []).filter((r) => r.needs_review).length;

  return (
    <div className="flex flex-col gap-4">
      <PageHeader
        title={t('pages.milestones')}
        meta={ready ? pending ? <Badge tone="amber">{t('pendingCount', { count: n(pending) })}</Badge> : <Badge tone="green">{t('allReviewed')}</Badge> : null}
        actions={
          canWrite ? (
            <Link href={itemHref('milestones', null)} className="btn btn-primary btn-sm">
              <Icon name="plus" size={15} />
              {t('newMonth')}
            </Link>
          ) : null
        }
      />
      <ChildTabs active="milestones" />
      <p className="field-hint m-0">{t('milestonesHint')}</p>
      {canWrite ? null : <ReadOnlyNotice />}
      {failed ? (
        <Panel>
          <ErrorState error={failed.error} onRetry={() => all.forEach((q) => q.refetch())} />
        </Panel>
      ) : !ready ? (
        <Panel>
          <div className="flex flex-col gap-2">
            <Skeleton className="h-10 w-full" />
            <Skeleton className="h-24 w-full" />
          </div>
        </Panel>
      ) : (
        <>
          {bands.length ? (
            <div role="tablist" aria-label={t('monthsLabel')} className="flex flex-wrap gap-1.5">
              {bands.map((b) => (
                <button
                  key={b}
                  type="button"
                  role="tab"
                  aria-selected={b === month}
                  className={cn('btn btn-sm min-h-11', b === month ? 'btn-primary' : 'btn-ghost')}
                  onClick={() => router.replace(`${pathname}?month=${b}`, { scroll: false })}
                >
                  {ageLabel(b)}
                </button>
              ))}
            </div>
          ) : null}
          {MONTH_KINDS.map((k) => (
            <KindPanel key={k} kind={k} month={month} rows={(queries[k].data ?? []).filter((r) => ageOf(r.meta) === month)} />
          ))}
        </>
      )}
    </div>
  );
}

function KindPanel({ kind, month, rows }: { kind: MonthKind; month: number; rows: ChildItem[] }) {
  const t = useTranslations('childContent');
  const localize = useLocalized();
  const labels = useCodeLabels();
  const canWrite = useCanWrite();
  return (
    <Panel
      bodyClassName=""
      title={t(`sections.${kind}`)}
      actions={
        canWrite ? (
          <Link href={itemHref(kind, null, { month })} className="btn btn-sm">
            <Icon name="plus" size={14} />
            {t('add')}
          </Link>
        ) : null
      }
    >
      {rows.length === 0 ? (
        <p className="cell-empty m-0 p-5 text-center text-muted">{t('emptySection')}</p>
      ) : (
        <ul className="m-0 flex list-none flex-col p-0">
          {rows.map((r) => (
            <li key={r.id} className="flex flex-wrap items-start gap-3 border-b border-line px-4 py-3 last:border-b-0">
              <div className="flex min-w-0 flex-1 flex-col gap-1">
                <span className="flex flex-wrap items-center gap-2">
                  <strong>{localize(r.title) || r.code}</strong>
                  {kind === 'milestones' && typeof r.meta?.domain === 'string' ? (
                    <Badge tone="data">{labels.domain(r.meta.domain)}</Badge>
                  ) : null}
                </span>
                {localize(r.body) ? <span className="text-sm text-ink-3">{excerpt(localize(r.body), 140)}</span> : null}
                <span className="flex flex-wrap items-center gap-2">
                  <span dir="ltr" className="cell-mono text-xs text-muted">
                    {r.code}
                  </span>
                  <ReviewBadges kind={kind} row={r} />
                </span>
              </div>
              <ItemActions kind={kind} row={r} editHref={itemHref(kind, r.id)} />
            </li>
          ))}
        </ul>
      )}
    </Panel>
  );
}

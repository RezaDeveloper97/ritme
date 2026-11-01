'use client';

import Link from 'next/link';
import { useLocale, useTranslations } from 'next-intl';
import { useState, type ReactNode } from 'react';

import { useCan, useCurrentAdmin } from '@/features/auth';
import { cn, formatDateTime, formatNumber } from '@/shared/lib';
import {
  Badge,
  Button,
  DataTable,
  ErrorState,
  Pagination,
  Panel,
  Skeleton,
  toast,
  useNotifyError,
  type Column,
} from '@/shared/ui';

import { SIGNALS, SOURCES, useClaim, useSafety, type ReviewItem, type SafetyData, type Source } from '../api/safety';
import { ageParts, sourceHref } from '../lib/links';
import { SelfTestRuns } from './SelfTestRuns';

/** The review queue can hold hundreds of catalog items; show it ten rows at a time. */
const QUEUE_PAGE_SIZE = 10;

/**
 * /safety «ایمنی و محتوای بالینی» (nbl_Admin_Safety): 7-day urgent-signal counts (aggregates only, no user ids), the
 * unified clinical review queue with claim / release and deep links to each source screen, and the urgent-path
 * health checks (admin-api.md §24).
 */
export function SafetyScreen() {
  const t = useTranslations('safety');
  const q = useSafety();

  return (
    <div className="flex flex-col gap-4">
      <div className="page-head">
        <div className="flex min-w-0 flex-col gap-1">
          <h1 className="page-title">{t('title')}</h1>
          <p className="text-ink-3 text-sm">{t('subtitle')}</p>
        </div>
      </div>
      {q.error && !q.data ? (
        <Panel>
          <ErrorState error={q.error} onRetry={() => q.refetch()} />
        </Panel>
      ) : (
        <>
          <Signals data={q.data} />
          <div className="grid gap-4 lg:grid-cols-[1.3fr_1fr]">
            <Queue data={q.data} loading={q.isPending} />
            <Checks data={q.data} />
          </div>
          <SelfTestRuns />
        </>
      )}
    </div>
  );
}

function Signals({ data }: { data: SafetyData | undefined }) {
  const t = useTranslations('safety');
  const locale = useLocale();
  const byKey = new Map(data?.signals.map((s) => [s.key, s]));
  return (
    <Panel bodyClassName="">
      <dl className="ledger m-0">
        {SIGNALS.map((key) => {
          const s = byKey.get(key);
          return (
            <div key={key} className="ledger-cell" data-alert={s ? s.count > 0 : undefined}>
              <dt className="text-[13px] font-semibold text-ink-3">{t(`signals.${key}`)}</dt>
              <dd className="m-0 mt-1">
                {s ? (
                  <span className="ledger-value">{formatNumber(s.count, locale)}</span>
                ) : (
                  <Skeleton className="h-8 w-16" />
                )}
                <span className="mt-0.5 block text-xs text-ink-3">
                  {s && data ? (
                    <>
                      {t('window', { days: formatNumber(data.window_days, locale) })}{' '}
                      <span dir="ltr" className={cn('font-semibold', s.delta > 0 && 'text-danger-deep')}>
                        {`${s.delta > 0 ? '+' : s.delta < 0 ? '−' : ''}${formatNumber(Math.abs(s.delta), locale)}`}
                      </span>
                    </>
                  ) : (
                    ' '
                  )}
                </span>
              </dd>
            </div>
          );
        })}
      </dl>
    </Panel>
  );
}

function Queue({ data, loading }: { data: SafetyData | undefined; loading: boolean }) {
  const t = useTranslations('safety');
  const locale = useLocale();
  const me = useCurrentAdmin();
  const canClaim = useCan('clinical', 'manage');
  const claim = useClaim();
  const notifyError = useNotifyError();
  const [source, setSource] = useState<Source | ''>('');
  const [page, setPage] = useState(1);
  const filtered = (data?.queue.items ?? []).filter((i) => source === '' || i.source === source);
  const lastPage = Math.max(1, Math.ceil(filtered.length / QUEUE_PAGE_SIZE));
  const current = Math.min(page, lastPage);
  const items = filtered.slice((current - 1) * QUEUE_PAGE_SIZE, current * QUEUE_PAGE_SIZE);
  const num = (n: number) => formatNumber(n, locale);

  const act = (item: ReviewItem, action: 'claim' | 'release') =>
    claim.mutate(
      { source: item.source, id: item.id, action },
      { onSuccess: () => toast.success(t(action === 'claim' ? 'claimed' : 'released')), onError: notifyError },
    );

  const status = (i: ReviewItem) =>
    i.status === 'partially_approved' ? (
      <Badge tone="amber">{t('approvalsOf', { n: num(i.approvals), of: num(i.required_approvals) })}</Badge>
    ) : (
      <Badge tone={i.status === 'changed' ? 'data' : 'amber'}>{t(`status.${i.status}`)}</Badge>
    );

  const columns: Column<ReviewItem>[] = [
    {
      key: 'item',
      header: t('item'),
      cell: (i) => (
        <span className="flex flex-col gap-0.5">
          <Link href={sourceHref(i)} className="font-semibold hover:underline">
            {i.source === 'catalog' ? t('catalogGroup', { group: i.title, n: num(i.items) }) : i.title}
          </Link>
          {i.source === 'threshold' || i.source === 'assistant' ? (
            <span className="text-ink-3 font-mono text-xs" dir="ltr">
              {i.ref}
            </span>
          ) : null}
        </span>
      ),
    },
    { key: 'type', header: t('type'), cell: (i) => t(`sources.${i.source}`) },
    { key: 'by', header: t('submitter'), cell: (i) => i.submitted_by ?? <span className="text-ink-3">—</span> },
    {
      key: 'age',
      header: t('age'),
      cell: (i) => {
        const a = ageParts(i.age_hours);
        const label = t(a.unit === 'd' ? 'ageDays' : 'ageHours', { n: num(a.value) });
        return i.overdue ? <Badge tone="red">{label}</Badge> : <span className="whitespace-nowrap">{label}</span>;
      },
      className: 'whitespace-nowrap',
    },
    { key: 'status', header: t('statusCol'), cell: status },
    {
      key: 'claim',
      header: t('reviewer'),
      cell: (i) => {
        const mine = i.claimed_by?.id === me?.id;
        return (
          <span className="flex flex-wrap items-center gap-2">
            {i.claimed_by ? <span className="text-ink-2 text-xs">{i.claimed_by.name}</span> : null}
            {canClaim && !i.claimed_by ? (
              <Button size="sm" variant="ghost" className="min-h-11" disabled={claim.isPending} onClick={() => act(i, 'claim')}>
                {t('claim')}
              </Button>
            ) : null}
            {canClaim && i.claimed_by && (mine || me?.role === 'super') ? (
              <Button size="sm" variant="ghost" className="min-h-11" disabled={claim.isPending} onClick={() => act(i, 'release')}>
                {t('release')}
              </Button>
            ) : null}
            <Link href={sourceHref(i)} className="btn btn-sm btn-ghost min-h-11">
              {t('open')}
            </Link>
          </span>
        );
      },
      className: 'cell-actions',
    },
  ];

  return (
    <Panel
      title={t('queueTitle')}
      bodyClassName=""
      actions={data ? <Badge tone="amber">{t('pending', { n: num(data.queue.pending) })}</Badge> : null}
    >
      <div className="flex flex-wrap gap-2 p-3" role="group" aria-label={t('sourcesLabel')}>
        {(['', ...SOURCES] as const).map((s) => (
          <button
            key={s || 'all'}
            type="button"
            aria-pressed={source === s}
            onClick={() => {
              setSource(s);
              setPage(1);
            }}
            className={cn('btn btn-sm min-h-11', source === s ? 'btn-primary' : 'btn-ghost')}
          >
            {t(`sources.${s || 'all'}`)}
            {data ? ` · ${num(s === '' ? data.queue.pending : (data.queue.by_source[s] ?? 0))}` : ''}
          </button>
        ))}
      </div>
      {data ? (
        <p className="panel-note">{t('slaNote', { hours: num(data.queue.sla_hours), n: num(data.queue.overdue) })}</p>
      ) : null}
      <DataTable
        columns={columns}
        items={items}
        rowKey={(i) => `${i.source}-${i.id}`}
        loading={loading}
        emptyText={t('empty')}
        caption={t('queueTitle')}
      />
      {filtered.length > QUEUE_PAGE_SIZE ? (
        <Pagination
          meta={{ current_page: current, last_page: lastPage, per_page: QUEUE_PAGE_SIZE, total: filtered.length }}
          onPage={setPage}
        />
      ) : null}
    </Panel>
  );
}

function Checks({ data }: { data: SafetyData | undefined }) {
  const t = useTranslations('safety');
  const locale = useLocale();
  const c = data?.checks;
  const num = (n: number) => formatNumber(n, locale);
  const rows: { key: string; label: string; value: ReactNode }[] = c
    ? [
        {
          key: 'latency',
          label: t('checks.latency'),
          value:
            c.modal_latency_p95_ms == null ? (
              <Badge>{t('checks.notInstrumented')}</Badge>
            ) : (
              <Badge tone="data">{`p95: ${num(c.modal_latency_p95_ms)}ms`}</Badge>
            ),
        },
        {
          key: 'ctr',
          label: t('checks.ctr'),
          value:
            c.call_115_ctr == null ? (
              <Badge>{t('checks.notInstrumented')}</Badge>
            ) : (
              <Badge tone="data">{`${num(c.call_115_ctr)}٪`}</Badge>
            ),
        },
        {
          key: 'parity',
          label: t('checks.parity'),
          value: c.text_parity.ok ? (
            <Badge tone="data">{t('checks.yes')}</Badge>
          ) : (
            <Badge tone="red">{t('checks.unapproved', { n: num(c.text_parity.unapproved) })}</Badge>
          ),
        },
        {
          key: 'scenarios',
          label: t('checks.scenarios'),
          value: !c.scenarios ? (
            <Badge>{t('checks.notRun')}</Badge>
          ) : (
            <Badge tone={c.scenarios.failed.length ? 'red' : 'data'}>
              {`${c.scenarios.ran_at ? formatDateTime(c.scenarios.ran_at, locale) : ''} · `}
              <span dir="ltr">{`${num(c.scenarios.passed)}/${num(c.scenarios.total)}`}</span>
            </Badge>
          ),
        },
      ]
    : [];
  return (
    <Panel title={t('checksTitle')} bodyClassName="flex flex-col divide-y divide-line">
      {c
        ? rows.map((r) => (
            <div key={r.key} className="flex flex-wrap items-center justify-between gap-2 px-4 py-3 text-sm">
              <span className="text-ink-2">{r.label}</span>
              {r.value}
            </div>
          ))
        : Array.from({ length: 4 }, (_, i) => (
            <div key={i} className="px-4 py-3">
              <Skeleton className="w-full" />
            </div>
          ))}
      {c?.scenarios?.failed.length ? (
        <p className="text-danger-deep m-0 px-4 py-3 text-xs" dir="ltr">
          {c.scenarios.failed.join(', ')}
        </p>
      ) : null}
    </Panel>
  );
}

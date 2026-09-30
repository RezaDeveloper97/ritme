'use client';

import Link from 'next/link';
import { useLocale, useTranslations } from 'next-intl';
import { useRouter } from 'next/navigation';

import { formatDateTime, formatNumber, useListParams } from '@/shared/lib';
import { DataTable, Icon, LinkTabs, Pagination, Panel, type Column } from '@/shared/ui';

import { supportReportsApi, type SupportReportRow } from '../api/support-reports';
import { reporterName, StatusBadge } from './status';

const TABS = ['open', 'resolved', 'all'] as const;
const tabHref = (status: string) => (status === 'open' ? '/support-reports' : `/support-reports?status=${status}`);

/** /support-reports?status= — «گزارش مشکل» inbox, newest first (B-N1-12b). */
export function SupportReportsScreen() {
  const t = useTranslations('supportReports');
  const locale = useLocale();
  const router = useRouter();
  const list = useListParams({ status: 'open' });
  const status = list.params.filters.status ?? 'open';
  const query = supportReportsApi.useList(list.query);
  const counts = query.data?.counts;
  const n = (v: number) => formatNumber(v, locale);

  const tabLabel = (tab: (typeof TABS)[number]) => {
    const label = t(`tabs.${tab}`);
    if (!counts || tab === 'all') return label;
    return `${label} (${n(counts[tab])})`;
  };

  const columns: Column<SupportReportRow>[] = [
    { key: 'id', header: '#', cell: (r) => n(r.id), className: 'cell-num w-14 text-muted' },
    {
      key: 'user',
      header: t('reporter'),
      cell: (r) => (
        <span className="flex flex-col items-start">
          <span className="font-semibold" dir="auto">
            {reporterName(r.user)}
          </span>
          {r.user.name && r.user.mobile ? (
            <span dir="ltr" className="text-xs text-muted">
              {r.user.mobile}
            </span>
          ) : null}
        </span>
      ),
    },
    {
      key: 'preview',
      header: t('message'),
      cell: (r) => (
        <span className="flex items-start gap-1.5">
          {r.has_screenshot ? (
            <span className="mt-0.5 shrink-0 text-data" title={t('hasScreenshot')}>
              <Icon name="image" size={15} />
              <span className="sr-only">{t('hasScreenshot')}</span>
            </span>
          ) : null}
          <span className="line-clamp-2">{r.preview}</span>
        </span>
      ),
      className: 'cell-wrap',
    },
    {
      key: 'version',
      header: t('appVersion'),
      cell: (r) => (r.app_version ? <span dir="ltr">{r.app_version}</span> : '—'),
      className: 'cell-num text-ink-3',
    },
    { key: 'status', header: t('statusLabel'), cell: (r) => <StatusBadge status={r.status} /> },
    {
      key: 'date',
      header: t('sentAt'),
      cell: (r) => formatDateTime(r.created_at, locale),
      className: 'cell-num whitespace-nowrap text-ink-3',
    },
    {
      key: 'open',
      header: <span className="sr-only">{t('actions')}</span>,
      cell: (r) => (
        <Link href={`/support-reports/${r.id}`} className="btn btn-sm" onClick={(e) => e.stopPropagation()}>
          {t('view')}
        </Link>
      ),
      className: 'cell-actions',
    },
  ];

  return (
    <div className="flex flex-col gap-4">
      <LinkTabs
        label={t('title')}
        items={TABS.map((tab) => ({ key: tab, href: tabHref(tab), label: tabLabel(tab), active: tab === status }))}
      />
      <Panel title={t('title')} bodyClassName="">
        <p className="panel-note">{t('intro')}</p>
        <DataTable
          columns={columns}
          items={query.data?.items}
          rowKey={(r) => r.id}
          loading={query.isPending}
          error={query.error}
          onRetry={() => query.refetch()}
          onRowClick={(r) => router.push(`/support-reports/${r.id}`)}
          emptyText={status === 'open' ? t('emptyOpen') : t('empty')}
          caption={t('title')}
        />
        <Pagination meta={query.data?.meta} onPage={list.setPage} />
      </Panel>
    </div>
  );
}

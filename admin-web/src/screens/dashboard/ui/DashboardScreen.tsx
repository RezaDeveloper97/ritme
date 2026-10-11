'use client';

import { useLocale, useTranslations } from 'next-intl';

import { useCurrentAdmin } from '@/features/auth';
import { formatDateTime, formatNumber } from '@/shared/lib';
import { Badge, DataTable, Panel, Skeleton, type Column } from '@/shared/ui';

import { useDashboard, type RecentUser } from '../api/dashboard';
import { Overview } from './OverviewPanels';

/**
 * Reference implementation for T-M2-23 screens: one query hook in the screen's
 * `api/` segment (zod-validated), shared/ui building blocks, every string via
 * next-intl, numbers and dates via shared/lib.
 */
export function DashboardScreen() {
  const t = useTranslations('dashboard');
  const tCommon = useTranslations('common');
  const admin = useCurrentAdmin();
  const locale = useLocale();
  const query = useDashboard();
  const n = (value: number) => formatNumber(value, locale);

  const columns: Column<RecentUser>[] = [
    { key: 'id', header: t('colId'), cell: (u) => n(u.id), className: 'cell-num w-16 text-muted' },
    { key: 'name', header: t('colName'), cell: (u) => u.name || '—' },
    {
      key: 'mobile',
      header: t('colMobile'),
      cell: (u) => (u.mobile ? <span dir="ltr">{u.mobile}</span> : '—'),
      className: 'cell-num',
    },
    {
      key: 'status',
      header: t('colStatus'),
      cell: (u) =>
        u.is_blocked ? <Badge tone="red">{tCommon('blocked')}</Badge> : <Badge tone="green">{tCommon('active')}</Badge>,
    },
    {
      key: 'joined',
      header: t('colJoined'),
      cell: (u) => formatDateTime(u.created_at, locale),
      className: 'cell-num whitespace-nowrap text-ink-3',
    },
  ];

  return (
    <div className="flex flex-col gap-5">
      <div className="flex flex-col gap-1">
        <p className="m-0 text-xl font-extrabold">{t('greeting', { name: admin?.name ?? '' })}</p>
        {query.data ? (
          <p className="m-0 flex items-center gap-2 text-ink-3">
            <span className="data-tick" aria-hidden="true" />
            {t('summary', { today: query.data.stats.users_new_today, week: query.data.stats.users_new_week })}
          </p>
        ) : (
          <Skeleton className="w-72" />
        )}
      </div>

      <Overview />

      <Panel title={t('recentUsers')} bodyClassName="">
        <DataTable
          columns={columns}
          items={query.data?.recent_users}
          rowKey={(u) => u.id}
          loading={query.isPending}
          error={query.error}
          onRetry={() => query.refetch()}
          emptyText={t('noUsers')}
          skeletonRows={8}
          caption={t('recentUsers')}
        />
      </Panel>
    </div>
  );
}

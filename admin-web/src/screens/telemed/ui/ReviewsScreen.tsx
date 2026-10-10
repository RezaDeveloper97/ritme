'use client';

import { useLocale, useTranslations } from 'next-intl';

import { formatDateTime, formatNumber, useListParams } from '@/shared/lib';
import { Badge, Button, DataTable, LinkTabs, PageHeader, Pagination, Panel, toast, useNotifyError, type Column } from '@/shared/ui';

import { reviewsApi, useReviewVisibility, type Review } from '../api/telemed';

const TABS = ['all', 'visible', 'hidden'] as const;

/** /telemed/reviews — hide or show a patient review; the doctor's rating is recomputed (admin-api.md §19). */
export function ReviewsScreen() {
  const t = useTranslations('telemed.reviews');
  const locale = useLocale();
  const notifyError = useNotifyError();
  const list = useListParams({ visibility: 'all' });
  const tab = list.params.filters.visibility ?? 'all';
  const query = reviewsApi.useList(list.query);
  const visibility = useReviewVisibility();
  const toggle = (r: Review) =>
    visibility.mutate({ id: r.id, visible: !r.is_visible }, { onSuccess: () => toast.success(t('saved')), onError: notifyError });

  const columns: Column<Review>[] = [
    { key: 'rating', header: t('rating'), cell: (r) => formatNumber(r.rating, locale), className: 'cell-num' },
    {
      key: 'body',
      header: t('body'),
      cell: (r) => (
        <span className="whitespace-pre-wrap break-words" dir="auto">
          {r.body || '—'}
        </span>
      ),
    },
    { key: 'doctor', header: t('doctor'), cell: (r) => `#${formatNumber(r.doctor_id, locale)}`, className: 'cell-num' },
    { key: 'date', header: t('date'), cell: (r) => formatDateTime(r.created_at, locale), className: 'cell-num whitespace-nowrap text-ink-3' },
    {
      key: 'state',
      header: t('state'),
      cell: (r) => <Badge tone={r.is_visible ? 'green' : 'neutral'}>{r.is_visible ? t('visible') : t('hidden')}</Badge>,
    },
    {
      key: 'actions',
      header: <span className="sr-only">{t('actions')}</span>,
      cell: (r) => (
        <div className="row-actions">
          <Button size="sm" loading={visibility.isPending && visibility.variables?.id === r.id} onClick={() => toggle(r)}>
            {r.is_visible ? t('hide') : t('show')}
          </Button>
        </div>
      ),
      className: 'cell-actions',
    },
  ];

  return (
    <div className="flex flex-col gap-4">
      <PageHeader title={t('title')} />
      <LinkTabs
        label={t('title')}
        items={TABS.map((s) => ({
          key: s,
          href: s === 'all' ? '/telemed/reviews' : `/telemed/reviews?visibility=${s}`,
          label: t(`tabs.${s}`),
          active: s === tab,
        }))}
      />
      <Panel bodyClassName="">
        <DataTable
          columns={columns}
          items={query.data?.items}
          rowKey={(r) => r.id}
          loading={query.isPending}
          error={query.error}
          onRetry={() => query.refetch()}
          emptyText={t('empty')}
          caption={t('title')}
        />
        <Pagination meta={query.data?.meta} onPage={list.setPage} />
      </Panel>
    </div>
  );
}

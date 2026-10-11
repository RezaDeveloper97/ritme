'use client';

import Link from 'next/link';
import { useLocale, useTranslations } from 'next-intl';

import { formatDateTime, formatNumber, useListParams } from '@/shared/lib';
import { Badge, Button, DataTable, PageHeader, Pagination, Panel, Select, toast, useNotifyError, type BadgeTone, type Column } from '@/shared/ui';

import { useCityServiceOptions } from '../api/city-services';
import { reportsApi, reviewsApi, type Report, type Review } from '../api/directory';

const REVIEW_TONE: Record<string, BadgeTone> = { pending: 'amber', published: 'green', hidden: 'neutral' };
const REPORT_TONE: Record<string, BadgeTone> = { open: 'amber', resolved: 'green', dismissed: 'neutral' };

function PlaceLink({ place }: { place: { id: number; name: string } }) {
  return (
    <Link href={`/city-services/${place.id}`} className="text-brand no-underline hover:underline">
      {place.name}
    </Link>
  );
}

/**
 * /city-services/reviews — review moderation (CB-DIR-05, capability content). A mother's review waits as pending;
 * publishing or hiding it recomputes the place's rating. Authors appear as a first name only.
 */
export function CityServiceReviewsScreen() {
  const t = useTranslations('cityServices.dir.reviews');
  const locale = useLocale();
  const notifyError = useNotifyError();
  const options = useCityServiceOptions();
  const list = useListParams({ status: 'pending' });
  const status = list.params.filters.status ?? 'pending';
  const query = reviewsApi.useList({ ...list.query, ...(status === 'all' ? {} : { status }) });

  const columns: Column<Review>[] = [
    { key: 'date', header: t('date'), cell: (r) => (r.created_at ? formatDateTime(r.created_at, locale) : '—'), className: 'cell-num whitespace-nowrap' },
    { key: 'place', header: t('place'), cell: (r) => <PlaceLink place={r.listing} /> },
    {
      key: 'rating',
      header: t('rating'),
      cell: (r) => (
        <span className="flex flex-col">
          <span aria-label={formatNumber(r.rating, locale)}>{'★'.repeat(r.rating)}</span>
          <span className="text-xs text-muted">
            {t('scores', {
              c: r.score_cleanliness == null ? '—' : formatNumber(r.score_cleanliness, locale),
              i: r.score_instructor == null ? '—' : formatNumber(r.score_instructor, locale),
              a: r.score_amenities == null ? '—' : formatNumber(r.score_amenities, locale),
            })}
          </span>
        </span>
      ),
    },
    { key: 'body', header: t('body'), cell: (r) => r.body ?? '—' },
    { key: 'author', header: t('author'), cell: (r) => r.author ?? '—' },
    { key: 'status', header: t('status'), cell: (r) => <Badge tone={REVIEW_TONE[r.status] ?? 'neutral'}>{t(`statusName.${r.status}` as 'statusName.pending')}</Badge> },
    { key: 'actions', header: <span className="sr-only">{t('status')}</span>, cell: (r) => <ReviewActions review={r} onError={notifyError} />, className: 'cell-actions' },
  ];

  return (
    <div className="flex flex-col gap-4">
      <PageHeader title={t('title')} backHref="/city-services" />
      <Panel bodyClassName="">
        <div className="filter-bar">
          <Select
            aria-label={t('status')}
            value={status}
            onChange={(e) => list.setFilter('status', e.target.value)}
            options={[{ value: 'all', label: t('allStatuses') }, ...(options.data?.review_statuses ?? []).map((s) => ({ value: s, label: t(`statusName.${s}` as 'statusName.pending') }))]}
          />
        </div>
        <DataTable columns={columns} items={query.data?.items} rowKey={(r) => r.id} loading={query.isPending} error={query.error} onRetry={() => query.refetch()} emptyText={t('empty')} caption={t('title')} />
        <Pagination meta={query.data?.meta} onPage={list.setPage} />
      </Panel>
    </div>
  );
}

function ReviewActions({ review, onError }: { review: Review; onError: (e: unknown) => void }) {
  const t = useTranslations('cityServices.dir.reviews');
  const save = reviewsApi.useSave(review.id);
  const to = (status: string) => save.mutate({ status }, { onSuccess: () => toast.success(t('moved')), onError });
  return (
    <div className="row-actions">
      {review.status !== 'published' ? (
        <Button size="sm" variant="primary" loading={save.isPending} onClick={() => to('published')}>
          {t('publish')}
        </Button>
      ) : null}
      {review.status !== 'hidden' ? (
        <Button size="sm" loading={save.isPending} onClick={() => to('hidden')}>
          {t('hide')}
        </Button>
      ) : null}
    </div>
  );
}

/** /city-services/reports — «گزارش اطلاعات نادرست» queue (CB-DIR-05). The reporter is never shown. */
export function CityServiceReportsScreen() {
  const t = useTranslations('cityServices.dir.reports');
  const locale = useLocale();
  const notifyError = useNotifyError();
  const options = useCityServiceOptions();
  const list = useListParams({ status: 'open' });
  const status = list.params.filters.status ?? 'open';
  const query = reportsApi.useList({ ...list.query, ...(status === 'all' ? {} : { status }) });

  const columns: Column<Report>[] = [
    { key: 'date', header: t('date'), cell: (r) => (r.created_at ? formatDateTime(r.created_at, locale) : '—'), className: 'cell-num whitespace-nowrap' },
    { key: 'place', header: t('place'), cell: (r) => <PlaceLink place={r.listing} /> },
    { key: 'reason', header: t('reason'), cell: (r) => t(`reasonName.${r.reason}` as 'reasonName.other') },
    { key: 'note', header: t('note'), cell: (r) => r.note ?? '—' },
    { key: 'status', header: t('status'), cell: (r) => <Badge tone={REPORT_TONE[r.status] ?? 'neutral'}>{t(`statusName.${r.status}` as 'statusName.open')}</Badge> },
    { key: 'actions', header: <span className="sr-only">{t('status')}</span>, cell: (r) => (r.status === 'open' ? <ReportActions report={r} onError={notifyError} /> : null), className: 'cell-actions' },
  ];

  return (
    <div className="flex flex-col gap-4">
      <PageHeader title={t('title')} backHref="/city-services" />
      <Panel bodyClassName="">
        <div className="filter-bar">
          <Select
            aria-label={t('status')}
            value={status}
            onChange={(e) => list.setFilter('status', e.target.value)}
            options={[{ value: 'all', label: t('allStatuses') }, ...(options.data?.report_statuses ?? []).map((s) => ({ value: s, label: t(`statusName.${s}` as 'statusName.open') }))]}
          />
        </div>
        <DataTable columns={columns} items={query.data?.items} rowKey={(r) => r.id} loading={query.isPending} error={query.error} onRetry={() => query.refetch()} emptyText={t('empty')} caption={t('title')} />
        <Pagination meta={query.data?.meta} onPage={list.setPage} />
      </Panel>
    </div>
  );
}

function ReportActions({ report, onError }: { report: Report; onError: (e: unknown) => void }) {
  const t = useTranslations('cityServices.dir.reports');
  const save = reportsApi.useSave(report.id);
  const to = (status: string) => save.mutate({ status }, { onSuccess: () => toast.success(t('moved')), onError });
  return (
    <div className="row-actions">
      <Button size="sm" variant="primary" loading={save.isPending} onClick={() => to('resolved')}>
        {t('resolve')}
      </Button>
      <Button size="sm" loading={save.isPending} onClick={() => to('dismissed')}>
        {t('dismiss')}
      </Button>
    </div>
  );
}

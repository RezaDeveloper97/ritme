'use client';

import Link from 'next/link';
import { useLocale, useTranslations } from 'next-intl';

import { formatDateTime, formatNumber, toLocaleDigits, useListParams } from '@/shared/lib';
import { DataTable, LinkTabs, Pagination, Panel, type Column } from '@/shared/ui';

import { reviewsApi, type ReviewItem } from '../api/learning';
import { formatDuration, REVIEW_TABS, reviewHref, type ReviewTab } from '../lib/review';
import { LessonActions, PublicationBadge, ReviewBadge } from './parts';

/** /learning/reviews?state= — content review queue (feeds the B-N9-10 safety queue). */
export function LearningReviewsScreen() {
  const t = useTranslations('learning.reviews');
  const tk = useTranslations('learning.lessonKind');
  const locale = useLocale();
  const list = useListParams({ state: 'open' });
  const state = (list.params.filters.state ?? 'open') as ReviewTab;
  const query = reviewsApi.useList(list.query);
  const counts = query.data?.counts;
  const n = (v: number) => formatNumber(v, locale);

  const columns: Column<ReviewItem>[] = [
    {
      key: 'lesson',
      header: t('lesson'),
      cell: (l) => (
        <span className="flex flex-col items-start gap-0.5">
          <span className="font-semibold" dir="auto">
            {l.title}
          </span>
          <span className="flex flex-wrap items-center gap-2 text-xs text-ink-3">
            <span>{tk.has(l.kind as 'video') ? tk(l.kind as 'video') : l.kind}</span>
            {formatDuration(l.duration_seconds) ? <span dir="ltr">{toLocaleDigits(formatDuration(l.duration_seconds) ?? '', locale)}</span> : null}
          </span>
          {l.review.note ? (
            <span className="text-xs text-ink-3" dir="auto">
              {t('note', { note: l.review.note })}
            </span>
          ) : null}
        </span>
      ),
      className: 'cell-wrap',
    },
    {
      key: 'course',
      header: t('course'),
      cell: (l) => (
        <span className="flex flex-col items-start">
          <Link href={`/learning/courses/${l.course.id}`} className="text-brand no-underline hover:underline" dir="auto">
            {l.course.title}
          </Link>
          <span className="text-xs text-ink-3" dir="auto">
            {l.instructor.display_name}
          </span>
        </span>
      ),
      className: 'cell-wrap',
    },
    {
      key: 'state',
      header: t('state'),
      cell: (l) => (
        <span className="flex flex-col items-start gap-1">
          <ReviewBadge state={l.review.state} />
          <PublicationBadge status={l.status} />
        </span>
      ),
    },
    {
      key: 'updated',
      header: t('updatedAt'),
      cell: (l) => formatDateTime(l.updated_at, locale),
      className: 'cell-num whitespace-nowrap text-ink-3',
    },
    {
      key: 'actions',
      header: <span className="sr-only">{t('actions')}</span>,
      cell: (l) => <LessonActions lesson={l} />,
      className: 'cell-actions',
    },
  ];

  return (
    <div className="flex flex-col gap-4">
      <LinkTabs
        label={t('title')}
        items={REVIEW_TABS.map((tab) => ({
          key: tab,
          href: reviewHref(tab),
          label: counts && tab !== 'all' ? `${t(`tabs.${tab}`)} (${n(counts[tab] ?? 0)})` : t(`tabs.${tab}`),
          active: tab === state,
        }))}
      />
      <Panel title={t('title')} bodyClassName="">
        <p className="panel-note">{t('intro')}</p>
        <DataTable
          columns={columns}
          items={query.data?.items}
          rowKey={(l) => l.id}
          loading={query.isPending}
          error={query.error}
          onRetry={() => query.refetch()}
          emptyText={state === 'open' ? t('emptyOpen') : t('empty')}
          caption={t('title')}
        />
        <Pagination meta={query.data?.meta} onPage={list.setPage} />
      </Panel>
    </div>
  );
}

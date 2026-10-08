'use client';

import Link from 'next/link';
import { useLocale, useTranslations } from 'next-intl';
import { useRouter } from 'next/navigation';

import { formatDate, formatNumber, useListParams } from '@/shared/lib';
import { DataTable, Pagination, Panel, SearchInput, Select, Skeleton, type Column } from '@/shared/ui';

import { coursesApi, useLearningStats, type CourseRow } from '../api/learning';
import { COURSE_STATUSES } from '../lib/review';
import { Completion, PublicationBadge } from './parts';

/** /learning/courses — usage stats band + every course with counts and completion (B-N8-08). */
export function LearningCoursesScreen() {
  const t = useTranslations('learning.courses');
  const locale = useLocale();
  const router = useRouter();
  const list = useListParams({ status: 'all', instructor_id: '' });
  const query = coursesApi.useList(list.query);
  const counts = query.data?.counts;
  const n = (v: number) => formatNumber(v, locale);
  const filtered = Boolean(list.params.q) || list.params.filters.status !== 'all' || Boolean(list.params.filters.instructor_id);

  const columns: Column<CourseRow>[] = [
    {
      key: 'title',
      header: t('course'),
      cell: (c) => (
        <span className="flex flex-col items-start">
          <span className="font-semibold" dir="auto">
            {c.title}
          </span>
          <span className="text-xs text-ink-3">{t(`kind.${c.kind === 'standalone' ? 'standalone' : 'course'}`)}</span>
        </span>
      ),
      className: 'cell-wrap',
    },
    {
      key: 'instructor',
      header: t('instructor'),
      cell: (c) => (
        <span dir="auto" className="text-ink-3">
          {c.instructor.display_name}
        </span>
      ),
    },
    { key: 'status', header: t('status'), cell: (c) => <PublicationBadge status={c.status} /> },
    {
      key: 'lessons',
      header: t('lessons'),
      cell: (c) => t('lessonsCount', { published: n(c.published_lessons), total: n(c.lessons_count) }),
      className: 'whitespace-nowrap text-ink-3',
    },
    {
      key: 'review',
      header: t('review'),
      cell: (c) =>
        c.review_open > 0 ? (
          <span className={c.flagged_count > 0 ? 'font-semibold text-red' : 'font-semibold'}>
            {t('reviewOpen', { count: n(c.review_open) })}
          </span>
        ) : (
          <span className="text-muted">—</span>
        ),
      className: 'whitespace-nowrap',
    },
    { key: 'students', header: t('students'), cell: (c) => n(c.students_count), className: 'cell-num' },
    {
      key: 'completion',
      header: t('completion'),
      cell: (c) => <Completion percent={c.completion_percent} students={c.students_count} />,
    },
    {
      key: 'date',
      header: t('createdAt'),
      cell: (c) => formatDate(c.created_at, locale),
      className: 'cell-num whitespace-nowrap text-ink-3',
    },
  ];

  return (
    <div className="flex flex-col gap-4">
      <StatsBand />
      <Panel title={t('title')} bodyClassName="">
        <div className="filter-bar">
          <SearchInput value={list.params.q} onChange={list.setSearch} placeholder={t('searchPlaceholder')} />
          <Select
            label={t('status')}
            className="w-44"
            value={list.params.filters.status}
            onChange={(e) => list.setFilter('status', e.target.value)}
            options={COURSE_STATUSES.map((s) => ({
              value: s,
              label: counts ? `${t(`filter.${s}`)} (${n(counts[s] ?? 0)})` : t(`filter.${s}`),
            }))}
          />
          {list.params.filters.instructor_id ? (
            <Link href="/learning/courses" className="btn btn-sm self-end">
              {t('clearInstructor')}
            </Link>
          ) : null}
        </div>
        <p className="panel-note">{t('intro')}</p>
        <DataTable
          columns={columns}
          items={query.data?.items}
          rowKey={(c) => c.id}
          loading={query.isPending}
          error={query.error}
          onRetry={() => query.refetch()}
          onRowClick={(c) => router.push(`/learning/courses/${c.id}`)}
          emptyText={filtered ? t('emptyFiltered') : t('empty')}
          caption={t('title')}
        />
        <Pagination meta={query.data?.meta} onPage={list.setPage} />
      </Panel>
    </div>
  );
}

/** Platform usage: instructors, courses, students, completion, review queue. */
function StatsBand() {
  const t = useTranslations('learning.stats');
  const locale = useLocale();
  const { data } = useLearningStats();
  const n = (v: number) => formatNumber(v, locale);
  const cells = data
    ? [
        { key: 'instructors', value: n(data.instructors.approved ?? 0), sub: t('instructorsSub', { pending: n(data.instructors.pending ?? 0) }), alert: (data.instructors.pending ?? 0) > 0 },
        { key: 'courses', value: n(data.courses.published ?? 0), sub: t('coursesSub', { draft: n(data.courses.draft ?? 0) }), alert: false },
        { key: 'grants', value: n(data.grants.active ?? 0), sub: t('grantsSub', { pending: n(data.grants.pending ?? 0), recent: n(data.grants.new_30d ?? 0) }), alert: false },
        { key: 'students', value: n(data.students.with_access), sub: t('studentsSub', { active: n(data.students.active_30d), days: n(data.active_days) }), alert: false },
        { key: 'completion', value: t('percent', { value: n(data.completion.percent) }), sub: t('completionSub', { completed: n(data.completion.completed), total: n(data.completion.enrolments) }), alert: false },
        { key: 'review', value: n(data.review.open ?? 0), sub: t('reviewSub', { flagged: n(data.review.flagged ?? 0) }), alert: (data.review.flagged ?? 0) > 0 },
      ]
    : (['instructors', 'courses', 'grants', 'students', 'completion', 'review'] as const).map((key) => ({ key, value: null, sub: ' ', alert: false }));
  return (
    <dl className="ledger m-0">
      {cells.map((c) => (
        <div key={c.key} className="ledger-cell" data-alert={c.alert ? 'true' : undefined}>
          <dt className="text-[13px] font-semibold text-ink-3">{t(c.key as 'instructors')}</dt>
          <dd className="m-0 mt-1">
            {c.value === null ? <Skeleton className="h-8 w-20" /> : <span className="ledger-value">{c.value}</span>}
            <span className="mt-0.5 block text-xs text-muted">{c.sub}</span>
          </dd>
        </div>
      ))}
    </dl>
  );
}

'use client';

import { useLocale, useTranslations } from 'next-intl';

import { formatDate, formatNumber, toLocaleDigits } from '@/shared/lib';
import { ErrorState, PageHeader, Panel, Skeleton } from '@/shared/ui';

import { coursesApi, type Course } from '../api/learning';
import { formatDuration } from '../lib/review';
import { InstructorBadge, LessonActions, PublicationBadge, ReviewBadge } from './parts';

/** /learning/courses/:id — outline with every lesson's review state, grants and usage (B-N8-08). */
export function LearningCourseDetailScreen({ id }: { id: number }) {
  const t = useTranslations('learning.course');
  const query = coursesApi.useDetail(id);
  if (query.error && !query.data) {
    return (
      <div className="flex flex-col gap-4">
        <PageHeader title={t('title')} backHref="/learning/courses" backLabel={t('back')} />
        <Panel>
          <ErrorState error={query.error} onRetry={() => query.refetch()} />
        </Panel>
      </div>
    );
  }
  if (!query.data) {
    return (
      <div className="flex flex-col gap-4" aria-busy="true">
        <Skeleton className="h-8 w-60" />
        <Skeleton className="h-28 w-full" />
        <Skeleton className="h-64 w-full" />
      </div>
    );
  }
  return <CourseDetail course={query.data.course} />;
}

function CourseDetail({ course }: { course: Course }) {
  const t = useTranslations('learning.course');
  const tk = useTranslations('learning.lessonKind');
  const locale = useLocale();
  const n = (v: number) => formatNumber(v, locale);
  const chapterTitle = new Map(course.chapters.map((c) => [c.id, c.title]));

  return (
    <div className="flex flex-col gap-5">
      <PageHeader
        title={<span dir="auto">{course.title}</span>}
        backHref="/learning/courses"
        backLabel={t('back')}
        meta={
          <>
            <PublicationBadge status={course.status} />
            <span dir="auto">{course.instructor.display_name}</span>
            <InstructorBadge status={course.instructor.status} />
            <span>{t('createdOn', { date: formatDate(course.created_at, locale) })}</span>
          </>
        }
      />

      <dl className="ledger m-0">
        {[
          { key: 'students', value: n(course.usage.students_count) },
          { key: 'completion', value: t('percent', { value: n(course.usage.completion_percent) }) },
          { key: 'completed', value: n(course.usage.completed_count) },
          { key: 'grantsActive', value: n(course.grants.active) },
          { key: 'grantsPending', value: n(course.grants.pending) },
          { key: 'grantsExpired', value: n(course.grants.expired) },
        ].map((c) => (
          <div key={c.key} className="ledger-cell">
            <dt className="text-[13px] font-semibold text-ink-3">{t(`stat.${c.key}` as 'stat.students')}</dt>
            <dd className="m-0 mt-1">
              <span className="ledger-value">{c.value}</span>
            </dd>
          </div>
        ))}
      </dl>

      {course.description ? (
        <Panel title={t('description')}>
          <p className="m-0 whitespace-pre-wrap break-words leading-7" dir="auto">
            {course.description}
          </p>
        </Panel>
      ) : null}

      <Panel title={t('lessons', { count: n(course.lessons.length) })} bodyClassName="">
        <p className="panel-note">{t('lessonsNote')}</p>
        {course.lessons.length === 0 ? (
          <p className="m-0 p-4 text-ink-3">{t('noLessons')}</p>
        ) : (
          <ul className="m-0 list-none p-0">
            {course.lessons.map((l) => (
              <li key={l.id} className="flex flex-wrap items-center justify-between gap-3 border-b border-line px-4 py-3 last:border-b-0">
                <div className="flex min-w-0 flex-col gap-1">
                  <span className="font-semibold" dir="auto">
                    {l.title}
                  </span>
                  <span className="flex flex-wrap items-center gap-2 text-xs text-ink-3">
                    <span>{tk.has(l.kind as 'video') ? tk(l.kind as 'video') : l.kind}</span>
                    {formatDuration(l.duration_seconds) ? <span dir="ltr">{toLocaleDigits(formatDuration(l.duration_seconds) ?? '', locale)}</span> : null}
                    {l.chapter_id && chapterTitle.get(l.chapter_id) ? <span dir="auto">{chapterTitle.get(l.chapter_id)}</span> : null}
                    <PublicationBadge status={l.status} />
                    <ReviewBadge state={l.review.state} />
                  </span>
                  {l.review.note ? (
                    <span className="text-xs text-ink-3" dir="auto">
                      {t('reviewNote', { note: l.review.note })}
                    </span>
                  ) : null}
                </div>
                <LessonActions lesson={l} />
              </li>
            ))}
          </ul>
        )}
      </Panel>
    </div>
  );
}

'use client';

import { clsx } from 'clsx';
import { useLocale, useTranslations } from 'next-intl';

import { pilot } from '@/shared/pilot';
import { Link, type Locale } from '@/shared/i18n';
import { formatDayMonth, formatNumber } from '@/shared/lib/date';
import { Icon, type Tone } from '@/shared/ui';

import { KIND_LOOK, minutesOf } from '../model/format';
import type { CourseAccess, LessonKind, StudentCourse, StudentLesson } from '../model/types';

const toneClass = (tone: Tone) => `nb-tone-${tone}`;

/** Square gradient tile with the kind's icon (video / audio / PDF). */
export function KindThumb({ kind, locked, size = 'md' }: { kind: LessonKind | null; locked?: boolean; size?: 'sm' | 'md' }) {
  const look = KIND_LOOK[kind ?? 'video'];
  return (
    <span className={clsx('lrn-thumb', `is-${size}`, toneClass(look.tone))} aria-hidden>
      <Icon name={look.icon} size={size === 'sm' ? 20 : 24} />
      {locked ? (
        <span className="lrn-thumb-lock">
          <Icon name="lock" size={12} />
        </span>
      ) : null}
    </span>
  );
}

/** «۱۲ درس · ویدیو» / «صوتی · ۱۸ دقیقه» / «PDF · ۱۴ صفحه». */
export function useCourseMeta() {
  const t = useTranslations('learn');
  const locale = useLocale() as Locale;
  return (c: Pick<StudentCourse, 'lessonsCount' | 'primaryKind' | 'durationSeconds' | 'pageCount'>) => {
    const kind = c.primaryKind ?? 'video';
    if (c.lessonsCount <= 1 && kind === 'pdf') return t('meta.pdfPages', { pages: formatNumber(c.pageCount, locale) });
    if (c.lessonsCount <= 1 && kind === 'audio') return t('meta.audioMinutes', { minutes: formatNumber(minutesOf(c.durationSeconds), locale) });
    return t('meta.lessonsKind', { count: formatNumber(c.lessonsCount, locale), kind: t(`kinds.${kind}`) });
  };
}

/** «ویدیو · ۶ دقیقه · ۷ دقیقه مانده» of a lesson row. */
export function useLessonMeta() {
  const t = useTranslations('learn');
  const locale = useLocale() as Locale;
  return (l: Pick<StudentLesson, 'kind' | 'durationSeconds' | 'pageCount' | 'progress'>) => {
    const parts = [t(`kinds.${l.kind}`)];
    if (l.kind === 'pdf') {
      if (l.pageCount) parts.push(t('meta.pages', { pages: formatNumber(l.pageCount, locale) }));
    } else if (l.durationSeconds) {
      parts.push(t('meta.minutes', { minutes: formatNumber(minutesOf(l.durationSeconds), locale) }));
      const pos = l.progress?.positionSeconds ?? 0;
      if (pos > 0 && !l.progress?.completed) {
        parts.push(t('meta.minutesLeft', { minutes: formatNumber(minutesOf(l.durationSeconds - pos), locale) }));
      }
    }
    return parts.join(' · ');
  };
}

/** «۸۵ روز دسترسی» / «نامحدود» / «تا ۳۰ دی» / «دسترسی تمام شد». */
export function useAccessLabel() {
  const t = useTranslations('learn');
  const locale = useLocale() as Locale;
  return (a: CourseAccess, style: 'days' | 'until' = 'days') => {
    if (a.expired) return t('access.expired');
    if (a.unlimited || !a.expiresAt) return t('access.unlimited');
    if (style === 'until') return t('access.until', { date: formatDayMonth(new Date(a.expiresAt), locale) });
    return t('access.daysLeft', { days: formatNumber(a.daysLeft ?? 0, locale) });
  };
}

/** One card of «دوره‌های من» (Learn_Hub). Expired courses stay visible, dimmed and not tappable. */
export function CourseCard({ course }: { course: StudentCourse }) {
  const t = useTranslations('learn');
  const locale = useLocale() as Locale;
  const meta = useCourseMeta();
  const access = useAccessLabel();
  const expired = course.access.expired;
  const pct = Math.max(0, Math.min(100, course.progressPercent));
  const tone = KIND_LOOK[course.primaryKind ?? 'video'].tone;
  const body = (
    <>
      <KindThumb kind={course.primaryKind} locked={expired} />
      <span className="lrn-cc-main">
        <span className="lrn-cc-title">{course.title}</span>
        <span className="lrn-cc-meta">{meta(course)}</span>
        <span className={clsx('lrn-bar', toneClass(tone))} aria-hidden>
          <span style={{ inlineSize: `${pct}%` }} />
        </span>
        <span className="lrn-cc-foot">
          <span>{t('hub.watched', { percent: formatNumber(pct, locale) })}</span>
          <span className={clsx(expired && 'is-expired')}>{access(course.access)}</span>
        </span>
      </span>
    </>
  );
  if (expired) {
    return <div className="lrn-cc is-expired">{body}</div>;
  }
  return (
    <Link href={`/learn/${course.id}`} className="lrn-cc" {...pilot(`learn.course:${course.id}`)}>
      {body}
    </Link>
  );
}

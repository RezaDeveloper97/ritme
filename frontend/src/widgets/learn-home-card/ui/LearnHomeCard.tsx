'use client';

import { useLocale, useTranslations } from 'next-intl';

import { KindThumb, minutesOf, newCourses, useCourseMeta, useMyCourses, useOfflinePurge, type ContinueCard, type StudentCourse } from '@/entities/course';
import { Link, useRouter, type Locale } from '@/shared/i18n';
import { formatDayMonth, formatNumber } from '@/shared/lib/date';
import { Card, Icon, SectionTitle } from '@/shared/ui';
import '@/shared/ui/styles/lrn.css';

/**
 * Home learning card (Option A — `nbl_/nbd_LearnEntry_HomeContinue` and
 * `LearnEntry_HomeNew`): «ادامه یادگیری» when a lesson is half-way, otherwise
 * «جدید · از <مدرس>» for a course granted in the last two weeks. Renders
 * nothing for everyone without a course (or while loading / on error — the home
 * feed must not jump or nag).
 */
export function LearnHomeCard() {
  const q = useMyCourses();
  // B-N8-04: on app start (home), drop offline downloads whose access ended.
  useOfflinePurge();
  if (!q.data) return null;
  if (q.data.continue) return <ContinueVariant item={q.data.continue} courses={q.data.courses} />;
  const fresh = newCourses(q.data.courses);
  if (fresh[0]) return <NewVariant course={fresh[0]} more={fresh.length - 1} />;
  return null;
}

function ContinueVariant({ item, courses }: { item: ContinueCard; courses: readonly StudentCourse[] }) {
  const t = useTranslations('learn');
  const locale = useLocale() as Locale;
  const router = useRouter();
  const course = courses.find((c) => c.id === item.course.id);
  const left = course ? Math.max(0, course.lessonsCount - course.completedLessons) : null;
  const href = `/learn/${item.course.id}/${item.lesson.id}`;
  const remaining = minutesOf(item.remainingSeconds);
  const pct = Math.max(0, Math.min(100, item.percent));
  return (
    <section className="lrn-home" aria-labelledby="lrn-home-title">
      <SectionTitle id="lrn-home-title" title={t('home.continueTitle')} actionLabel={t('home.allCourses')} onAction={() => router.push('/learn')} />
      <Card padding="md" className="lrn-home-card">
        <p className="lrn-home-sub">
          {left !== null
            ? t('home.courseLeft', { title: item.course.title, count: formatNumber(left, locale) })
            : item.course.title}
        </p>
        <div className="lrn-home-row">
          <KindThumb kind={item.lesson.kind} size="sm" />
          <div className="lrn-home-main">
            <span className="lrn-home-lesson">{item.lesson.title}</span>
            {remaining ? <span className="lrn-home-meta">{t('meta.minutesLeft', { minutes: formatNumber(remaining, locale) })}</span> : null}
          </div>
        </div>
        <span className="lrn-bar nb-tone-period" aria-hidden>
          <span style={{ inlineSize: `${pct}%` }} />
        </span>
        <div className="lrn-home-actions">
          <Link href={href} className="lrn-pill is-primary">
            <Icon name="play" size={16} />
            {t(`home.open.${item.lesson.kind}`)}
          </Link>
          {item.lesson.kind === 'video' ? (
            <Link href={`${href}?audio=1`} className="lrn-pill">
              <Icon name="music" size={16} />
              {t('home.audioOnly')}
            </Link>
          ) : null}
        </div>
      </Card>
    </section>
  );
}

function NewVariant({ course, more }: { course: StudentCourse; more: number }) {
  const t = useTranslations('learn');
  const locale = useLocale() as Locale;
  const meta = useCourseMeta();
  const access = course.access;
  const until =
    !access.unlimited && access.expiresAt ? t('home.accessUntil', { date: formatDayMonth(new Date(access.expiresAt), locale) }) : t('access.unlimited');
  return (
    <section className="lrn-home" aria-label={t('home.newLabel')}>
      <Link href={`/learn/${course.id}`} className="lrn-home-new">
        <span className="lrn-home-new-art" aria-hidden>
          <span className="lrn-home-new-chip">{t('home.newFrom', { name: course.instructor.name })}</span>
          <span className="lrn-play-disc">
            <Icon name="play" size={22} fill="currentColor" />
          </span>
          <span className="lrn-home-new-title">{course.title}</span>
        </span>
        <span className="lrn-home-new-foot">
          <span className="lrn-home-meta">
            {meta(course)} · {until}
          </span>
          {more > 0 ? <span className="lrn-home-more">{t('home.more', { count: formatNumber(more, locale) })}</span> : null}
        </span>
      </Link>
    </section>
  );
}

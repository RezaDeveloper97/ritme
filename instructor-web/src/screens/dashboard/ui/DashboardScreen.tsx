'use client';

import { useTranslations } from 'next-intl';
import Link from 'next/link';

import {
  CourseRow,
  faNum,
  hoursOf,
  useDashboard,
  type Activity,
} from '@/entities/course';
import type { Instructor } from '@/entities/instructor';
import { Icon, QueryError, Skeleton, type IconName } from '@/shared/ui';

import { relativeTime } from '../model/relative';

/** / — nbl_Ins_Dashboard: stat tiles, quick actions, my courses, recent activity (GET /dashboard). */
export function DashboardScreen({ instructor }: { instructor: Instructor }) {
  const t = useTranslations('dashboard');
  const tc = useTranslations('content');
  const { data, isPending, isError, error, refetch } = useDashboard();

  if (isPending) {
    return (
      <div className="page" aria-busy="true">
        <span className="sr-only">{t('hello', { name: instructor.display_name })}</span>
        <div className="stat-grid">
          {[0, 1, 2, 3].map((i) => (
            <Skeleton key={i} className="skeleton--tile" />
          ))}
        </div>
        <Skeleton className="skeleton--card" />
        <Skeleton className="skeleton--card" />
      </div>
    );
  }
  if (isError) {
    return (
      <div className="page">
        <QueryError error={error} onRetry={() => void refetch()} />
      </div>
    );
  }

  const s = data.stats;
  return (
    <div className="page">
      <h1 className="sr-only">{t('hello', { name: instructor.display_name })}</h1>
      <section className="stat-grid" aria-label={t('statsLabel')}>
        <StatTile label={t('activeStudents')} value={faNum(s.active_students)} note={t('newThisWeek', { count: faNum(s.new_students_week) })} tone="data" />
        <StatTile
          label={t('coursesTile')}
          value={faNum(s.courses)}
          note={s.drafts > 0 ? t('drafts', { count: faNum(s.drafts) }) : t('noDrafts')}
          tone="warm"
        />
        <StatTile label={t('watchHours')} value={faNum(hoursOf(s.watch_seconds_week))} note={t('last7Days')} tone="brand" title={t('watchNote')} />
        <StatTile label={t('groupsTile')} value={faNum(s.groups)} note={t('members', { count: faNum(s.group_members) })} tone="bloom" />
      </section>

      <nav className="quick-actions" aria-label={t('statsLabel')}>
        <QuickAction href="/content/upload" icon="upload" label={t('quickUpload')} tone="brand" />
        <QuickAction href="/students" icon="userPlus" label={t('quickAddStudent')} tone="data" />
        <QuickAction href="/groups" icon="groups" label={t('quickNewGroup')} tone="bloom" />
      </nav>

      <div className="dash-cols">
        <section className="panel-card">
          <header className="panel-card__head">
            <h2 className="panel-card__title">{t('myCourses')}</h2>
            <Link href="/content" className="panel-card__link">
              {t('all')}
            </Link>
          </header>
          {data.courses.length === 0 ? (
            <p className="panel-card__empty">{t('coursesEmpty')}</p>
          ) : (
            <ul className="row-list">
              {data.courses.slice(0, 3).map((c) => (
                <CourseRow
                  key={c.id}
                  course={c}
                  meta={
                    c.status === 'draft'
                      ? `${tc('lessonsCount', { count: faNum(c.lessons_count) })} · ${tc('status.draft')}`
                      : tc('lessonsCount', { count: faNum(c.lessons_count) })
                  }
                />
              ))}
            </ul>
          )}
        </section>

        <section className="panel-card">
          <header className="panel-card__head">
            <h2 className="panel-card__title">{t('activity')}</h2>
          </header>
          {data.activity.length === 0 ? (
            <p className="panel-card__empty">{t('activityEmpty')}</p>
          ) : (
            <ul className="activity">
              {data.activity.map((a, i) => (
                <ActivityRow key={`${a.type}-${a.at}-${i}`} item={a} />
              ))}
            </ul>
          )}
        </section>
      </div>
    </div>
  );
}

function StatTile({ label, value, note, tone, title }: { label: string; value: string; note: string; tone: string; title?: string }) {
  return (
    <div className="stat-tile" title={title}>
      <span className="stat-tile__label">{label}</span>
      <span className="stat-tile__value">{value}</span>
      <span className={`stat-tile__note stat-tile__note--${tone}`}>{note}</span>
    </div>
  );
}

function QuickAction({ href, icon, label, tone }: { href: string; icon: IconName; label: string; tone: string }) {
  return (
    <Link href={href} className="quick-action">
      <span className={`quick-action__icon quick-action__icon--${tone}`}>
        <Icon name={icon} />
      </span>
      <span className="quick-action__label">{label}</span>
    </Link>
  );
}

function ActivityRow({ item }: { item: Activity }) {
  const t = useTranslations('dashboard');
  let text: string;
  let tone: string;
  if (item.type === 'lesson_published') {
    text = t('actLessonPublished', { title: item.lesson?.title ?? '' });
    tone = 'brand';
  } else if (item.group) {
    text = t('actStudentsGroup', { count: faNum(item.count ?? 0), name: item.group.name ?? '' });
    tone = 'data';
  } else {
    text = t('actStudentsCourse', { count: faNum(item.count ?? 0), name: item.course?.title ?? '' });
    tone = 'data';
  }
  return (
    <li className="activity__item">
      <span className={`activity__dot activity__dot--${tone}`} aria-hidden="true" />
      <span className="activity__text">{text}</span>
      <time className="activity__time" dateTime={item.at ?? undefined}>
        {relativeTime(item.at)}
      </time>
    </li>
  );
}

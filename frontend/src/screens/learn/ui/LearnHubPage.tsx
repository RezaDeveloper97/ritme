'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useState } from 'react';

import {
  CourseCard,
  filterCourses,
  groupByInstructor,
  minutesOf,
  useMyCourses,
  type ContinueCard,
  type CourseFilter,
} from '@/entities/course';
import { pilot } from '@/shared/pilot';
import { Link, type Locale } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib/date';
import { Avatar, Card, EmptyState, Icon, PrimaryButton, SegmentedTabs, Skeleton, SkeletonGroup, SkyLayer, StatusPill } from '@/shared/ui';
import { BottomNav } from '@/widgets/bottom-nav';

const FILTERS: readonly CourseFilter[] = ['learning', 'done', 'all'];

/**
 * `/learn` — «دوره‌های من» (nbl_/nbd_Learn_Hub): the resume hero, filter tabs
 * and the courses grouped by instructor (access days, expired ones dimmed with
 * the «ask your instructor» note). Under «من» in the nav.
 */
export function LearnHubPage() {
  const t = useTranslations('learn');
  const q = useMyCourses();
  const [filter, setFilter] = useState<CourseFilter>('learning');

  let body;
  if (q.isPending) {
    body = (
      <SkeletonGroup label={t('common.loading')} className="lrn-body">
        <Skeleton shape="card" />
        <Skeleton shape="block" />
        <Skeleton shape="block" />
        <Skeleton shape="block" />
      </SkeletonGroup>
    );
  } else if (q.isError) {
    body = (
      <Card>
        <EmptyState
          icon="warning"
          title={t('common.loadError')}
          action={
            <PrimaryButton icon="refresh" loading={q.isFetching} onClick={() => void q.refetch()}>
              {t('common.retry')}
            </PrimaryButton>
          }
        />
      </Card>
    );
  } else if (q.data.courses.length === 0) {
    body = (
      <Card className="lrn-body">
        <EmptyState icon="gradCap" className="nb-tone-bloom" title={t('hub.emptyTitle')} body={t('hub.emptyBody')} />
      </Card>
    );
  } else {
    const list = filterCourses(q.data.courses, filter);
    const groups = groupByInstructor(list);
    const anyExpired = list.some((c) => c.access.expired);
    body = (
      <div className="lrn-body">
        {q.data.continue ? <ResumeHero item={q.data.continue} /> : null}
        <SegmentedTabs
          tabs={FILTERS.map((f) => ({ value: f, label: t(`hub.filters.${f}`) }))}
          value={filter}
          onChange={setFilter}
          label={t('hub.filtersLabel')}
          track="surface"
          pilot={(v) => pilot(`learn.filter:${v}`)}
        />
        {groups.length === 0 ? (
          <EmptyState icon="checkCircle" title={t(filter === 'done' ? 'hub.noneDone' : 'hub.noneLearning')} />
        ) : (
          groups.map((g) => (
            <section key={g.instructor.id} className="lrn-group" aria-label={t('hub.from', { name: g.instructor.name })}>
              <div className="lrn-group-head">
                <Avatar name={g.instructor.name} size="sm" />
                <span className="lrn-group-name">{t('hub.from', { name: g.instructor.name })}</span>
                {g.groupName ? (
                  <StatusPill tone="bloom" className="lrn-group-pill">
                    {g.groupName}
                  </StatusPill>
                ) : null}
              </div>
              <div className="lrn-cards">
                {g.courses.map((c) => (
                  <CourseCard key={c.id} course={c} />
                ))}
              </div>
            </section>
          ))
        )}
        {anyExpired ? <p className="lrn-note">{t('hub.expiredNote')}</p> : null}
      </div>
    );
  }

  return (
    <div className="view lrn-view">
      <SkyLayer />
      <div className="scroll">
        <header className="lrn-hub-head has-action">
          <h1 className="lrn-hub-title">{t('hub.title')}</h1>
          <Link href="/learn/downloads" className="nb-hbtn" aria-label={t('downloads.title')} {...pilot('learn.open:learn-downloads')}>
            <Icon name="download" size={20} />
          </Link>
        </header>
        {body}
        <div className="page-tail" />
      </div>
      <BottomNav />
    </div>
  );
}

function ResumeHero({ item }: { item: ContinueCard }) {
  const t = useTranslations('learn');
  const locale = useLocale() as Locale;
  const remaining = minutesOf(item.remainingSeconds);
  const pct = Math.max(0, Math.min(100, item.percent));
  return (
    <Link href={`/learn/${item.course.id}/${item.lesson.id}`} className="lrn-resume" {...pilot('learn.resume')}>
      <span className="lrn-resume-art" aria-hidden>
        <span className="lrn-play-disc">
          <Icon name="play" size={26} fill="currentColor" />
        </span>
      </span>
      <span className="lrn-resume-body">
        <span className="lrn-resume-top">
          <span>{t('hub.resume')}</span>
          {remaining ? <span>{t('meta.minutesLeft', { minutes: formatNumber(remaining, locale) })}</span> : null}
        </span>
        <span className="lrn-resume-title">
          {t('hub.lessonTitle', { number: formatNumber(item.lesson.number, locale), title: item.lesson.title })}
        </span>
        <span className="lrn-resume-sub">
          {item.course.title} · {item.course.instructor.name}
        </span>
        <span className="lrn-bar nb-tone-period" aria-hidden>
          <span style={{ inlineSize: `${pct}%` }} />
        </span>
      </span>
    </Link>
  );
}

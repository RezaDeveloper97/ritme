'use client';

import { clsx } from 'clsx';
import { useLocale, useTranslations } from 'next-intl';
import { useState } from 'react';

import {
  KIND_LOOK,
  useLessonDownload,
  lessonBlock,
  useAccessLabel,
  useCourseOutline,
  useLessonMeta,
  type CourseChapter,
  type StudentLesson,
} from '@/entities/course';
import { pilot } from '@/shared/pilot';
import { Link, useRouter, type Locale } from '@/shared/i18n';
import { formatDayMonth, formatNumber } from '@/shared/lib/date';
import { Avatar, Card, EmptyState, HeaderButton, Icon, InfoNote, PrimaryButton, Skeleton, SkeletonGroup, StatusPill } from '@/shared/ui';

type Tab = 'lessons' | 'about' | 'files';
const TABS: readonly Tab[] = ['lessons', 'about', 'files'];

/**
 * `/learn/[course]` — Learn_Course: hero, instructor, access pill, progress,
 * «ادامه · درس n», tabs (lessons by chapter / about / PDF files), locked
 * chapters with their unlock date and an expired-access banner. Offline
 * download (the toggle on the artboard) is B-N8-04.
 */
export function LearnCoursePage({ courseId }: { courseId: number }) {
  const t = useTranslations('learn');
  const locale = useLocale() as Locale;
  const router = useRouter();
  const q = useCourseOutline(courseId);
  const access = useAccessLabel();
  const [tab, setTab] = useState<Tab>('lessons');
  const back = () => router.push('/learn');

  if (q.isPending) {
    return (
      <Shell onBack={back} backLabel={t('common.back')}>
        <SkeletonGroup label={t('common.loading')} className="lrn-body">
          <Skeleton shape="line" width="medium" />
          <Skeleton shape="block" />
          <Skeleton shape="block" />
          <Skeleton shape="block" />
        </SkeletonGroup>
      </Shell>
    );
  }
  if (q.isError) {
    const notFound = lessonBlock(q.error) === 'notFound';
    return (
      <Shell onBack={back} backLabel={t('common.back')}>
        <Card className="lrn-body">
          <EmptyState
            icon={notFound ? 'lock' : 'warning'}
            title={t(notFound ? 'course.notFound' : 'common.loadError')}
            action={
              notFound ? (
                <PrimaryButton onClick={back}>{t('unlocked.later')}</PrimaryButton>
              ) : (
                <PrimaryButton icon="refresh" loading={q.isFetching} onClick={() => void q.refetch()}>
                  {t('common.retry')}
                </PrimaryButton>
              )
            }
          />
        </Card>
      </Shell>
    );
  }

  const o = q.data;
  const c = o.course;
  const pct = Math.max(0, Math.min(100, c.progressPercent));
  const allLessons = [...o.chapters.flatMap((ch) => ch.lessons), ...o.lessons];
  const files = allLessons.filter((l) => l.kind === 'pdf');
  const lockedChapters = o.chapters.filter((ch) => ch.locked && ch.unlockAt);
  const currentId = o.nextLesson?.id ?? null;

  return (
    <Shell onBack={back} backLabel={t('common.back')} onPlay={currentId ? () => router.push(`/learn/${c.id}/${currentId}`) : undefined} playLabel={t('course.play')}>
      <div className="lrn-body">
        <h1 className="lrn-course-title">{c.title}</h1>
        <div className="lrn-course-by">
          <Avatar name={c.instructor.name} size="sm" />
          <span className="lrn-course-ins">{c.instructor.title ? `${c.instructor.name} · ${c.instructor.title}` : c.instructor.name}</span>
          <StatusPill tone={c.access.expired ? 'danger' : 'data'} icon="clock" className="lrn-course-access">
            {access(c.access, 'until')}
          </StatusPill>
        </div>
        <div className="lrn-course-prog">
          <span>{t('course.progress', { done: formatNumber(c.completedLessons, locale), total: formatNumber(c.lessonsCount, locale) })}</span>
          <span>{t('course.percent', { percent: formatNumber(pct, locale) })}</span>
        </div>
        <span className="lrn-bar nb-tone-period" aria-hidden>
          <span style={{ inlineSize: `${pct}%` }} />
        </span>
        {c.access.expired ? (
          <InfoNote icon="lock" className="lrn-expired-note">
            {t('course.expired')}
          </InfoNote>
        ) : o.nextLesson ? (
          <Link href={`/learn/${c.id}/${o.nextLesson.id}`} className="lrn-cta" {...pilot('learn.continue')}>
            <Icon name="play" size={18} fill="currentColor" />
            {t('course.continue', { number: formatNumber(o.nextLesson.number, locale) })}
          </Link>
        ) : null}

        <div className="lrn-tabs" role="tablist" aria-label={t('course.tabsLabel')}>
          {TABS.map((v) => (
            <button key={v} type="button" role="tab" aria-selected={tab === v} className="lrn-tab" onClick={() => setTab(v)} {...pilot(`learn.course-tab:${v}`)}>
              {t(`course.tabs.${v}`)}
            </button>
          ))}
        </div>

        {tab === 'lessons' ? (
          <div className="lrn-chapters">
            {o.chapters.map((ch, i) => (
              <ChapterBlock key={ch.id} chapter={ch} index={i + 1} courseId={c.id} courseTitle={c.title} currentId={currentId} />
            ))}
            {o.lessons.length ? (
              <ul className="lrn-lessons">
                {o.lessons.map((l) => (
                  <LessonRow key={l.id} lesson={l} courseId={c.id} courseTitle={c.title} current={l.id === currentId} />
                ))}
              </ul>
            ) : null}
            {allLessons.length === 0 ? <EmptyState icon="bookOpen" title={t('course.noLessons')} /> : null}
            {lockedChapters.map((ch) => (
              <InfoNote key={ch.id} icon="lock" className="lrn-lock-note">
                {t('course.chapterUnlocks', {
                  chapter: formatNumber(o.chapters.indexOf(ch) + 1, locale),
                  date: formatDayMonth(new Date(ch.unlockAt ?? ''), locale),
                })}
              </InfoNote>
            ))}
          </div>
        ) : tab === 'about' ? (
          <Card className="lrn-about">
            <p>{c.description ?? t('course.noDescription')}</p>
          </Card>
        ) : files.length ? (
          <ul className="lrn-lessons">
            {files.map((l) => (
              <LessonRow key={l.id} lesson={l} courseId={c.id} courseTitle={c.title} current={false} />
            ))}
          </ul>
        ) : (
          <EmptyState icon="fileDoc" title={t('course.noFiles')} />
        )}
      </div>
    </Shell>
  );
}

function Shell({
  onBack,
  backLabel,
  onPlay,
  playLabel,
  children,
}: {
  onBack: () => void;
  backLabel: string;
  onPlay?: () => void;
  playLabel?: string;
  children: React.ReactNode;
}) {
  return (
    <div className="view lrn-view">
      <div className="scroll">
        <div className="lrn-hero">
          <HeaderButton label={backLabel} icon="arrowR" onClick={onBack} variant="soft" className="lrn-hero-back" />
          {onPlay ? (
            <button type="button" className="lrn-play-disc is-hero" onClick={onPlay} aria-label={playLabel}>
              <Icon name="play" size={26} fill="currentColor" />
            </button>
          ) : null}
        </div>
        {children}
        <div className="page-tail" />
      </div>
    </div>
  );
}

function ChapterBlock({ chapter, index, courseId, courseTitle, currentId }: { chapter: CourseChapter; index: number; courseId: number; courseTitle: string; currentId: number | null }) {
  const t = useTranslations('learn');
  const locale = useLocale() as Locale;
  const id = `lrn-ch-${chapter.id}`;
  return (
    <section className="lrn-chapter" aria-labelledby={id}>
      <div className="lrn-chapter-head">
        <h2 id={id} className="lrn-chapter-title">
          {t('course.chapter', { number: formatNumber(index, locale), title: chapter.title })}
        </h2>
        <span className="lrn-chapter-count">{t('course.lessonsCount', { count: formatNumber(chapter.lessons.length, locale) })}</span>
      </div>
      <ul className="lrn-lessons">
        {chapter.lessons.map((l) => (
          <LessonRow key={l.id} lesson={l} courseId={courseId} courseTitle={courseTitle} current={l.id === currentId} />
        ))}
      </ul>
    </section>
  );
}

function LessonRow({ lesson, courseId, courseTitle, current }: { lesson: StudentLesson; courseId: number; courseTitle: string; current: boolean }) {
  const t = useTranslations('learn');
  const locale = useLocale() as Locale;
  const meta = useLessonMeta();
  const done = lesson.progress?.completed ?? false;
  const look = KIND_LOOK[lesson.kind];
  const status = lesson.locked ? (
    <span className="lrn-st is-locked">
      <Icon name="lock" size={16} />
    </span>
  ) : done ? (
    <span className="lrn-st is-done">
      <Icon name="check" size={16} />
    </span>
  ) : current ? (
    <span className="lrn-st is-current">
      <Icon name="play" size={14} fill="currentColor" />
    </span>
  ) : (
    <span className="lrn-st">{formatNumber(lesson.number, locale)}</span>
  );
  const content = (
    <>
      {status}
      <span className="lrn-row-main">
        <span className="lrn-row-title">
          {lesson.title}
          {lesson.isNew && !lesson.locked && !done ? (
            <StatusPill tone="bloom" className="lrn-new">
              {t('course.new')}
            </StatusPill>
          ) : null}
        </span>
        <span className="lrn-row-meta">
          <Icon name={look.icon} size={14} className={clsx('lrn-row-kind', `nb-tone-${look.tone}`)} />
          {meta(lesson)}
        </span>
      </span>
    </>
  );
  return (
    <li className="lrn-li">
      {lesson.locked ? (
        <div className="lrn-row is-locked" aria-disabled>
          {content}
        </div>
      ) : (
        <Link href={`/learn/${courseId}/${lesson.id}`} {...pilot(`learn.lesson:${lesson.id}`)} className={clsx('lrn-row', current && 'is-current')} aria-current={current ? 'step' : undefined}>
          {content}
        </Link>
      )}
      {lesson.locked ? null : <RowDownload lesson={lesson} courseId={courseId} courseTitle={courseTitle} />}
    </li>
  );
}

/** Per-lesson offline download (the arrow beside each row on Learn_Course): tap to save, again to remove, cancel while running. */
function RowDownload({ lesson, courseId, courseTitle }: { lesson: StudentLesson; courseId: number; courseTitle: string }) {
  const t = useTranslations('learn');
  const dl = useLessonDownload({ lessonId: lesson.id, courseId, courseTitle, title: lesson.title, kind: lesson.kind });
  if (dl.state === 'unsupported' || dl.state === 'checking' || lesson.media?.status !== 'ready') return null;
  const busy = dl.state === 'downloading';
  const saved = dl.state === 'saved';
  return (
    <button
      type="button"
      className={clsx('lrn-row-dl', saved && 'is-saved')}
      aria-busy={busy || undefined}
      aria-label={busy ? t('download.cancel') : saved ? t('download.remove') : t('download.action')}
      onClick={() => (busy ? dl.cancel() : saved ? void dl.remove() : void dl.start())}
      {...pilot(`learn.download:${lesson.id}`)}
    >
      {busy ? <span className="lrn-row-dl-pct">{dl.percent ?? '…'}</span> : <Icon name={saved ? 'check' : 'download'} size={18} />}
    </button>
  );
}

'use client';

import { useMutation, useQueryClient } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { useState, type FormEvent } from 'react';

import {
  KindTile,
  courseKeys,
  createChapter,
  deleteChapter,
  deleteCourse,
  deleteLesson,
  faNum,
  groupByChapter,
  minutesOf,
  toTehranWallClock,
  updateChapter,
  updateCourse,
  updateLesson,
  useOutline,
  type Chapter,
  type Lesson,
  type Outline,
} from '@/entities/course';
import { fetchPlayback } from '@/features/media-upload';
import { useErrorMessage } from '@/shared/i18n';
import { cn } from '@/shared/lib';
import { Button, Icon, IconButton, QueryError, Skeleton, TextField } from '@/shared/ui';

import { moveLesson } from '../model/reorder';

const dateFormat = new Intl.DateTimeFormat('fa-IR-u-ca-persian', {
  month: 'long',
  day: 'numeric',
  hour: '2-digit',
  minute: '2-digit',
  timeZone: 'Asia/Tehran',
});

/** /content/{id} — nbl_Ins_Course: preview, stats, chapters + lessons (reorder, publish, scheduled unlock), access. */
export function CourseScreen({ id }: { id: number }) {
  const { data, isPending, isError, error, refetch } = useOutline(id);
  if (isPending) {
    return (
      <div className="page" aria-busy="true">
        <Skeleton className="skeleton--line" />
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
  return <CourseDetail outline={data} />;
}

function useInvalidate(id: number) {
  const qc = useQueryClient();
  return () => {
    void qc.invalidateQueries({ queryKey: courseKeys.outline(id) });
    void qc.invalidateQueries({ queryKey: courseKeys.dashboard() });
  };
}

function CourseDetail({ outline }: { outline: Outline }) {
  const t = useTranslations('courseDetail');
  const tc = useTranslations('content');
  const errorMessage = useErrorMessage();
  const router = useRouter();
  const { course } = outline;
  const invalidate = useInvalidate(course.id);
  const [menu, setMenu] = useState(false);
  const [addingChapter, setAddingChapter] = useState(false);

  const toggleCourse = useMutation({
    mutationFn: () => updateCourse(course.id, { status: course.status === 'published' ? 'draft' : 'published' }),
    onSuccess: () => {
      setMenu(false);
      invalidate();
    },
  });
  const removeCourse = useMutation({
    mutationFn: () => deleteCourse(course.id),
    onSuccess: () => {
      invalidate();
      router.push('/content');
    },
  });

  const published = outline.lessons.filter((l) => l.status === 'published').length;
  const minutes = minutesOf(outline.duration_seconds);
  const groups = groupByChapter(outline.chapters, outline.lessons);
  const standalone = course.kind === 'standalone';
  let n = 0;

  return (
    <div className="page">
      <header className="page-head page-head--center">
        <Link href="/content" className="icon-btn" aria-label={t('back')} title={t('back')}>
          <Icon name="back" />
        </Link>
        <div className="page-head__text">
          <h1 className="page-head__title">{course.title}</h1>
          <p className="page-head__sub">
            {t('subtitle', { kind: tc(`kind.${course.kind}`), status: tc(`status.${course.status}`) })}
          </p>
        </div>
        <div className="menu">
          <IconButton label={t('more')} aria-expanded={menu} onClick={() => setMenu((v) => !v)}>
            <Icon name="more" />
          </IconButton>
          {menu ? (
            <div className="menu__pop" role="menu">
              <button type="button" role="menuitem" className="menu__item" onClick={() => toggleCourse.mutate()}>
                <Icon name={course.status === 'published' ? 'edit' : 'check'} size={18} />
                {course.status === 'published' ? t('unpublishCourse') : t('publishCourse')}
              </button>
              <button
                type="button"
                role="menuitem"
                className="menu__item menu__item--danger"
                onClick={() => {
                  if (window.confirm(t('deleteCourseConfirm', { title: course.title }))) removeCourse.mutate();
                }}
              >
                <Icon name="trash" size={18} />
                {t('deleteCourse')}
              </button>
            </div>
          ) : null}
        </div>
      </header>
      {toggleCourse.isError || removeCourse.isError ? (
        <p className="form-error" role="alert">
          {errorMessage(toggleCourse.error ?? removeCourse.error)}
        </p>
      ) : null}

      <Preview lessons={outline.lessons} />

      <section className="stat-grid stat-grid--3">
        <div className="stat-tile">
          <span className="stat-tile__label">{t('access')}</span>
          <span className="stat-tile__value">{faNum(outline.access.students)}</span>
          <span className="stat-tile__note stat-tile__note--data">{t('groupsCount', { count: faNum(outline.access.groups.length) })}</span>
        </div>
        <div className="stat-tile">
          <span className="stat-tile__label">{t('published')}</span>
          <span className="stat-tile__value">{faNum(published)}</span>
          <span className="stat-tile__note stat-tile__note--brand">{t('ofLessons', { count: faNum(outline.lessons.length) })}</span>
        </div>
        <div className="stat-tile">
          <span className="stat-tile__label">{t('lessons')}</span>
          <span className="stat-tile__value">{faNum(outline.lessons.length)}</span>
          <span className="stat-tile__note stat-tile__note--warm">{t('duration', { count: faNum(minutes ?? 0) })}</span>
        </div>
      </section>

      <section className="panel-card">
        <header className="panel-card__head">
          <h2 className="panel-card__title">{t('lessonsTitle')}</h2>
          {!standalone || outline.lessons.length === 0 ? (
            <Link href={`/content/upload?course=${course.id}`} className="panel-card__link">
              {t('addLesson')}
            </Link>
          ) : null}
        </header>
        {outline.lessons.length === 0 && outline.chapters.length === 0 ? (
          <p className="panel-card__empty">{t('noLessons')}</p>
        ) : (
          groups.map((g) => (
            <div key={g.chapter?.id ?? 'none'} className="chapter">
              {g.chapter ? (
                <ChapterHead courseId={course.id} chapter={g.chapter} />
              ) : groups.length > 1 ? (
                <h3 className="chapter__title">{t('noChapter')}</h3>
              ) : null}
              <ul className="row-list">
                {g.lessons.map((l) => {
                  n += 1;
                  return <LessonRow key={l.id} courseId={course.id} lesson={l} index={n} group={g.lessons} />;
                })}
              </ul>
            </div>
          ))
        )}
        {!standalone ? (
          addingChapter ? (
            <ChapterForm courseId={course.id} onDone={() => setAddingChapter(false)} />
          ) : (
            <button type="button" className="link-btn" onClick={() => setAddingChapter(true)}>
              {t('addChapter')}
            </button>
          )
        ) : null}
      </section>

      <section className="panel-card">
        <header className="panel-card__head">
          <h2 className="panel-card__title">{t('whoHasAccess')}</h2>
          <Link href="/students" className="panel-card__link">
            {t('addAccess')}
          </Link>
        </header>
        {outline.access.groups.length === 0 && outline.access.direct === 0 ? (
          <p className="panel-card__empty">{t('noAccess')}</p>
        ) : (
          <div className="chips">
            {outline.access.groups.map((g, i) => (
              <span key={g.id} className={cn('access-chip', i % 2 === 0 ? 'access-chip--bloom' : 'access-chip--data')}>
                {t('groupChip', { name: g.name, count: faNum(g.members) })}
              </span>
            ))}
            {outline.access.direct > 0 ? (
              <span className="access-chip access-chip--brand">{t('directChip', { count: faNum(outline.access.direct) })}</span>
            ) : null}
          </div>
        )}
      </section>
    </div>
  );
}

function Preview({ lessons }: { lessons: Lesson[] }) {
  const t = useTranslations('courseDetail');
  const ready = lessons.find((l) => l.media?.status === 'ready' && l.kind !== 'pdf');
  const [src, setSrc] = useState<{ url: string; kind: string } | null>(null);
  const load = useMutation({
    mutationFn: () => fetchPlayback(ready?.media?.id ?? 0),
    onSuccess: (p) => setSrc({ url: p.playback.url, kind: p.playback.kind }),
  });
  if (src) {
    return src.kind === 'audio' ? (
      <audio className="preview preview--audio" src={src.url} controls autoPlay />
    ) : (
      <video className="preview" src={src.url} controls autoPlay playsInline />
    );
  }
  return (
    <div className="preview">
      {ready ? (
        <button type="button" className="preview__play" aria-label={t('preview')} onClick={() => load.mutate()} disabled={load.isPending}>
          <Icon name="play" size={40} />
        </button>
      ) : (
        <p className="preview__empty">{t('previewEmpty')}</p>
      )}
    </div>
  );
}

function ChapterHead({ courseId, chapter }: { courseId: number; chapter: Chapter }) {
  const t = useTranslations('courseDetail');
  const invalidate = useInvalidate(courseId);
  const [editing, setEditing] = useState(false);
  const remove = useMutation({ mutationFn: () => deleteChapter(courseId, chapter.id), onSuccess: invalidate });
  if (editing) return <ChapterForm courseId={courseId} chapter={chapter} onDone={() => setEditing(false)} />;
  return (
    <div className="chapter__head">
      <h3 className="chapter__title">{chapter.title}</h3>
      {chapter.locked && chapter.unlock_at ? (
        <span className="chip chip--warm">
          <Icon name="lock" size={14} />
          {t('chapterLocked', { date: dateFormat.format(new Date(chapter.unlock_at)) })}
        </span>
      ) : null}
      <span className="chapter__actions">
        <IconButton label={t('editChapter')} onClick={() => setEditing(true)}>
          <Icon name="edit" size={18} />
        </IconButton>
        <IconButton
          label={t('deleteChapter')}
          onClick={() => {
            if (window.confirm(t('deleteChapterConfirm'))) remove.mutate();
          }}
        >
          <Icon name="trash" size={18} />
        </IconButton>
      </span>
    </div>
  );
}

function ChapterForm({ courseId, chapter, onDone }: { courseId: number; chapter?: Chapter; onDone: () => void }) {
  const t = useTranslations('courseDetail');
  const tc = useTranslations('content');
  const errorMessage = useErrorMessage();
  const invalidate = useInvalidate(courseId);
  const [title, setTitle] = useState(chapter?.title ?? '');
  const [unlock, setUnlock] = useState(toTehranWallClock(chapter?.unlock_at ?? null).replace(' ', 'T'));
  const save = useMutation({
    mutationFn: () => {
      const input = { title: title.trim(), unlock_at: unlock ? unlock.replace('T', ' ') : null };
      return chapter ? updateChapter(courseId, chapter.id, input) : createChapter(courseId, input);
    },
    onSuccess: () => {
      invalidate();
      onDone();
    },
  });
  function submit(e: FormEvent) {
    e.preventDefault();
    if (title.trim()) save.mutate();
  }
  return (
    <form className="form-card form-card--inline" onSubmit={submit} noValidate>
      <TextField label={t('chapterTitle')} value={title} maxLength={150} onChange={(e) => setTitle(e.target.value)} autoFocus />
      <TextField
        label={t('unlockAt')}
        hint={t('unlockHint')}
        type="datetime-local"
        value={unlock}
        onChange={(e) => setUnlock(e.target.value)}
        error={save.isError ? errorMessage(save.error) : undefined}
      />
      <div className="form-card__actions">
        <Button type="submit" loading={save.isPending} disabled={!title.trim()}>
          {t('saveChapter')}
        </Button>
        <Button variant="text" onClick={onDone}>
          {tc('cancel')}
        </Button>
      </div>
    </form>
  );
}

function LessonRow({ courseId, lesson, index, group }: { courseId: number; lesson: Lesson; index: number; group: Lesson[] }) {
  const t = useTranslations('courseDetail');
  const errorMessage = useErrorMessage();
  const invalidate = useInvalidate(courseId);
  const [open, setOpen] = useState(false);
  const move = useMutation({
    mutationFn: async (dir: -1 | 1) => {
      for (const u of moveLesson(group, lesson.id, dir)) await updateLesson(courseId, u.id, { sort_order: u.sort_order });
    },
    onSettled: invalidate,
  });
  const toggle = useMutation({
    mutationFn: () => updateLesson(courseId, lesson.id, { status: lesson.status === 'published' ? 'draft' : 'published' }),
    onSuccess: invalidate,
  });
  const remove = useMutation({ mutationFn: () => deleteLesson(courseId, lesson.id), onSuccess: invalidate });
  const failed = move.error ?? toggle.error ?? remove.error;

  const minutes = minutesOf(lesson.duration_seconds);
  const meta = [
    lesson.kind === 'pdf'
      ? lesson.page_count
        ? t('pages', { count: faNum(lesson.page_count) })
        : 'PDF'
      : minutes
        ? t('duration', { count: faNum(minutes) })
        : null,
    lesson.status === 'draft' ? t('draft') : null,
    lesson.media_status !== 'ready' ? t(`media.${mediaKey(lesson.media_status)}`) : null,
  ]
    .filter(Boolean)
    .join(' · ');
  const i = group.findIndex((l) => l.id === lesson.id);

  return (
    <li className="lesson">
      <div className="row row--static">
        <span className="lesson__num">{faNum(index)}</span>
        <KindTile kind={lesson.kind} size="sm" />
        <span className="row__body">
          <span className="row__title">{lesson.title}</span>
          <span className="row__meta">{meta}</span>
        </span>
        <IconButton label={t('lessonMenu', { title: lesson.title })} aria-expanded={open} onClick={() => setOpen((v) => !v)}>
          <Icon name="more" />
        </IconButton>
      </div>
      {open ? (
        <div className="lesson__actions">
          <IconButton label={t('moveUp')} disabled={i <= 0 || move.isPending} onClick={() => move.mutate(-1)}>
            <Icon name="up" size={18} />
          </IconButton>
          <IconButton label={t('moveDown')} disabled={i >= group.length - 1 || move.isPending} onClick={() => move.mutate(1)}>
            <Icon name="down" size={18} />
          </IconButton>
          <button type="button" className="pill-btn" onClick={() => toggle.mutate()} disabled={toggle.isPending}>
            {lesson.status === 'published' ? t('unpublish') : t('publish')}
          </button>
          <Link href={`/content/upload?course=${courseId}&lesson=${lesson.id}`} className="pill-btn">
            {t('uploadFile')}
          </Link>
          <button
            type="button"
            className="pill-btn pill-btn--danger"
            onClick={() => {
              if (window.confirm(t('deleteLessonConfirm', { title: lesson.title }))) remove.mutate();
            }}
          >
            {t('deleteLesson')}
          </button>
        </div>
      ) : null}
      {failed ? (
        <p className="form-error" role="alert">
          {errorMessage(failed)}
        </p>
      ) : null}
    </li>
  );
}

function mediaKey(status: string): 'none' | 'uploading' | 'processing' | 'ready' | 'failed' {
  return status === 'uploading' || status === 'processing' || status === 'ready' || status === 'failed' ? status : 'none';
}

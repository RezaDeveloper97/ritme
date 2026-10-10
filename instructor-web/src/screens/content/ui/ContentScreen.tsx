'use client';

import { useMutation, useQueryClient } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { useState, type FormEvent } from 'react';

import {
  COURSE_LIMITS,
  CourseRow,
  courseKeys,
  createCourse,
  faNum,
  minutesOf,
  useDashboard,
  type CourseListItem,
} from '@/entities/course';
import { useErrorMessage } from '@/shared/i18n';
import { cn } from '@/shared/lib';
import { Button, Icon, QueryError, Skeleton, TextField } from '@/shared/ui';

import { CONTENT_FILTERS, filterCourses, type ContentFilter } from '../model/filter';

/** /content — nbl_Ins_Content: every course and standalone item, kind filter, new course. */
export function ContentScreen() {
  const t = useTranslations('content');
  const { data, isPending, isError, error, refetch } = useDashboard();
  const [filter, setFilter] = useState<ContentFilter>('all');
  const [creating, setCreating] = useState(false);

  const courses = data?.courses ?? [];
  const files = courses.reduce((n, c) => n + c.lessons_count, 0);
  const shown = filterCourses(courses, filter);

  return (
    <div className="page">
      <header className="page-head">
        <div className="page-head__text">
          <h1 className="page-head__title">{t('title')}</h1>
          {data ? (
            <p className="page-head__sub">{t('summary', { courses: faNum(courses.length), files: faNum(files) })}</p>
          ) : null}
        </div>
        <Link href="/content/upload" className="icon-btn" aria-label={t('upload')} title={t('upload')}>
          <Icon name="upload" />
        </Link>
      </header>

      <div className="segmented" role="tablist" aria-label={t('filtersLabel')}>
        {CONTENT_FILTERS.map((f) => (
          <button
            key={f}
            type="button"
            role="tab"
            aria-selected={filter === f}
            className={cn('segmented__item', filter === f && 'is-active')}
            onClick={() => setFilter(f)}
          >
            {t(`filter.${f}`)}
          </button>
        ))}
      </div>

      {isPending ? (
        <Skeleton className="skeleton--card" />
      ) : isError ? (
        <QueryError error={error} onRetry={() => void refetch()} />
      ) : (
        <section className="panel-card">
          {shown.length === 0 ? (
            <p className="panel-card__empty">{courses.length === 0 ? t('empty') : t('emptyFilter')}</p>
          ) : (
            <ul className="row-list">
              {shown.map((c) => (
                <CourseRow key={c.id} course={c} meta={metaOf(c, t)} />
              ))}
            </ul>
          )}
        </section>
      )}

      {creating ? (
        <NewCourseForm onCancel={() => setCreating(false)} />
      ) : (
        <Button variant="ghost" onClick={() => setCreating(true)}>
          <Icon name="plus" size={18} />
          {t('newCourse')}
        </Button>
      )}
    </div>
  );
}

type T = ReturnType<typeof useTranslations<'content'>>;

function metaOf(c: CourseListItem, t: T): string {
  const kind = c.lesson_kinds[0];
  const minutes = minutesOf(c.duration_seconds);
  if (c.kind === 'standalone') {
    const parts = [t('standaloneMeta', { kind: kind ? t(`lessonKind.${kind}`) : t('kind.standalone') })];
    if (minutes) parts.push(t('minutes', { count: faNum(minutes) }));
    return parts.join(' · ');
  }
  const parts = [t('kind.course'), t('lessonsCount', { count: faNum(c.lessons_count) })];
  if (minutes) parts.push(t('minutes', { count: faNum(minutes) }));
  return parts.join(' · ');
}

function NewCourseForm({ onCancel }: { onCancel: () => void }) {
  const t = useTranslations('content');
  const errorMessage = useErrorMessage();
  const router = useRouter();
  const qc = useQueryClient();
  const [title, setTitle] = useState('');
  const [invalid, setInvalid] = useState(false);
  const mutation = useMutation({
    mutationFn: (value: string) => createCourse({ kind: 'course', title: value }),
    onSuccess: (outline) => {
      void qc.invalidateQueries({ queryKey: courseKeys.all });
      router.push(`/content/${outline.course.id}`);
    },
  });

  function submit(e: FormEvent) {
    e.preventDefault();
    const value = title.trim();
    if (!value) {
      setInvalid(true);
      return;
    }
    mutation.mutate(value);
  }

  return (
    <form className="panel-card form-card" onSubmit={submit} noValidate>
      <TextField
        label={t('newCourseTitle')}
        hint={t('newCourseHint')}
        value={title}
        maxLength={COURSE_LIMITS.titleMax}
        onChange={(e) => {
          setTitle(e.target.value);
          setInvalid(false);
        }}
        error={invalid ? t('titleRequired') : mutation.isError ? errorMessage(mutation.error) : undefined}
        autoFocus
      />
      <div className="form-card__actions">
        <Button type="submit" loading={mutation.isPending}>
          {t('create')}
        </Button>
        <Button variant="text" onClick={onCancel}>
          {t('cancel')}
        </Button>
      </div>
    </form>
  );
}

'use client';

import { useMutation, useQueryClient } from '@tanstack/react-query';
import { useTranslations } from 'next-intl';
import Link from 'next/link';
import { useRouter, useSearchParams } from 'next/navigation';
import { useEffect, useRef, useState, type ChangeEvent } from 'react';

import {
  COURSE_LIMITS,
  KindTile,
  courseKeys,
  createCourse,
  createLesson,
  faNum,
  lessonKindOfFile,
  megabytesOf,
  updateCourse,
  updateLesson,
  useCourses,
  useOutline,
  type LessonKind,
} from '@/entities/course';
import {
  isSameFile,
  percentOf,
  readPending,
  removePending,
  deleteUpload,
  uploadErrorKey,
  useResumableUpload,
  type PendingUpload,
} from '@/features/media-upload';
import { useErrorMessage } from '@/shared/i18n';
import { cn } from '@/shared/lib';
import { Button, Icon, TextField } from '@/shared/ui';

const ACCEPT = 'video/mp4,video/quicktime,video/webm,audio/*,application/pdf,.mp4,.mov,.webm,.mp3,.m4a,.pdf';

const num = (v: string | null) => {
  const n = Number(v);
  return Number.isInteger(n) && n > 0 ? n : null;
};

/**
 * /content/upload — nbl_Ins_Upload. Pick a file, set title / placement (inside a course + chapter, or standalone),
 * publish or save a draft. The lesson is created as a draft, the file goes up in resumable chunks, and «انتشار»
 * publishes the lesson only once the file is ready. A reload keeps the upload: picking the same file continues
 * from the server's offset. Per-lesson visibility (groups / people) is not supported by the API yet.
 */
export function UploadScreen() {
  const t = useTranslations('upload');
  const errorMessage = useErrorMessage();
  const router = useRouter();
  const qc = useQueryClient();
  const params = useSearchParams();
  const lessonParam = num(params.get('lesson'));
  const courseParam = num(params.get('course'));

  const [file, setFile] = useState<File | null>(null);
  const [kind, setKind] = useState<LessonKind | null>(null);
  const [title, setTitle] = useState('');
  const [description, setDescription] = useState('');
  const [placement, setPlacement] = useState<'course' | 'standalone'>('course');
  const [courseId, setCourseId] = useState<number | null>(courseParam);
  const [chapterId, setChapterId] = useState<number | null>(null);
  const [formError, setFormError] = useState<string | null>(null);
  const [pending, setPending] = useState<PendingUpload[]>([]);
  const [resumeFor, setResumeFor] = useState<PendingUpload | null>(null);
  const [doneCourse, setDoneCourse] = useState<number | null>(null);
  const pickRef = useRef<HTMLInputElement>(null);
  const resumeRef = useRef<HTMLInputElement>(null);

  const courses = useCourses();
  const outline = useOutline(courseId);
  const existing = lessonParam ? outline.data?.lessons.find((l) => l.id === lessonParam) : undefined;
  const courseOptions = (courses.data?.courses ?? []).filter((c) => c.kind === 'course');

  useEffect(() => setPending(readPending()), []);

  const upload = useResumableUpload(async (p) => {
    if (p.publish) {
      await updateLesson(p.courseId, p.lessonId, { status: 'published' });
      if (p.standalone) await updateCourse(p.courseId, { status: 'published' });
    }
    setDoneCourse(p.courseId);
    setPending(readPending().filter((x) => x.mediaId !== p.mediaId));
    void qc.invalidateQueries({ queryKey: courseKeys.all });
  });

  const create = useMutation({
    mutationFn: async ({ publish }: { publish: boolean }) => {
      if (!file || !kind) throw new Error('file');
      let cid = lessonParam ? courseParam : courseId;
      let lessonId = lessonParam;
      const standalone = !lessonParam && placement === 'standalone';
      if (!lessonId) {
        if (standalone) {
          cid = (await createCourse({ kind: 'standalone', title: title.trim(), description: description.trim() || null })).course.id;
        }
        if (!cid) throw new Error('course');
        const { lesson } = await createLesson(cid, {
          kind,
          title: title.trim(),
          description: description.trim() || null,
          chapter_id: standalone ? null : chapterId,
          size_bytes: file.size,
          status: 'draft',
        });
        lessonId = lesson.id;
      }
      if (!cid || !lessonId) throw new Error('course');
      void qc.invalidateQueries({ queryKey: courseKeys.all });
      await upload.start(file, {
        lessonId,
        courseId: cid,
        lessonTitle: existing?.title ?? title.trim(),
        publish,
        standalone,
      });
    },
  });

  function pick(e: ChangeEvent<HTMLInputElement>) {
    const f = e.target.files?.[0] ?? null;
    e.target.value = '';
    if (!f) return;
    const k = lessonKindOfFile(f);
    if (!k || (existing && existing.kind !== k)) {
      setFormError(t('unsupported'));
      return;
    }
    setFormError(null);
    setFile(f);
    setKind(k);
    if (!title) setTitle(f.name.replace(/\.[^.]+$/, '').slice(0, COURSE_LIMITS.titleMax));
  }

  function pickResume(e: ChangeEvent<HTMLInputElement>) {
    const f = e.target.files?.[0] ?? null;
    e.target.value = '';
    if (!f || !resumeFor) return;
    if (!isSameFile(resumeFor, f)) {
      setFormError(t('wrongFile', { name: resumeFor.name }));
      return;
    }
    setFormError(null);
    setFile(f);
    setKind(lessonKindOfFile(f));
    void upload.resume(resumeFor, f);
  }

  function submit(publish: boolean) {
    if (!file) return setFormError(t('errors.fileRequired'));
    if (!lessonParam && !title.trim()) return setFormError(t('errors.titleRequired'));
    if (!lessonParam && placement === 'course' && !courseId) return setFormError(t('errors.courseRequired'));
    setFormError(null);
    create.mutate({ publish });
  }

  const state = upload.state;
  const busy = state.phase === 'uploading' || create.isPending;
  const failure = create.error ?? (state.phase === 'error' ? state.error : null);
  const failureKey = uploadErrorKey(failure);
  const failureText = failure ? (failureKey ? t(`errors.${failureKey}`) : errorMessage(failure)) : null;
  const percent = percentOf(state.sent, state.total);
  const standaloneMode = !lessonParam && placement === 'standalone';

  if (state.phase === 'done') {
    return (
      <div className="page page--narrow">
        <UploadHead title={t('title')} back={t('back')} onBack={() => router.back()} />
        <section className="state-card">
          <span className="state-card__icon">
            <Icon name="check" />
          </span>
          <p className="state-card__title">{t('done')}</p>
          <p className="state-card__text">{t('doneLead')}</p>
          {doneCourse ? (
            <Link href={`/content/${doneCourse}`} className="btn btn--primary">
              {t('openCourse')}
            </Link>
          ) : null}
          <Button variant="text" onClick={() => window.location.assign('/content/upload')}>
            {t('another')}
          </Button>
        </section>
      </div>
    );
  }

  return (
    <div className="page page--narrow">
      <UploadHead title={t('title')} back={t('back')} onBack={() => router.back()} />

      {state.phase === 'idle' && !file
        ? pending.map((p) => (
            <section key={p.mediaId} className="panel-card resume-card">
              <h2 className="panel-card__title">{t('resumeTitle')}</h2>
              <p className="panel-card__empty">{t('resumeLeadNoPercent', { name: p.name, title: p.lessonTitle })}</p>
              <div className="form-card__actions">
                <Button
                  onClick={() => {
                    setResumeFor(p);
                    resumeRef.current?.click();
                  }}
                >
                  {t('resumePick')}
                </Button>
                <Button
                  variant="text"
                  onClick={() => {
                    removePending(p.mediaId);
                    setPending((list) => list.filter((x) => x.mediaId !== p.mediaId));
                    void deleteUpload(p.mediaId).catch(() => undefined);
                  }}
                >
                  {t('resumeDiscard')}
                </Button>
              </div>
            </section>
          ))
        : null}
      <input ref={resumeRef} type="file" accept={ACCEPT} className="sr-only" tabIndex={-1} aria-hidden="true" onChange={pickResume} />

      {existing ? <p className="page-head__sub">{t('existingLesson', { title: existing.title })}</p> : null}

      {state.phase === 'idle' ? (
        <button type="button" className="dropzone" onClick={() => pickRef.current?.click()} disabled={busy}>
          <span className="dropzone__icon">
            <Icon name="upload" size={26} />
          </span>
          <span className="dropzone__title">{file ? t('change') : t('pick')}</span>
          <span className="dropzone__hint">{t('pickHint')}</span>
        </button>
      ) : null}
      <input ref={pickRef} type="file" accept={ACCEPT} className="sr-only" tabIndex={-1} aria-hidden="true" onChange={pick} />

      {file && kind ? (
        <section className="panel-card file-card">
          <KindTile kind={kind} />
          <div className="file-card__body">
            <span className="file-card__name">{file.name}</span>
            <progress className="upload-bar" max={100} value={state.phase === 'idle' ? 0 : percent} aria-label={file.name} />
            <span className="row__meta">
              {state.phase === 'paused'
                ? t('paused')
                : t('progress', {
                    percent: faNum(state.phase === 'idle' ? 0 : percent),
                    sent: faNum(megabytesOf(state.phase === 'idle' ? 0 : state.sent)),
                    total: faNum(megabytesOf(file.size)),
                  })}
            </span>
          </div>
          {state.phase === 'uploading' ? (
            <button type="button" className="icon-btn" aria-label={t('pause')} title={t('pause')} onClick={upload.pause}>
              <Icon name="pause" size={18} />
            </button>
          ) : state.phase === 'paused' || state.phase === 'error' ? (
            <button type="button" className="icon-btn" aria-label={t('resumeBtn')} title={t('resumeBtn')} onClick={() => void upload.retry()}>
              <Icon name="refresh" size={18} />
            </button>
          ) : null}
        </section>
      ) : null}

      {state.phase !== 'idle' ? (
        <Button variant="text" onClick={() => void upload.cancel().then(() => setFile(null))}>
          {t('cancelUpload')}
        </Button>
      ) : null}

      {!lessonParam ? (
        <>
          <section className="panel-card form-card">
            <h2 className="panel-card__title">{t('details')}</h2>
            <TextField
              label={t('titleLabel')}
              value={title}
              maxLength={COURSE_LIMITS.titleMax}
              onChange={(e) => setTitle(e.target.value)}
              disabled={state.phase !== 'idle'}
            />
            <TextField
              label={t('descLabel')}
              placeholder={t('descHint')}
              value={description}
              maxLength={COURSE_LIMITS.descriptionMax}
              onChange={(e) => setDescription(e.target.value)}
              disabled={state.phase !== 'idle'}
            />
          </section>

          <section className="panel-card form-card">
            <h2 className="panel-card__title">{t('placement')}</h2>
            <div className="choice-row" role="radiogroup" aria-label={t('placement')}>
              {(['course', 'standalone'] as const).map((p) => (
                <button
                  key={p}
                  type="button"
                  role="radio"
                  aria-checked={placement === p}
                  className={cn('choice', placement === p && 'is-active')}
                  onClick={() => setPlacement(p)}
                  disabled={state.phase !== 'idle'}
                >
                  {p === 'course' ? t('inCourse') : t('standalone')}
                </button>
              ))}
            </div>
            {placement === 'course' ? (
              courses.data && courseOptions.length === 0 ? (
                <p className="panel-card__empty">{t('noCourses')}</p>
              ) : (
                <>
                  <SelectField
                    label={t('course')}
                    icon="folder"
                    value={courseId ?? ''}
                    onChange={(v) => {
                      setCourseId(v ? Number(v) : null);
                      setChapterId(null);
                    }}
                    disabled={state.phase !== 'idle'}
                    options={[{ value: '', label: t('chooseCourse') }, ...courseOptions.map((c) => ({ value: c.id, label: c.title }))]}
                  />
                  {outline.data && outline.data.chapters.length > 0 ? (
                    <SelectField
                      label={t('chapter')}
                      icon="list"
                      value={chapterId ?? ''}
                      onChange={(v) => setChapterId(v ? Number(v) : null)}
                      disabled={state.phase !== 'idle'}
                      options={[
                        { value: '', label: t('noChapterOption') },
                        ...outline.data.chapters.map((c) => ({ value: c.id, label: c.title })),
                      ]}
                    />
                  ) : null}
                </>
              )
            ) : null}
          </section>

          <section className="panel-card form-card">
            <h2 className="panel-card__title">{t('visibility')}</h2>
            <ul className="radio-list">
              <li className="radio-list__item is-active">
                <span className="radio-dot is-on" aria-hidden="true" />
                {standaloneMode ? t('visAllStandalone') : t('visAll')}
              </li>
              <li className="radio-list__item is-disabled">
                <span className="radio-dot" aria-hidden="true" />
                {t('visGroups')}
              </li>
              <li className="radio-list__item is-disabled">
                <span className="radio-dot" aria-hidden="true" />
                {t('visPeople')}
              </li>
            </ul>
            <p className="field__hint">{t('visSoon')}</p>
          </section>
        </>
      ) : null}

      {formError || failureText ? (
        <p className="form-error" role="alert">
          {formError ?? failureText}
        </p>
      ) : null}

      {state.phase === 'idle' ? (
        <div className="upload-actions">
          {lessonParam ? (
            <Button block onClick={() => submit(false)} loading={busy}>
              {t('pick')}
            </Button>
          ) : (
            <>
              <Button block onClick={() => submit(true)} loading={busy}>
                {t('publish')}
              </Button>
              <p className="field__hint">{t('publishNote')}</p>
              <Button variant="text" onClick={() => submit(false)} disabled={busy}>
                {t('saveDraft')}
              </Button>
            </>
          )}
        </div>
      ) : null}
    </div>
  );
}

function UploadHead({ title, back, onBack }: { title: string; back: string; onBack: () => void }) {
  return (
    <header className="page-head page-head--center">
      <button type="button" className="icon-btn" aria-label={back} title={back} onClick={onBack}>
        <Icon name="back" />
      </button>
      <div className="page-head__text">
        <h1 className="page-head__title">{title}</h1>
      </div>
      <span className="page-head__spacer" aria-hidden="true" />
    </header>
  );
}

function SelectField({
  label,
  icon,
  value,
  options,
  onChange,
  disabled,
}: {
  label: string;
  icon: 'folder' | 'list';
  value: number | string;
  options: { value: number | string; label: string }[];
  onChange: (v: string) => void;
  disabled?: boolean;
}) {
  return (
    <label className="field">
      <span className="field__label">{label}</span>
      <span className="select">
        <Icon name={icon} size={18} />
        <select className="select__control" value={value} onChange={(e) => onChange(e.target.value)} disabled={disabled}>
          {options.map((o) => (
            <option key={o.value} value={o.value}>
              {o.label}
            </option>
          ))}
        </select>
      </span>
    </label>
  );
}

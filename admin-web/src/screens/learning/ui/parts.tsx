'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useEffect, useRef, useState, type FormEvent, type ReactNode } from 'react';

import { formatNumber } from '@/shared/lib';
import { Badge, Button, TextArea, toast, useNotifyError } from '@/shared/ui';

import { useLessonAction, type LessonAction } from '../api/learning';
import { instructorTone, lessonActions, NOTE_MAX, noteError, publicationTone, reviewTone } from '../lib/review';

/** «در انتظار بازبینی» / «تغییر کرده» / … */
export function ReviewBadge({ state }: { state: string }) {
  const t = useTranslations('learning.reviewState');
  return <Badge tone={reviewTone(state)}>{t.has(state as 'pending') ? t(state as 'pending') : state}</Badge>;
}

export function InstructorBadge({ status }: { status: string }) {
  const t = useTranslations('learning.instructorStatus');
  return <Badge tone={instructorTone(status)}>{t.has(status as 'pending') ? t(status as 'pending') : status}</Badge>;
}

export function PublicationBadge({ status }: { status: string }) {
  const t = useTranslations('learning.publication');
  return <Badge tone={publicationTone(status)}>{t.has(status as 'draft') ? t(status as 'draft') : status}</Badge>;
}

/** Completion as a figure and a thin data-colour meter. */
export function Completion({ percent, students }: { percent: number; students: number }) {
  const t = useTranslations('learning');
  const locale = useLocale();
  if (students === 0) return <span className="text-muted">—</span>;
  const clamped = Math.min(Math.max(percent, 0), 100);
  return (
    <span className="flex min-w-24 flex-col gap-1">
      <span className="tabular-nums">{t('percent', { value: formatNumber(clamped, locale) })}</span>
      <span className="meter" role="img" aria-label={t('percent', { value: formatNumber(clamped, locale) })}>
        <span style={{ inlineSize: `${clamped}%` }} />
      </span>
    </span>
  );
}

/** A modal with one note field (native <dialog>: focus trap + Esc), styled like the shared confirm dialog. */
export function NoteDialog({
  open,
  title,
  message,
  submitLabel,
  required,
  tone = 'primary',
  saving,
  onClose,
  onSubmit,
}: {
  open: boolean;
  title: string;
  message?: ReactNode;
  submitLabel: string;
  required: boolean;
  tone?: 'primary' | 'danger';
  saving: boolean;
  onClose: () => void;
  onSubmit: (note: string) => void;
}) {
  const t = useTranslations('learning.note');
  const tc = useTranslations('confirm');
  const ref = useRef<HTMLDialogElement>(null);
  const [note, setNote] = useState('');
  const [touched, setTouched] = useState(false);

  useEffect(() => {
    const dialog = ref.current;
    if (!dialog) return;
    if (open && !dialog.open) {
      setNote('');
      setTouched(false);
      dialog.showModal();
    }
    if (!open && dialog.open) dialog.close();
  }, [open]);

  const problem = noteError(note, required);
  const submit = (e: FormEvent) => {
    e.preventDefault();
    setTouched(true);
    if (problem || saving) return;
    onSubmit(note.trim());
  };

  return (
    <dialog
      ref={ref}
      className="dialog"
      aria-label={title}
      onCancel={(e) => {
        e.preventDefault();
        onClose();
      }}
    >
      {open ? (
        <form className="flex flex-col gap-4 p-5" onSubmit={submit} noValidate>
          <h2 className="m-0 text-base font-bold">{title}</h2>
          {message ? <p className="m-0 text-ink-3">{message}</p> : null}
          <TextArea
            label={required ? t('labelRequired') : t('label')}
            hint={t('hint', { max: NOTE_MAX })}
            required={required}
            rows={3}
            maxLength={NOTE_MAX + 50}
            value={note}
            onChange={(e) => setNote(e.target.value)}
            error={touched && problem ? t(`error.${problem}`, { max: NOTE_MAX }) : undefined}
            dir="auto"
          />
          <div className="mt-1 flex justify-end gap-2">
            <Button type="button" onClick={onClose}>
              {tc('cancel')}
            </Button>
            <Button type="submit" variant={tone} loading={saving}>
              {submitLabel}
            </Button>
          </div>
        </form>
      ) : null}
    </dialog>
  );
}

/** Approve / flag / unpublish buttons of one lesson (review queue and course outline). */
export function LessonActions({ lesson }: { lesson: { id: number; title: string; status: string; review: { state: string } } }) {
  const t = useTranslations('learning.lessonActions');
  const notifyError = useNotifyError();
  const mutation = useLessonAction();
  const [dialog, setDialog] = useState<Exclude<LessonAction, 'approve'> | null>(null);
  const can = lessonActions(lesson);

  const run = (action: LessonAction, note?: string) =>
    mutation.mutate(
      { id: lesson.id, action, note },
      {
        onSuccess: () => {
          toast.success(t(`${action}Toast`));
          setDialog(null);
        },
        onError: notifyError,
      },
    );

  return (
    <div className="row-actions" onClick={(e) => e.stopPropagation()}>
      {can.approve ? (
        <Button size="sm" variant="primary" onClick={() => run('approve')} loading={mutation.isPending && !dialog}>
          {t('approve')}
        </Button>
      ) : null}
      {can.flag ? (
        <Button size="sm" onClick={() => setDialog('flag')}>
          {t('flag')}
        </Button>
      ) : null}
      {can.unpublish ? (
        <Button size="sm" variant="danger" onClick={() => setDialog('unpublish')}>
          {t('unpublish')}
        </Button>
      ) : null}
      <NoteDialog
        open={dialog !== null}
        title={dialog ? t(`${dialog}Title`) : ''}
        message={dialog ? t(`${dialog}Message`, { title: lesson.title }) : undefined}
        submitLabel={dialog ? t(dialog) : ''}
        required
        tone={dialog === 'unpublish' ? 'danger' : 'primary'}
        saving={mutation.isPending}
        onClose={() => setDialog(null)}
        onSubmit={(note) => dialog && run(dialog, note)}
      />
    </div>
  );
}

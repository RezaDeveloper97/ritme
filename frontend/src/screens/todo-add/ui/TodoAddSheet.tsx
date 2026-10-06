'use client';

import { clsx } from 'clsx';
import { useLocale, useTranslations } from 'next-intl';
import { type FormEvent, useState } from 'react';

import {
  CATEGORY_LOOK,
  TODO_CATEGORIES,
  TODO_LIMITS,
  todoFieldErrors,
  useCreateTodoTask,
  useDeleteTodoTask,
  useTodoBoard,
  useTodoTask,
  useUpdateTodoTask,
  type TodoCategory,
  type TodoTask,
  type TodoTaskInput,
} from '@/entities/todo';
import type { Locale } from '@/shared/i18n';
import { addDays, formatDayMonth, formatNumber, fromApiDate, partsToDate, toApiDate, toParts, today } from '@/shared/lib/date';
import { AppSheet, closeSheet, type SheetContentProps } from '@/shared/sheet';
import { CalendarPicker, Icon, PrimaryButton, SecondaryButton, Skeleton, SkeletonGroup } from '@/shared/ui';

import { parseTodoAddArg } from '../model/arg';

/** Visually hidden: the sheet opens on the title input (nbl_Todo_Add has no heading); it still names the dialog. */
export function TodoAddTitle({ arg }: SheetContentProps) {
  const t = useTranslations('todo');
  const target = parseTodoAddArg(arg);
  return <span className="sr-only">{t(target.taskId ? 'add.editTitle' : 'add.title')}</span>;
}

/**
 * `?sheet=todo-add` (nbl_/nbd_Todo_Add): title, optional note, date chips
 * (امروز · فردا · تاریخ), time, category and reminder, and the round ✓.
 * `arg` = a category to preset (the board's active chip) or a task id to edit.
 */
export function TodoAddSheet({ arg }: SheetContentProps) {
  const t = useTranslations('todo');
  const target = parseTodoAddArg(arg);
  const existing = useTodoTask(target.taskId);
  if (target.taskId) {
    if (existing.isPending) {
      return (
        <SkeletonGroup label={t('common.loading')} className="td-form">
          <Skeleton width="medium" />
          <Skeleton width="short" />
          <Skeleton shape="block" />
        </SkeletonGroup>
      );
    }
    if (existing.isError) {
      return (
        <p className="td-error" role="alert">
          {t('add.notFound')}
        </p>
      );
    }
    return <TaskForm key={existing.data.task.id} task={existing.data.task} />;
  }
  return <TaskForm category={target.category ?? 'personal'} />;
}

function initial(task: TodoTask | undefined, category: TodoCategory): TodoTaskInput {
  if (task) {
    return { title: task.title, note: task.note, category: task.category, dueDate: task.dueDate, dueTime: task.dueTime, remind: task.remind };
  }
  return { title: '', note: null, category, dueDate: toApiDate(today()), dueTime: null, remind: false };
}

function TaskForm({ task, category = 'personal' }: { task?: TodoTask; category?: TodoCategory }) {
  const t = useTranslations('todo');
  const locale = useLocale() as Locale;
  const board = useTodoBoard();
  const create = useCreateTodoTask();
  const update = useUpdateTodoTask();
  const del = useDeleteTodoTask();
  const [form, setForm] = useState<TodoTaskInput>(() => initial(task, category));
  const [timeOpen, setTimeOpen] = useState(false);
  const [catOpen, setCatOpen] = useState(false);
  const [calOpen, setCalOpen] = useState(false);
  const [confirmDelete, setConfirmDelete] = useState(false);
  const [hint, setHint] = useState<string | null>(null);

  const todayIso = toApiDate(today());
  const tomorrowIso = toApiDate(addDays(today(), 1));
  const custom = form.dueDate !== null && form.dueDate !== todayIso && form.dueDate !== tomorrowIso;
  const saving = create.isPending || update.isPending;
  const err = create.error ?? update.error;
  const errors = err ? todoFieldErrors(err) : {};
  const errText = err ? (Object.values(errors)[0] ?? t('common.saveError')) : null;
  const look = CATEGORY_LOOK[form.category];

  const set = (patch: Partial<TodoTaskInput>) => {
    setHint(null);
    setForm((f) => {
      const next = { ...f, ...patch };
      // A reminder needs a date and a time (server rule): drop it when either goes.
      if (next.remind && (!next.dueDate || !next.dueTime)) next.remind = false;
      return next;
    });
  };
  const toggleDate = (iso: string) => set({ dueDate: form.dueDate === iso ? null : iso });

  const toggleRemind = () => {
    if (!form.remind && (!form.dueDate || !form.dueTime)) {
      setHint(t('add.remindNeedsTime'));
      if (!form.dueTime) setTimeOpen(true);
      return;
    }
    set({ remind: !form.remind });
  };

  const submit = (e: FormEvent) => {
    e.preventDefault();
    if (!form.title.trim() || saving) return;
    const done = { onSuccess: () => closeSheet() };
    if (task) update.mutate({ id: task.id, input: form }, done);
    else create.mutate(form, done);
  };

  return (
    <form className="td-form" onSubmit={submit} noValidate>
      <div className="td-form-title">
        <span className={clsx('td-check-box is-static', `nb-tone-${look.tone}`)} aria-hidden />
        <label className="sr-only" htmlFor="td-title">
          {t('add.titleLabel')}
        </label>
        <input
          id="td-title"
          className="td-title-input"
          value={form.title}
          maxLength={TODO_LIMITS.title}
          placeholder={t('add.titlePlaceholder')}
          onChange={(e) => set({ title: e.target.value })}
           
          autoFocus={!task}
          aria-invalid={errors.title ? true : undefined}
          enterKeyHint="done"
        />
      </div>
      <label className="sr-only" htmlFor="td-note">
        {t('add.note')}
      </label>
      <input
        id="td-note"
        className="td-note-input"
        value={form.note ?? ''}
        maxLength={TODO_LIMITS.note}
        placeholder={t('add.note')}
        onChange={(e) => set({ note: e.target.value || null })}
      />

      <div className="td-form-chips" role="group" aria-label={t('add.dateGroup')}>
        <button type="button" className="td-fchip nb-tone-brand" aria-pressed={form.dueDate === todayIso} onClick={() => toggleDate(todayIso)}>
          <Icon name="calendar" size={15} />
          {t('add.today')}
        </button>
        <button type="button" className="td-fchip nb-tone-brand" aria-pressed={form.dueDate === tomorrowIso} onClick={() => toggleDate(tomorrowIso)}>
          <Icon name="calendar" size={15} />
          {t('add.tomorrow')}
        </button>
        <button type="button" className="td-fchip nb-tone-brand" aria-pressed={custom} onClick={() => setCalOpen(true)}>
          <Icon name="calendar" size={15} />
          {custom && form.dueDate ? formatDayMonth(fromApiDate(form.dueDate), locale) : t('add.pickDate')}
        </button>
        <button
          type="button"
          className="td-fchip nb-tone-data"
          aria-pressed={!!form.dueTime}
          aria-expanded={timeOpen}
          onClick={() => setTimeOpen((o) => !o)}
        >
          <Icon name="clock" size={15} />
          {form.dueTime ? <bdi dir="ltr">{formatNumber(form.dueTime, locale)}</bdi> : t('add.time')}
        </button>
        <button
          type="button"
          className={clsx('td-fchip', `nb-tone-${look.tone}`)}
          aria-pressed
          aria-expanded={catOpen}
          onClick={() => setCatOpen((o) => !o)}
        >
          <Icon name={look.icon} size={15} />
          {t(`categories.${form.category}`)}
        </button>
        <button type="button" className="td-fchip nb-tone-brand" aria-pressed={form.remind} onClick={toggleRemind}>
          <Icon name="bell" size={15} />
          {t('add.remind')}
        </button>
      </div>

      {timeOpen ? (
        <div className="td-form-row">
          <label className="td-time-label">
            <span className="sr-only">{t('add.timeLabel')}</span>
            <input
              className="field td-time-input"
              type="time"
              dir="ltr"
              value={form.dueTime ?? ''}
              onChange={(e) => set({ dueTime: e.target.value || null })}
            />
          </label>
          {form.dueTime ? (
            <button type="button" className="td-link-btn" onClick={() => set({ dueTime: null })}>
              {t('add.clearTime')}
            </button>
          ) : null}
        </div>
      ) : null}

      {catOpen ? (
        <div className="td-form-chips" role="group" aria-label={t('add.categoryGroup')}>
          {TODO_CATEGORIES.map((c) => (
            <button
              key={c}
              type="button"
              className={clsx('td-fchip', `nb-tone-${CATEGORY_LOOK[c].tone}`)}
              aria-pressed={form.category === c}
              onClick={() => {
                set({ category: c });
                setCatOpen(false);
              }}
            >
              <Icon name={CATEGORY_LOOK[c].icon} size={15} />
              {t(`categories.${c}`)}
            </button>
          ))}
        </div>
      ) : null}

      {hint ? (
        <p className="td-hint" role="status">
          {hint}
        </p>
      ) : null}
      {form.remind && board.data && !board.data.remindersEnabled ? <p className="td-hint">{t('add.remindOff')}</p> : null}
      {errText ? (
        <p className="td-error" role="alert">
          {errText}
        </p>
      ) : null}

      <div className="td-form-foot">
        {task ? (
          confirmDelete ? (
            <div className="td-form-del">
              <span className="td-menu-q">{t('add.deleteConfirm')}</span>
              <SecondaryButton
                variant="text"
                block={false}
                className="td-danger-text"
                loading={del.isPending}
                onClick={() => del.mutate(task.id, { onSuccess: () => closeSheet() })}
              >
                {t('add.confirmDelete')}
              </SecondaryButton>
              <SecondaryButton variant="text" block={false} onClick={() => setConfirmDelete(false)}>
                {t('common.cancel')}
              </SecondaryButton>
            </div>
          ) : (
            <SecondaryButton variant="text" block={false} icon="trash" className="td-danger-text" onClick={() => setConfirmDelete(true)}>
              {t('add.delete')}
            </SecondaryButton>
          )
        ) : (
          <span />
        )}
        <button
          type="submit"
          className="td-save"
          aria-label={t('add.save')}
          disabled={!form.title.trim() || saving}
          aria-busy={saving || undefined}
        >
          <Icon name="check" size={22} strokeWidth={2.6} />
        </button>
      </div>

      <AppSheet open={calOpen} onClose={() => setCalOpen(false)} size="half" title={t('add.calendarTitle')}>
        <div className="td-cal">
          <CalendarPicker
            value={form.dueDate ? toParts(fromApiDate(form.dueDate), locale) : null}
            onSelect={(p) => {
              set({ dueDate: toApiDate(partsToDate(p, locale)) });
              setCalOpen(false);
            }}
          />
          <PrimaryButton
            className="td-cal-clear"
            onClick={() => {
              set({ dueDate: null });
              setCalOpen(false);
            }}
          >
            {t('add.clearDate')}
          </PrimaryButton>
        </div>
      </AppSheet>
    </form>
  );
}

'use client';

import { clsx } from 'clsx';
import { useLocale, useTranslations } from 'next-intl';
import type { ReactNode } from 'react';

import { Link, type Locale } from '@/shared/i18n';
import { formatDayMonth, formatNumber, fromApiDate } from '@/shared/lib/date';
import { Icon, type Tone } from '@/shared/ui';

import { CATEGORY_LOOK, doneKind, itemDueKind } from '../model/board';
import type { TodoCategory, TodoGroupKey, TodoItem, TodoTask } from '../model/types';

/** The 8px category dot of the chips and meta lines. */
export function CategoryDot({ category, className }: { category: TodoCategory; className?: string }) {
  return <span className={clsx('td-dot', `nb-tone-${CATEGORY_LOOK[category].tone}`, className)} aria-hidden />;
}

/**
 * The 26px round check of a task / item row inside a 44px hit target
 * (`aria-pressed`). Ticked = filled with the row's tone + a check.
 */
export function TodoCheck({
  done,
  tone,
  label,
  onToggle,
  disabled,
}: {
  done: boolean;
  tone: Tone;
  label: string;
  onToggle: () => void;
  disabled?: boolean;
}) {
  return (
    <button
      type="button"
      className={clsx('td-check', `nb-tone-${tone}`, done && 'is-done')}
      aria-pressed={done}
      aria-label={label}
      onClick={onToggle}
      disabled={disabled}
    >
      <span className="td-check-box" aria-hidden>
        {done ? <Icon name="check" size={14} strokeWidth={3} /> : null}
      </span>
    </button>
  );
}

/** One line of the meta: parts joined by « · ». */
function Meta({ category, parts, danger }: { category: TodoCategory; parts: ReactNode[]; danger?: boolean }) {
  const t = useTranslations('todo');
  return (
    <span className={clsx('td-meta', danger && 'is-danger')}>
      <CategoryDot category={category} />
      <span>
        {parts.map((p, i) => (
          <span key={i}>
            {i > 0 ? t('common.separator') : null}
            {p}
          </span>
        ))}
      </span>
    </span>
  );
}

/** The meta line of a task row: «۱۰:۰۰ · کار», «خرید · ۳ قلم», «بدون تاریخ · شخصی». */
export function useTaskMeta(task: TodoTask, group: TodoGroupKey | 'done'): { parts: ReactNode[]; danger: boolean } {
  const t = useTranslations('todo');
  const locale = useLocale() as Locale;
  const parts: ReactNode[] = [];
  let danger = false;
  if (group === 'done') {
    if (task.doneAt) parts.push(t('meta.doneOn', { date: formatDayMonth(fromApiDate(task.doneAt.slice(0, 10)), locale) }));
  } else if (task.overdue && task.dueDate) {
    parts.push(t('meta.overdue', { date: formatDayMonth(fromApiDate(task.dueDate), locale) }));
    danger = true;
  } else if (group === 'later') {
    parts.push(task.dueDate ? formatDayMonth(fromApiDate(task.dueDate), locale) : t('meta.noDate'));
  }
  if (task.dueTime && group !== 'done') parts.push(<bdi dir="ltr">{formatNumber(task.dueTime, locale)}</bdi>);
  parts.push(t(`categories.${task.category}`));
  // «خرید · ۳ قلم»: the items still to buy (all of them once the list is done).
  if (task.list) parts.push(t('meta.items', { count: task.list.open || task.list.total }));
  return { parts, danger };
}

interface TaskRowProps {
  task: TodoTask;
  group: TodoGroupKey | 'done';
  onToggle: (task: TodoTask) => void;
  /** A list opens its page; any other task calls `onEdit`. */
  href?: string;
  onEdit?: (task: TodoTask) => void;
}

/** A row of the board (nbl_Todo_Home): check, title (struck through when done), meta. */
export function TaskRow({ task, group, onToggle, href, onEdit }: TaskRowProps) {
  const t = useTranslations('todo');
  const { parts, danger } = useTaskMeta(task, group);
  const tone = CATEGORY_LOOK[task.category].tone;
  const body = (
    <>
      <span className="td-row-title">{task.title}</span>
      <Meta category={task.category} parts={parts} danger={danger} />
    </>
  );
  return (
    <li className={clsx('td-row', task.done && 'is-done')}>
      <TodoCheck
        done={task.done}
        tone={tone}
        label={t(task.done ? 'check.unmark' : 'check.mark', { title: task.title })}
        onToggle={() => onToggle(task)}
      />
      {href ? (
        <Link href={href} className="td-row-main" aria-label={t('openTask', { title: task.title })}>
          {body}
        </Link>
      ) : (
        <button type="button" className="td-row-main" onClick={() => onEdit?.(task)} aria-label={t('editTask', { title: task.title })}>
          {body}
        </button>
      )}
    </li>
  );
}

interface ItemRowProps {
  item: TodoItem;
  category: TodoCategory;
  todayIso: string;
  onToggle: (item: TodoItem) => void;
  onDelete: (item: TodoItem) => void;
  busy?: boolean;
}

/** A row of a list (nbl_Todo_List): «امروز», «قبل از ۱۷ مهر», done «دیروز». */
export function ItemRow({ item, category, todayIso, onToggle, onDelete, busy }: ItemRowProps) {
  const t = useTranslations('todo');
  const locale = useLocale() as Locale;
  const parts: ReactNode[] = [];
  let danger = false;
  if (item.done) {
    const k = doneKind(item.doneAt, todayIso);
    if (k === 'today' || k === 'yesterday') parts.push(t(`list.doneWhen.${k}`));
    else if (k === 'date' && item.doneAt) parts.push(formatDayMonth(fromApiDate(item.doneAt.slice(0, 10)), locale));
  } else {
    const k = itemDueKind(item.dueDate, todayIso);
    if (k && item.dueDate) {
      const date = formatDayMonth(fromApiDate(item.dueDate), locale);
      parts.push(k === 'today' || k === 'tomorrow' ? t(`list.due.${k}`) : t(`list.due.${k}`, { date }));
      danger = k === 'past';
    }
  }
  return (
    <li className={clsx('td-row', item.done && 'is-done')}>
      <TodoCheck
        done={item.done}
        tone={CATEGORY_LOOK[category].tone}
        label={t(item.done ? 'check.unmark' : 'check.mark', { title: item.title })}
        onToggle={() => onToggle(item)}
      />
      <span className="td-row-main is-static">
        <span className="td-row-title">{item.title}</span>
        {parts.length ? <Meta category={category} parts={parts} danger={danger} /> : null}
      </span>
      <button
        type="button"
        className="td-row-del"
        aria-label={t('list.deleteItem', { title: item.title })}
        onClick={() => onDelete(item)}
        disabled={busy}
      >
        <Icon name="x" size={16} />
      </button>
    </li>
  );
}

'use client';

import { clsx } from 'clsx';
import { useLocale, useTranslations } from 'next-intl';
import { useState } from 'react';

import {
  CATEGORY_LOOK,
  CategoryDot,
  TODO_CATEGORIES,
  TaskRow,
  filterGroups,
  groupsProgress,
  isFreshBoard,
  opensList,
  useAcceptTodoSuggestion,
  useDismissTodoSuggestion,
  useTodoBoard,
  useToggleTodoTask,
  type TodoFilter,
  type TodoGroup,
  type TodoSuggestion,
  type TodoTask,
} from '@/entities/todo';
import type { Locale } from '@/shared/i18n';
import { formatWeekdayDayMonth, today } from '@/shared/lib/date';
import { openSheet } from '@/shared/sheet';
import {
  Accordion,
  Card,
  EmptyState,
  HubHeader,
  Icon,
  PrimaryButton,
  ProgressRing,
  Skeleton,
  SkeletonGroup,
  SkyLayer,
} from '@/shared/ui';
import { BottomNav } from '@/widgets/bottom-nav';

const FILTERS: readonly TodoFilter[] = ['all', ...TODO_CATEGORIES];

/** The add sheet; `arg` = a preset category (from the active chip) or a task id to edit. */
function openAdd(arg?: string) {
  openSheet('todo-add', arg);
}

/**
 * `/todo` — «کارهای من» (nbl_/nbd_Todo_Home, Todo_Empty): progress ring,
 * category chips, the cycle suggestion, today / tomorrow / later groups (a
 * ticked task stays struck through in its group), the recently done and the
 * «کار جدید» CTA. A hub: the bottom nav shows («من» tab).
 */
export function TodoPage() {
  const t = useTranslations('todo');
  const locale = useLocale() as Locale;
  const board = useTodoBoard();
  const toggle = useToggleTodoTask();
  const [filter, setFilter] = useState<TodoFilter>('all');

  const onToggle = (task: TodoTask) => toggle.mutate({ id: task.id, done: !task.done });
  const onEdit = (task: TodoTask) => openAdd(String(task.id));

  let body;
  let ring = null;
  if (board.isPending) {
    body = (
      <SkeletonGroup label={t('common.loading')} className="td-body">
        <Skeleton shape="block" className="td-skel-suggest" />
        <Skeleton width="short" />
        <Skeleton shape="block" />
        <Skeleton shape="block" />
        <Skeleton shape="block" />
      </SkeletonGroup>
    );
  } else if (board.isError) {
    body = (
      <Card>
        <EmptyState
          icon="warning"
          title={t('common.loadError')}
          action={
            <PrimaryButton icon="refresh" loading={board.isFetching} onClick={() => void board.refetch()}>
              {t('common.retry')}
            </PrimaryButton>
          }
        />
      </Card>
    );
  } else {
    const b = board.data;
    const groups = filterGroups(b.groups, filter);
    const progress = groupsProgress(groups);
    const done = filter === 'all' ? b.done : b.done.filter((x) => x.category === filter);
    const empty = groups.every((g) => g.tasks.length === 0);
    const fresh = isFreshBoard(b);
    ring = progress.total ? (
      <ProgressRing
        value={progress.done / progress.total}
        size={56}
        thickness={6}
        label={t('progress', { done: progress.done, total: progress.total })}
        className="td-ring"
      >
        <span className="td-ring-text">{t('progressShort', { done: progress.done, total: progress.total })}</span>
      </ProgressRing>
    ) : null;
    body = (
      <div className="td-body">
        {b.suggestion ? <SuggestionCard suggestion={b.suggestion} /> : null}
        {empty ? (
          <EmptyState
            icon={filter === 'all' ? 'todo' : CATEGORY_LOOK[filter].icon}
            className={clsx('td-empty', `nb-tone-${filter === 'all' ? 'brand' : CATEGORY_LOOK[filter].tone}`)}
            title={t(fresh ? 'empty.freshTitle' : 'empty.allDoneTitle')}
            body={
              fresh
                ? t('empty.freshBody')
                : filter === 'all'
                  ? t('empty.allDoneBodyAll')
                  : t('empty.allDoneBody', { category: t(`categories.${filter}`) })
            }
            action={
              <PrimaryButton icon="plus" onClick={() => openAdd(filter === 'all' ? undefined : filter)}>
                {t('newTask')}
              </PrimaryButton>
            }
          />
        ) : (
          <>
            {groups.map((g) => (g.tasks.length ? <GroupSection key={g.key} group={g} onToggle={onToggle} onEdit={onEdit} /> : null))}
            <PrimaryButton icon="plus" className="td-cta" onClick={() => openAdd(filter === 'all' ? undefined : filter)}>
              {t('newTask')}
            </PrimaryButton>
          </>
        )}
        {done.length ? (
          <Accordion title={t('doneSection', { count: done.length })} icon="checkCircle" tone="success" className="td-done-acc">
            <ul className="td-rows">
              {done.map((task) => (
                <TaskRow
                  key={task.id}
                  task={task}
                  group="done"
                  onToggle={onToggle}
                  href={opensList(task) ? `/todo/lists/${task.id}` : undefined}
                  onEdit={onEdit}
                />
              ))}
            </ul>
          </Accordion>
        ) : null}
      </div>
    );
  }

  return (
    <div className="view td-screen">
      <SkyLayer />
      <div className="scroll td-scroll">
        <HubHeader date={formatWeekdayDayMonth(today(), locale)} greeting={t('title')} actions={ring} className="td-hub" />
        <div className="td-chips" role="group" aria-label={t('filters.label')}>
          {FILTERS.map((f) => (
            <button
              key={f}
              type="button"
              aria-pressed={filter === f}
              className={clsx('td-chip', `nb-tone-${f === 'all' ? 'brand' : CATEGORY_LOOK[f].tone}`)}
              onClick={() => setFilter(f)}
            >
              {f === 'all' ? <span className="td-dot nb-tone-brand" aria-hidden /> : <CategoryDot category={f} />}
              {f === 'all' ? t('filters.all') : t(`categories.${f}`)}
            </button>
          ))}
        </div>
        {body}
      </div>
      <BottomNav />
    </div>
  );
}

function GroupSection({ group, onToggle, onEdit }: { group: TodoGroup; onToggle: (t: TodoTask) => void; onEdit: (t: TodoTask) => void }) {
  const t = useTranslations('todo');
  const id = `td-g-${group.key}`;
  return (
    <section className="td-group" aria-labelledby={id}>
      <div className="td-group-head">
        <h2 id={id} className="td-group-title">
          {t(`groups.${group.key}`)}
        </h2>
        <span className="td-group-count">{t('count', { count: group.tasks.length })}</span>
      </div>
      <ul className="td-rows">
        {group.tasks.map((task) => (
          <TaskRow
            key={task.id}
            task={task}
            group={group.key}
            onToggle={onToggle}
            href={opensList(task) ? `/todo/lists/${task.id}` : undefined}
            onEdit={onEdit}
          />
        ))}
      </ul>
    </section>
  );
}

/** «پریودت ۵ روز دیگر است؛ «خرید نوار بهداشتی» را اضافه کنم؟ · افزودن» — admin copy, plain text. */
function SuggestionCard({ suggestion }: { suggestion: TodoSuggestion }) {
  const t = useTranslations('todo');
  const accept = useAcceptTodoSuggestion();
  const dismiss = useDismissTodoSuggestion();
  return (
    <aside className="td-suggest" aria-label={t('suggestion.label')}>
      <Icon name="sparkle" size={18} className="td-suggest-icon" />
      <p className="td-suggest-text">{suggestion.prompt}</p>
      <button
        type="button"
        className="td-suggest-add"
        onClick={() => accept.mutate(suggestion.key)}
        disabled={accept.isPending}
        aria-busy={accept.isPending || undefined}
      >
        {suggestion.action}
      </button>
      <button
        type="button"
        className="td-suggest-x"
        aria-label={t('suggestion.dismiss')}
        onClick={() => dismiss.mutate(suggestion.key)}
        disabled={dismiss.isPending}
      >
        <Icon name="x" size={16} />
      </button>
      {accept.isError ? (
        <p className="td-suggest-err" role="alert">
          {t('suggestion.error')}
        </p>
      ) : null}
    </aside>
  );
}

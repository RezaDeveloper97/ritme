'use client';

import { clsx } from 'clsx';
import { useTranslations } from 'next-intl';
import { type FormEvent, useState } from 'react';

import {
  CATEGORY_LOOK,
  ItemRow,
  TODO_LIMITS,
  isTodoNotFound,
  todoFieldErrors,
  useAddTodoItem,
  useDeleteTodoItem,
  useDeleteTodoTask,
  useTodoTask,
  useToggleTodoItem,
  type TodoItem,
  type TodoTask,
} from '@/entities/todo';
import { useRouter } from '@/shared/i18n';
import { toApiDate, today } from '@/shared/lib/date';
import { AppSheet, openSheet } from '@/shared/sheet';
import {
  Accordion,
  Card,
  EmptyState,
  HeaderButton,
  Icon,
  PrimaryButton,
  ScreenHeader,
  SecondaryButton,
  Skeleton,
  SkeletonGroup,
  SkyLayer,
} from '@/shared/ui';

/**
 * `/todo/lists/[id]` — one list (nbl_/nbd_Todo_List): «خرید · ۳ از ۵ مانده»,
 * the big title with the category icon, open items, «افزودن قلم» and the
 * «انجام‌شده» accordion. The ⋯ menu edits the task (the add sheet) or deletes
 * the list. A back-header screen: no bottom nav.
 */
export function TodoListPage({ id }: { id: number }) {
  const t = useTranslations('todo');
  const router = useRouter();
  const query = useTodoTask(id);
  const [menuOpen, setMenuOpen] = useState(false);
  const back = () => router.push('/todo');

  const task = query.data?.task;
  const header = (
    <ScreenHeader
      title={task ? t(`categories.${task.category}`) : t('title')}
      subtitle={task?.list ? t('list.remaining', { open: task.list.open, total: task.list.total }) : undefined}
      onBack={back}
      backLabel={t('common.back')}
      action={task ? <HeaderButton label={t('list.more')} icon="moreH" onClick={() => setMenuOpen(true)} /> : undefined}
    />
  );

  let body;
  if (id > 0 && query.isPending) {
    body = (
      <SkeletonGroup label={t('common.loading')} className="td-body">
        <Skeleton width="medium" />
        <Skeleton shape="block" />
        <Skeleton shape="block" />
        <Skeleton shape="block" />
      </SkeletonGroup>
    );
  } else if (id <= 0 || query.isError) {
    const gone = id <= 0 || isTodoNotFound(query.error);
    body = (
      <Card>
        <EmptyState
          icon={gone ? 'todo' : 'warning'}
          title={gone ? t('list.notFound') : t('common.loadError')}
          action={
            gone ? (
              <PrimaryButton onClick={back}>{t('list.backToTodo')}</PrimaryButton>
            ) : (
              <PrimaryButton icon="refresh" loading={query.isFetching} onClick={() => void query.refetch()}>
                {t('common.retry')}
              </PrimaryButton>
            )
          }
        />
      </Card>
    );
  } else if (query.data) {
    body = <ListBody task={query.data.task} items={query.data.items} />;
  }

  return (
    <div className="view td-screen td-list-screen">
      <SkyLayer />
      <div className="scroll td-scroll">
        {header}
        {body}
      </div>
      {task ? <ListMenu task={task} open={menuOpen} onClose={() => setMenuOpen(false)} onDeleted={back} /> : null}
    </div>
  );
}

function ListBody({ task, items }: { task: TodoTask; items: TodoItem[] }) {
  const t = useTranslations('todo');
  const todayIso = toApiDate(today());
  const toggle = useToggleTodoItem(task.id);
  const del = useDeleteTodoItem(task.id);
  const look = CATEGORY_LOOK[task.category];
  const open = items.filter((i) => !i.done);
  const done = items.filter((i) => i.done);
  const row = (item: TodoItem) => (
    <ItemRow
      key={item.id}
      item={item}
      category={task.category}
      todayIso={todayIso}
      onToggle={(it) => toggle.mutate({ id: it.id, done: !it.done })}
      onDelete={(it) => del.mutate(it.id)}
      busy={del.isPending}
    />
  );
  return (
    <div className="td-body">
      <div className="td-list-title">
        <span className={clsx('td-list-icon', `nb-tone-${look.tone}`)} aria-hidden>
          <Icon name={look.icon} size={22} />
        </span>
        <h2 className="td-list-name">{task.title}</h2>
      </div>
      {task.note ? <p className="td-list-note">{task.note}</p> : null}
      {open.length ? <ul className="td-rows">{open.map(row)}</ul> : items.length ? null : <p className="td-list-empty">{t('list.empty')}</p>}
      <AddItemForm taskId={task.id} />
      {done.length ? (
        <Accordion title={t('list.doneSection', { count: done.length })} icon="checkCircle" tone="success" defaultOpen className="td-done-acc">
          <ul className="td-rows">{done.map(row)}</ul>
        </Accordion>
      ) : null}
    </div>
  );
}

/** «+ افزودن قلم»: a row that turns into an input; Enter adds and keeps it open for the next item. */
function AddItemForm({ taskId }: { taskId: number }) {
  const t = useTranslations('todo');
  const add = useAddTodoItem(taskId);
  const [editing, setEditing] = useState(false);
  const [title, setTitle] = useState('');
  const error = add.isError ? (todoFieldErrors(add.error).title ?? t('common.saveError')) : null;

  const submit = (e: FormEvent) => {
    e.preventDefault();
    if (!title.trim() || add.isPending) return;
    add.mutate({ title }, { onSuccess: () => setTitle('') });
  };

  if (!editing) {
    return (
      <button type="button" className="td-additem" onClick={() => setEditing(true)}>
        <Icon name="plus" size={20} />
        {t('list.addItem')}
      </button>
    );
  }
  return (
    <form className="td-additem-form" onSubmit={submit}>
      <label className="sr-only" htmlFor="td-item-title">
        {t('list.itemPlaceholder')}
      </label>
      <input
        id="td-item-title"
        className="td-additem-input"
        value={title}
        maxLength={TODO_LIMITS.title}
        placeholder={t('list.itemPlaceholder')}
        onChange={(e) => setTitle(e.target.value)}
         
        autoFocus
        enterKeyHint="done"
      />
      <button type="submit" className="td-additem-save" disabled={!title.trim() || add.isPending} aria-busy={add.isPending || undefined}>
        {t('list.addItemSave')}
      </button>
      {error ? (
        <p className="td-error" role="alert">
          {error}
        </p>
      ) : null}
    </form>
  );
}

function ListMenu({ task, open, onClose, onDeleted }: { task: TodoTask; open: boolean; onClose: () => void; onDeleted: () => void }) {
  const t = useTranslations('todo');
  const del = useDeleteTodoTask();
  const [confirm, setConfirm] = useState(false);
  const close = () => {
    setConfirm(false);
    onClose();
  };
  return (
    <AppSheet open={open} onClose={close} size="half" title={task.title}>
      <div className="td-menu">
        {confirm ? (
          <>
            <p className="td-menu-q">{t('list.deleteConfirm')}</p>
            <PrimaryButton
              className="td-danger-btn"
              icon="trash"
              loading={del.isPending}
              onClick={() => del.mutate(task.id, { onSuccess: onDeleted })}
            >
              {t('add.confirmDelete')}
            </PrimaryButton>
            <SecondaryButton onClick={() => setConfirm(false)}>{t('common.cancel')}</SecondaryButton>
            {del.isError ? (
              <p className="td-error" role="alert">
                {t('common.saveError')}
              </p>
            ) : null}
          </>
        ) : (
          <>
            <button
              type="button"
              className="td-menu-row"
              onClick={() => {
                close();
                openSheet('todo-add', String(task.id));
              }}
            >
              <Icon name="pencil" size={20} />
              {t('list.edit')}
            </button>
            <button type="button" className="td-menu-row is-danger" onClick={() => setConfirm(true)}>
              <Icon name="trash" size={20} />
              {t('list.deleteList')}
            </button>
          </>
        )}
      </div>
    </AppSheet>
  );
}

'use client';

import type { ReactNode } from 'react';
import { useTranslations } from 'next-intl';

import { useModuleEnabled } from '@/entities/app-module';
import { useTodoBoard } from '@/entities/todo';
import { useRouter } from '@/shared/i18n';
import { pilot } from '@/shared/pilot';
import { ListGroup, ListRow, SectionTitle } from '@/shared/ui';

import { openTodayTasks } from '../model/todo-count';

/**
 * Today «برای امروز» (CB-NAV-03, nbd_Nav_Today / IA_Map): one section for what today asks of her —
 * `reminders` is the M3 «یادآورهای امروز» card the screen composes in (a widget can't import a widget), then
 * the to-do count (B-N6-08 → `/todo`, only when tasks are due today; hidden when the list API is missing, e.g.
 * production on Laravel). The companion suggestion row of the board («… پیشنهاد داد · از همدم») has no API in
 * bloom N4 yet — hidden until it exists. The admin «یادآورها» / «لیست کارها» module switches drop their part here,
 * so every home that composes this section follows them.
 */
export function ForToday({ reminders: remindersCard }: { reminders?: ReactNode }) {
  const t = useTranslations('nav.forToday');
  const router = useRouter();
  const remindersOn = useModuleEnabled('reminders');
  const todoOn = useModuleEnabled('todo');
  const board = useTodoBoard(todoOn);
  const reminders = remindersOn ? remindersCard : null;
  const todo = todoOn && board.isSuccess ? openTodayTasks(board.data) : 0;
  if (!reminders && todo === 0) return null;

  return (
    <section className="ftd" aria-labelledby="ftd-title">
      <SectionTitle id="ftd-title" title={t('title')} className="ftd-title" />
      {reminders}
      {todo > 0 ? (
        <ListGroup className="ftd-rows">
          <ListRow
            icon="todo"
            iconTone="brand"
            title={t('todo', { count: todo })}
            description={t('todoSub')}
            onClick={() => router.push('/todo')}
            pilot={pilot('home.today-tasks')}
          />
        </ListGroup>
      ) : null}
    </section>
  );
}

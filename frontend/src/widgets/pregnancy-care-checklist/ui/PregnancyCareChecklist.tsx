'use client';

import clsx from 'clsx';
import { useTranslations } from 'next-intl';

import { taskProgress, toggleDoneKey, type WeekTask } from '@/entities/pregnancy';
import { useUpdateWeekState } from '@/features/track-pregnancy';
import { Card, Icon } from '@/shared/ui';

interface Props {
  week: number;
  tasks: readonly WeekTask[];
  /** Section title; «مراقبت‌های این هفته» by default. */
  title?: string;
  /**
   * `today` (PregFull_Main) shows the «۲ از ۳» count and an «انجام شد» pill on
   * done rows; `week` (PregFull_Week «کارهای این هفته») keeps just the discs.
   */
  variant?: 'today' | 'week';
}

/**
 * The week's care tasks as checkbox rows (optimistic PUT; Today and Week share
 * one mutation). Each row: 34px disc (turquoise check when done, a quiet plus
 * when not) + text, ≥ 52px tall.
 */
export function PregnancyCareChecklist({ week, tasks, title, variant = 'today' }: Props) {
  const t = useTranslations('pregnancyV2');
  const update = useUpdateWeekState();
  const { done, total } = taskProgress(tasks);
  const headingId = `pgn-care-${variant}`;

  const toggle = (key: string) => {
    const doneKeys = tasks.filter((x) => x.done).map((x) => x.key);
    update.mutate({ week, state: { done_task_keys: toggleDoneKey(doneKeys, key) } });
  };

  return (
    <Card as="section" className="pgn-sect" aria-labelledby={headingId}>
      <div className="pgn-sect-head">
        <h2 id={headingId} className="pgn-sect-title">
          {title ?? t('today.careTitle')}
        </h2>
        {variant === 'today' && total > 0 && (
          <span className="pgn-opill nb-tone-data">{t('today.careDone', { done, total })}</span>
        )}
      </div>
      {total === 0 ? (
        <p className="pgn-muted">{t('today.careEmpty')}</p>
      ) : (
        <ul className="pgn-rows">
          {tasks.map((task) => (
            <li key={task.key}>
              <button
                type="button"
                role="checkbox"
                aria-checked={task.done}
                onClick={() => toggle(task.key)}
                className="pgn-row pgn-task"
              >
                <span className={clsx('pgn-task-disc', task.done && 'is-done')} aria-hidden>
                  <Icon name={task.done ? 'check' : 'plus'} size={16} strokeWidth={task.done ? 2.6 : 2} />
                </span>
                <span className="pgn-row-title">{task.text}</span>
                {variant === 'today' && task.done && (
                  <span className="pgn-opill nb-tone-data" aria-hidden>
                    {t('calendar.states.done')}
                  </span>
                )}
              </button>
            </li>
          ))}
        </ul>
      )}
      {update.isError && (
        <p role="alert" className="pgn-error-text">
          {t('common.saveError')}
        </p>
      )}
    </Card>
  );
}

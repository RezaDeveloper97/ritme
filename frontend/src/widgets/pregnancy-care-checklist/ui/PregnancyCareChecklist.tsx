'use client';

import clsx from 'clsx';
import { useTranslations } from 'next-intl';

import { taskProgress, toggleDoneKey, type WeekTask } from '@/entities/pregnancy';
import { useUpdateWeekState } from '@/features/track-pregnancy';
import { Icon } from '@/shared/ui';

interface Props {
  week: number;
  tasks: readonly WeekTask[];
}

/** The square brand box both task lists use (Today and the Week «کارهای هفته» tab). */
export function PregnancyCheckBox({ on }: { on: boolean }) {
  return (
    <span className={clsx('pg2-check', on && 'is-on')} aria-hidden>
      <Icon name="check" size={15} strokeWidth={3} />
    </span>
  );
}

/** «مراقبت‌های این هفته» — the current week's tasks with a done count (optimistic PUT). */
export function PregnancyCareChecklist({ week, tasks }: Props) {
  const t = useTranslations('pregnancyV2');
  const update = useUpdateWeekState();
  const { done, total } = taskProgress(tasks);

  const toggle = (key: string) => {
    const doneKeys = tasks.filter((x) => x.done).map((x) => x.key);
    update.mutate({ week, state: { done_task_keys: toggleDoneKey(doneKeys, key) } });
  };

  return (
    <section className="card mx-4 p-4">
      <div className="flex items-center justify-between">
        <h2 className="text-[15px] font-black text-(--ink)">{t('today.careTitle')}</h2>
        {total > 0 && (
          <span className="rounded-full bg-(--surface-2) px-2.5 py-0.5 text-[11.5px] font-black text-(--brand-deep)">
            {t('today.careDone', { done, total })}
          </span>
        )}
      </div>
      {total === 0 ? (
        <p className="mt-3 text-[13px] text-(--muted)">{t('today.careEmpty')}</p>
      ) : (
        <ul className="mt-2">
          {tasks.map((task) => (
            <li key={task.key} className="border-b border-(--line-2) last:border-b-0">
              <button
                type="button"
                role="checkbox"
                aria-checked={task.done}
                onClick={() => toggle(task.key)}
                className="flex w-full items-center gap-3 py-3 text-start focus-visible:shadow-(--ring) focus-visible:outline-none"
              >
                <PregnancyCheckBox on={task.done} />
                <span
                  className={clsx('text-[13.5px] font-bold', task.done ? 'text-(--muted) line-through' : 'text-(--ink)')}
                >
                  {task.text}
                </span>
              </button>
            </li>
          ))}
        </ul>
      )}
      {update.isError && (
        <p role="alert" className="mt-2 text-[12px] font-bold text-(--care-rose)">
          {t('common.saveError')}
        </p>
      )}
    </section>
  );
}

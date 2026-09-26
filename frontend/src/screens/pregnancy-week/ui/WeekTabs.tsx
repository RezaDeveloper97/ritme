'use client';

import clsx from 'clsx';
import { useTranslations } from 'next-intl';
import { useId, useState } from 'react';

import {
  highlightGlyph,
  type PregnancyWeek,
  toggleDoneKey,
} from '@/entities/pregnancy';
import { useUpdateWeekState } from '@/features/track-pregnancy';
import { Link, useDirection } from '@/shared/i18n';
import { Icon } from '@/shared/ui';

const TABS = ['baby', 'body', 'tasks'] as const;
type Tab = (typeof TABS)[number];

const CARD = 'rounded-2xl border border-(--line) bg-(--surface)';
const MUTED = 'text-[13px] leading-[1.9] text-(--ink-3)';

/** Segmented tabs جنین / بدن تو / کارهای هفته and their panels. */
export function WeekTabs({ data }: { data: PregnancyWeek }) {
  const t = useTranslations('pregnancyV2.week');
  const [tab, setTab] = useState<Tab>('baby');
  const base = useId();

  return (
    <>
      <div
        role="tablist"
        aria-label={t('tabsLabel')}
        className="flex gap-1 rounded-[14px] bg-(--violet-soft) p-1"
      >
        {TABS.map((key) => {
          const on = key === tab;
          return (
            <button
              key={key}
              type="button"
              role="tab"
              id={`${base}-tab-${key}`}
              aria-selected={on}
              aria-controls={`${base}-panel-${key}`}
              onClick={() => setTab(key)}
              className={clsx(
                'h-11 grow rounded-[11px] border-none text-[13px] font-extrabold focus-visible:shadow-(--ring) focus-visible:outline-none',
                on ? 'bg-(--surface) text-(--brand-deep) shadow-sm' : 'bg-transparent text-(--ink-3)',
              )}
            >
              {t(`tabs.${key}`)}
            </button>
          );
        })}
      </div>
      <section
        role="tabpanel"
        id={`${base}-panel-${tab}`}
        aria-labelledby={`${base}-tab-${tab}`}
        className={CARD}
      >
        {tab === 'baby' && <BabyPanel data={data} />}
        {tab === 'body' && <BodyPanel data={data} />}
        {tab === 'tasks' && <TasksPanel data={data} />}
      </section>
    </>
  );
}

function Empty() {
  const t = useTranslations('pregnancyV2.week');
  return <p className={clsx('m-0 px-3.5 py-4', MUTED)}>{t('empty')}</p>;
}

function BabyPanel({ data }: { data: PregnancyWeek }) {
  const items = data.details.highlights;
  if (items.length === 0) return <Empty />;
  return (
    <ul className="m-0 flex list-none flex-col gap-3.5 px-3.5 py-4">
      {items.map((h, i) => (
        <li key={`${h.title}-${i}`} className="flex gap-3">
          <span className={clsx('pg2-tile size-9 rounded-xl', `pg2-tone-${h.tone}`)} aria-hidden>
            <Icon name={highlightGlyph(h.icon)} size={18} />
          </span>
          <div className="min-w-0">
            <b className="text-sm text-(--ink)">{h.title}</b>
            <p className={clsx('mt-0.5 mb-0', MUTED)}>{h.body}</p>
          </div>
        </li>
      ))}
    </ul>
  );
}

function BodyPanel({ data }: { data: PregnancyWeek }) {
  const t = useTranslations('pregnancyV2.week');
  const dir = useDirection();
  const { bodySymptoms, bodyText } = data.details;
  if (bodySymptoms.length === 0 && !bodyText) return <Empty />;
  return (
    <div className="flex flex-col gap-2.5 px-3.5 py-4">
      {bodySymptoms.length > 0 && (
        <>
          <b className="text-[14.5px] text-(--ink)">{t('bodyTitle')}</b>
          <ul className="m-0 flex list-none flex-wrap gap-2 p-0">
            {bodySymptoms.map((s) => (
              <li
                key={s}
                className="inline-flex h-9 items-center rounded-xl bg-(--surface-2) px-3 text-[12.5px] font-bold text-(--ink)"
              >
                {s}
              </li>
            ))}
          </ul>
        </>
      )}
      {bodyText && <p className={clsx('m-0', MUTED)}>{bodyText}</p>}
      <Link
        href="/pregnancy/log"
        className="inline-flex h-11 items-center gap-1.5 self-start text-[13px] font-extrabold text-(--brand) no-underline focus-visible:shadow-(--ring) focus-visible:outline-none"
      >
        {t('logToday')}
        <Icon name={dir === 'rtl' ? 'chevronLeft' : 'chevronRight'} size={16} />
      </Link>
    </div>
  );
}

/** Same list as Today's «مراقبت‌های این هفته» — one mutation updates both. */
function TasksPanel({ data }: { data: PregnancyWeek }) {
  const tc = useTranslations('pregnancyV2.today');
  const tCommon = useTranslations('pregnancyV2.common');
  const update = useUpdateWeekState();
  if (data.tasks.length === 0) {
    return <p className={clsx('m-0 px-3.5 py-4', MUTED)}>{tc('careEmpty')}</p>;
  }
  const doneKeys = data.tasks.filter((x) => x.done).map((x) => x.key);
  return (
    <div className="flex flex-col px-3.5 py-1">
      {data.tasks.map((task, i) => (
        <label
          key={task.key}
          className={clsx(
            'flex min-h-12 cursor-pointer items-center gap-3 text-[13.5px] font-semibold text-(--ink)',
            i > 0 && 'border-t border-(--line-2)',
          )}
        >
          <input
            type="checkbox"
            checked={task.done}
            onChange={() =>
              update.mutate({ week: data.week, state: { done_task_keys: toggleDoneKey(doneKeys, task.key) } })
            }
            className="size-[22px] shrink-0 accent-(--brand-fill)"
          />
          <span className={clsx(task.done && 'text-(--muted) line-through')}>{task.text}</span>
        </label>
      ))}
      {update.isError && (
        <p role="alert" className="m-0 pb-2 text-xs font-semibold text-(--danger)">
          {tCommon('saveError')}
        </p>
      )}
    </div>
  );
}

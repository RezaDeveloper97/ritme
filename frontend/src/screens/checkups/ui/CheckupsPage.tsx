'use client';

import clsx from 'clsx';
import { useLocale, useTranslations } from 'next-intl';
import { type KeyboardEvent, useRef, useState } from 'react';

import {
  CHECKUP_LIST_FILTERS,
  type CheckupItem,
  type CheckupListFilter,
  type CheckupSummary,
  checkupIcon,
  useCheckups,
} from '@/entities/checkup';
import { type Locale, Link, useDirection, useRouter } from '@/shared/i18n';
import { formatLongDate, formatNumber, fromApiDate } from '@/shared/lib/date';
import { Icon } from '@/shared/ui';

import {
  barSegments,
  filterHref,
  groupBySection,
  nextFilter,
  parseFilter,
  worstStatus,
} from '../model/view';
import { PlanSettingsSheet } from './PlanSettingsSheet';

const PANEL_ID = 'ck-panel';

function StatusPill({ status, label }: { status: string; label: string }) {
  return (
    <span
      className={clsx(
        `ck-status-${status}`,
        'shrink-0 rounded-full bg-(--ck-soft) px-2.5 py-1 text-[11px] font-extrabold text-(--ck-ink)',
      )}
    >
      {label}
    </span>
  );
}

function SummaryCard({ summary }: { summary: CheckupSummary }) {
  const t = useTranslations('checkups');
  const locale = useLocale() as Locale;
  const num = (n: number) => formatNumber(n, locale);
  const worst = worstStatus(summary);
  const segments = barSegments(summary);
  const legend: [keyof CheckupSummary, 'up_to_date' | 'due' | 'overdue'][] = [
    ['upToDate', 'up_to_date'],
    ['due', 'due'],
    ['overdue', 'overdue'],
  ];

  return (
    <section className="card flex flex-col gap-3 p-4" aria-label={t('list.statusTitle')}>
      <div className="flex items-start justify-between gap-3">
        <div className="text-start">
          <p className="text-[12px] font-semibold text-(--ink-3)">{t('list.statusTitle')}</p>
          <p className="mt-1 font-['Lalezar'] text-[26px] leading-tight font-normal text-(--ink)">
            {t('list.upToDateOf', { upToDate: num(summary.upToDate), total: num(summary.total) })}
          </p>
        </div>
        <StatusPill status={worst} label={t(`status.${worst}`)} />
      </div>
      <div dir="ltr" className="flex h-2.5 w-full overflow-hidden rounded-full bg-(--track)" aria-hidden>
        {segments.map((s) => (
          <span
            key={s.status}
            className={clsx(`ck-status-${s.status}`, 'h-full bg-(--ck-bar)')}
            style={{ width: `${s.percent}%` }}
          />
        ))}
      </div>
      <ul className="flex flex-wrap gap-x-4 gap-y-1">
        {legend.map(([field, status]) => (
          <li key={status} className="flex items-center gap-1.5 text-[11.5px] font-semibold text-(--ink-3)">
            <span className={clsx(`ck-status-${status}`, 'size-2 rounded-full bg-(--ck-bar)')} aria-hidden />
            {t(`status.${status}`)} {num(summary[field])}
          </li>
        ))}
      </ul>
    </section>
  );
}

function CheckupRow({ item }: { item: CheckupItem }) {
  const t = useTranslations('checkups');
  const locale = useLocale() as Locale;
  const dir = useDirection();
  const date = (d: string | null) => (d ? formatLongDate(fromApiDate(d), locale) : t('noDate'));
  const next = item.nextDueLabel ?? date(item.nextDueOn);
  const meta = [item.intervalLabel, item.timingLabel].filter(Boolean).join(t('separator'));

  return (
    <li>
      <Link
        href={`/checkups/${item.id}`}
        className={clsx(
          `ck-tone-${item.tone}`,
          'card flex items-center gap-3 p-4 text-start no-underline focus-visible:shadow-(--ring) focus-visible:outline-none',
        )}
      >
        <span className="grid size-11 shrink-0 place-items-center rounded-2xl bg-(--ck-soft) text-(--ck-ink)">
          <Icon name={checkupIcon(item.icon, { category: item.category })} size={20} />
        </span>
        <span className="flex min-w-0 flex-1 flex-col gap-0.5">
          <span className="truncate text-[14px] font-extrabold text-(--ink)">{item.title}</span>
          {meta && <span className="truncate text-[11.5px] font-semibold text-(--ink-3)">{meta}</span>}
          <span className="truncate text-[11.5px] text-(--ink-3)">
            {t('list.lastNext', { last: date(item.lastDoneOn), next })}
          </span>
        </span>
        <StatusPill status={item.status} label={t(`status.${item.status}`)} />
        <Icon name={dir === 'rtl' ? 'chevronLeft' : 'chevronRight'} size={16} className="shrink-0 text-(--ink-3)" />
      </Link>
    </li>
  );
}

/**
 * Checkups (`v14_Checkups`): the age-based screening plan — summary card,
 * URL-synced tabs (`?filter=`), sections in server order, and the custom-checkup
 * entry. Personal health data (§11): display only, never logged.
 */
export function CheckupsPage({ initialFilter }: { initialFilter?: string }) {
  const t = useTranslations('checkups');
  const router = useRouter();
  const dir = useDirection();
  const locale = useLocale() as Locale;
  const [filter, setFilter] = useState<CheckupListFilter>(() => parseFilter(initialFilter));
  const [settingsOpen, setSettingsOpen] = useState(false);
  const tabRefs = useRef<Partial<Record<CheckupListFilter, HTMLButtonElement | null>>>({});
  const query = useCheckups(filter);
  // The settings sheet always lists the whole plan, whatever the tab.
  const all = useCheckups('all');

  const select = (next: CheckupListFilter, focus = false) => {
    if (focus) tabRefs.current[next]?.focus();
    if (next === filter) return;
    setFilter(next);
    router.replace(filterHref(next), { scroll: false });
  };
  const onTabKey = (event: KeyboardEvent<HTMLDivElement>) => {
    const next = nextFilter(filter, event.key, dir === 'rtl' ? 'rtl' : 'ltr');
    if (!next) return;
    event.preventDefault();
    select(next, true);
  };

  const age = query.data?.age ?? all.data?.age ?? null;
  const summary = all.data?.summary ?? query.data?.summary;
  const groups = query.data ? groupBySection(query.data.items) : [];
  const emptyText =
    filter === 'action' ? t('list.emptyAction') : filter === 'done' ? t('list.emptyDone') : t('list.empty');

  return (
    <div className="view rmd-page">
      <div className="scroll">
        <header className="rmd-hdr">
          <Link href="/home" className="rmd-hdr-btn" aria-label={t('back')}>
            <Icon name={dir === 'rtl' ? 'chevronRight' : 'chevronLeft'} size={20} strokeWidth={1.8} />
          </Link>
          <div className="rmd-hdr-text">
            <h1 className="rmd-hdr-title">{t('title')}</h1>
            {age !== null && (
              <p className="rmd-hdr-sub">{t('list.ageBasis', { age: formatNumber(age, locale) })}</p>
            )}
          </div>
          <button
            type="button"
            className="rmd-hdr-btn"
            aria-label={t('list.planSettings')}
            onClick={() => setSettingsOpen(true)}
          >
            <Icon name="cog" size={20} strokeWidth={1.8} />
          </button>
        </header>

        <div className="rmd-body flex flex-col gap-3 pb-8">
          {summary && summary.total > 0 && <SummaryCard summary={summary} />}

          <div className="rmd-tabs" role="tablist" aria-label={t('title')} onKeyDown={onTabKey}>
            {CHECKUP_LIST_FILTERS.map((key) => (
              <button
                key={key}
                ref={(el) => {
                  tabRefs.current[key] = el;
                }}
                type="button"
                role="tab"
                id={`ck-tab-${key}`}
                aria-selected={filter === key}
                aria-controls={PANEL_ID}
                tabIndex={filter === key ? 0 : -1}
                className={clsx('rmd-tab', filter === key && 'on')}
                onClick={() => select(key)}
              >
                {t(`list.tabs.${key}`)}
              </button>
            ))}
          </div>

          <div id={PANEL_ID} role="tabpanel" aria-labelledby={`ck-tab-${filter}`} className="flex flex-col gap-4">
            {query.isPending ? (
              <p className="rmd-state" role="status">
                {t('loading')}
              </p>
            ) : query.isError ? (
              <div className="rmd-state" role="alert">
                <p>{t('loadError')}</p>
                <button type="button" className="rmd-retry" onClick={() => void query.refetch()}>
                  {t('retry')}
                </button>
              </div>
            ) : groups.length === 0 ? (
              <div className="card flex flex-col items-center gap-2 px-4 py-8 text-center">
                <Icon name="stetho" size={28} className="text-(--brand)" />
                <p className="text-[13px] font-semibold text-(--ink-3)">{emptyText}</p>
              </div>
            ) : (
              groups.map((group) => (
                <section key={group.section} aria-labelledby={`ck-sec-${group.section}`}>
                  <h2
                    id={`ck-sec-${group.section}`}
                    className="mb-2 text-start text-[13px] font-extrabold text-(--ink-3)"
                  >
                    {t(`section.${group.section}`)}
                  </h2>
                  <ul className="flex flex-col gap-2">
                    {group.items.map((item) => (
                      <CheckupRow key={item.id} item={item} />
                    ))}
                  </ul>
                </section>
              ))
            )}
          </div>

          <p className="flex items-start gap-2 rounded-2xl bg-(--surface-2) p-3 text-start text-[11.5px] leading-relaxed text-(--ink-3)">
            <Icon name="info" size={16} className="mt-0.5 shrink-0" />
            {t('list.infoNote')}
          </p>

          <Link href="/checkups/custom/new" className="btn btn-ghost w-full border-[1.5px] border-(--brand)">
            <Icon name="plus" size={16} strokeWidth={2.2} />
            {t('list.addCustom')}
          </Link>
        </div>
      </div>

      <PlanSettingsSheet open={settingsOpen} items={all.data?.items ?? []} onClose={() => setSettingsOpen(false)} />
    </div>
  );
}

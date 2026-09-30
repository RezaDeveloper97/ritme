'use client';

import clsx from 'clsx';
import { useLocale, useTranslations } from 'next-intl';
import { useState } from 'react';

import {
  CHECKUP_LIST_FILTERS,
  type CheckupItem,
  type CheckupListFilter,
  type CheckupStatus,
  type CheckupSummary,
  checkupIcon,
  checkupStatusIcon,
  formatCheckupMonth,
  useCheckups,
} from '@/entities/checkup';
import { type Locale, Link, useRouter } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib/date';
import {
  EmptyState,
  HeaderButton,
  Icon,
  ScreenHeader,
  SegmentedTabs,
  Skeleton,
  SkeletonGroup,
  SkyLayer,
} from '@/shared/ui';

import {
  barSegments,
  filterHref,
  groupBySection,
  parseFilter,
  rowMeta,
  worstStatus,
} from '../model/view';
import { PlanSettingsSheet } from './PlanSettingsSheet';

const PANEL_ID = 'ck-panel';

/** v14 outlined pill with the status glyph (bell / info / clock / check). */
function StatusPill({ status, label }: { status: CheckupStatus; label: string }) {
  return (
    <span className={clsx(`ck-status-${status}`, 'ck-pill')}>
      <Icon name={checkupStatusIcon(status)} size={12} strokeWidth={2.2} />
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
  // Lists read month + year («مهر ۱۴۰۴»); the full date stays on the detail page.
  const date = (d: string | null) => (d ? formatCheckupMonth(d, locale) : t('noDate'));
  const next = item.nextDueLabel ?? date(item.nextDueOn);
  const meta = rowMeta(item, t('separator'));

  return (
    <li>
      <Link
        href={`/checkups/${item.id}`}
        className={clsx(
          `ck-tone-${item.tone}`,
          'flex items-center gap-3 p-4 text-start no-underline focus-visible:bg-(--surface-2) focus-visible:outline-none',
        )}
      >
        <span className="grid size-11 shrink-0 place-items-center rounded-2xl bg-(--ck-soft) text-(--ck-ink)">
          <Icon name={checkupIcon(item.icon, { category: item.category })} size={20} />
        </span>
        <span className="flex min-w-0 flex-1 flex-col gap-0.5">
          <span className="text-[14px] font-extrabold text-(--ink)">{item.title}</span>
          {meta && <span className="text-[11.5px] leading-5 font-semibold text-(--ink-3)">{meta}</span>}
          <span className="text-[11.5px] leading-5 text-(--ink-3)">
            {t('list.lastNext', { last: date(item.lastDoneOn), next })}
          </span>
        </span>
        <StatusPill status={item.status} label={t(`status.${item.status}`)} />
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
  const locale = useLocale() as Locale;
  const [filter, setFilter] = useState<CheckupListFilter>(() => parseFilter(initialFilter));
  const [settingsOpen, setSettingsOpen] = useState(false);
  const query = useCheckups(filter);
  // The settings sheet always lists the whole plan, whatever the tab.
  const all = useCheckups('all');

  const select = (next: CheckupListFilter) => {
    if (next === filter) return;
    setFilter(next);
    router.replace(filterHref(next), { scroll: false });
  };

  const age = query.data?.age ?? all.data?.age ?? null;
  const summary = all.data?.summary ?? query.data?.summary;
  const groups = query.data ? groupBySection(query.data.items) : [];
  const emptyText =
    filter === 'action' ? t('list.emptyAction') : filter === 'done' ? t('list.emptyDone') : t('list.empty');

  return (
    <div className="view rmd-page">
      <div className="scroll rmd-screen">
        <SkyLayer />
        <ScreenHeader
          title={t('title')}
          subtitle={age !== null ? t('list.ageBasis', { age: formatNumber(age, locale) }) : undefined}
          onBack={() => router.push('/home')}
          backLabel={t('back')}
          action={
            <HeaderButton icon="filterLines" label={t('list.planSettings')} onClick={() => setSettingsOpen(true)} />
          }
        />

        <div className="rmd-body flex flex-col gap-3 pb-8">
          {summary && summary.total > 0 && <SummaryCard summary={summary} />}

          <SegmentedTabs
            tabs={CHECKUP_LIST_FILTERS.map((key) => ({ value: key, label: t(`list.tabs.${key}`) }))}
            value={filter}
            onChange={select}
            label={t('title')}
            panelId={() => PANEL_ID}
          />

          <div id={PANEL_ID} role="tabpanel" aria-label={t(`list.tabs.${filter}`)} className="flex flex-col gap-4">
            {query.isPending ? (
              <SkeletonGroup label={t('loading')} className="rmd-form-skel">
                <Skeleton shape="card" />
                <Skeleton shape="card" />
                <Skeleton shape="card" />
              </SkeletonGroup>
            ) : query.isError ? (
              <div className="rmd-state" role="alert">
                <p>{t('loadError')}</p>
                <button type="button" className="rmd-retry" onClick={() => void query.refetch()}>
                  {t('retry')}
                </button>
              </div>
            ) : groups.length === 0 ? (
              <EmptyState icon="stetho" title={emptyText} className="card" />
            ) : (
              groups.map((group) => (
                <section key={group.section} aria-labelledby={`ck-sec-${group.section}`}>
                  <h2
                    id={`ck-sec-${group.section}`}
                    className="mb-2 text-start text-[15px] font-extrabold text-(--ink)"
                  >
                    {t(`section.${group.section}`)}
                  </h2>
                  {/* One card per section, rows split by hairlines (v14_Checkups «سالانه»). */}
                  <ul className="card flex flex-col divide-y divide-(--line) overflow-hidden p-0">
                    {group.items.map((item) => (
                      <CheckupRow key={item.id} item={item} />
                    ))}
                  </ul>
                </section>
              ))
            )}
          </div>

          <p className="card flex items-start gap-3 p-4 text-start text-[12px] leading-relaxed text-(--ink-2)">
            <span className="grid size-9 shrink-0 place-items-center rounded-full bg-(--pink-bg) text-(--brand-strong)" aria-hidden>
              <Icon name="info" size={17} />
            </span>
            {t('list.infoNote')}
          </p>

          <Link href="/checkups/custom/new" className="nb-btn is-outline is-block">
            <Icon name="plus" size={16} strokeWidth={2.2} />
            {t('list.addCustom')}
          </Link>
        </div>
      </div>

      <PlanSettingsSheet open={settingsOpen} items={all.data?.items ?? []} onClose={() => setSettingsOpen(false)} />
    </div>
  );
}

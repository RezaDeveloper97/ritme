'use client';

import clsx from 'clsx';
import { useFormatter, useLocale, useTranslations } from 'next-intl';
import { useState } from 'react';

import { PeriodDateEditor } from '@/features/log-period';
import { Link, useDirection, useRouter, type Locale } from '@/shared/i18n';
import { formatDayMonth, fromApiDate, monthName, shiftMonth, toParts, today } from '@/shared/lib/date';
import { useMounted } from '@/shared/lib/use-mounted';
import {
  Card,
  EmptyState,
  Icon,
  PrimaryButton,
  ScreenHeader,
  SecondaryButton,
  Skeleton,
  SkeletonGroup,
  SkyLayer,
} from '@/shared/ui';

import { useCycleHistory } from '../api/queries';
import { barGeometry, barScale } from '../model/bars';
import type { CycleHistory, CycleHistoryRow } from '../model/schema';

/**
 * «سیکل‌های من» (B-N1-08, `nbl_/nbd_Cycle_History`) at `/cycle`: the median
 * cycle and period length, the spread, a regularity verdict and one bar per
 * recent cycle, with the out-of-range ones marked. Every number comes from
 * `GET /cycle/history`; nothing is estimated here (§11: descriptive, never
 * diagnostic). A back-header screen — no bottom nav.
 */
export function CyclePage() {
  const t = useTranslations('cycle');
  const router = useRouter();
  const mounted = useMounted();
  const query = useCycleHistory();
  const [editorOpen, setEditorOpen] = useState(false);
  const loc = useLocale() as Locale;

  // «Add previous cycles» opens on last month: the user is adding the past.
  const now = toParts(today(), loc);
  const lastMonth = shiftMonth(now.year, now.month, -1);

  const data = query.data;
  const loading = !mounted || query.isLoading;

  return (
    <div className="view cyh-page">
      <SkyLayer />
      <div className="scroll cyh-scroll">
        <ScreenHeader
          title={t('history.title')}
          subtitle={data ? t('history.subtitle', { n: data.basedOn }) : undefined}
          onBack={() => router.push('/home')}
          backLabel={t('back')}
        />

        {loading ? (
          <HistorySkeleton label={t('loading')} />
        ) : query.isError || !data ? (
          <Card className="cyh-error" role="alert">
            <p className="cyh-error-t">{t('history.error')}</p>
            <SecondaryButton block={false} onClick={() => query.refetch()}>
              {t('retry')}
            </SecondaryButton>
          </Card>
        ) : data.cycles.length === 0 ? (
          <EmptyState
            icon="drop"
            title={t('history.empty.title')}
            body={t('history.empty.body')}
            action={
              <PrimaryButton icon="plus" onClick={() => setEditorOpen(true)}>
                {t('history.addPrevious')}
              </PrimaryButton>
            }
          />
        ) : (
          <HistoryBody data={data} onAdd={() => setEditorOpen(true)} />
        )}
      </div>

      <PeriodDateEditor open={editorOpen} onClose={() => setEditorOpen(false)} initialView={lastMonth} intent="edit" />
    </div>
  );
}

function HistoryBody({ data, onAdd }: { data: CycleHistory; onAdd: () => void }) {
  const t = useTranslations('cycle');
  const format = useFormatter();
  const rtl = useDirection() === 'rtl';
  const scale = barScale(data.cycles, data.predicted);
  const num = (v: number | null) => (v === null ? '—' : format.number(v));

  return (
    <>
      <div className="cyh-stats">
        <Stat label={t('history.stats.cycle')} value={num(data.medianCycle)} unit={t('history.stats.median')} tone="brand" />
        <Stat label={t('history.stats.period')} value={num(data.medianPeriod)} unit={t('history.stats.days')} tone="period" />
        <Stat
          label={t('history.stats.spread')}
          value={data.spread === null ? '—' : t('history.stats.spreadValue', { n: data.spread })}
          unit={t('history.stats.days')}
          tone="data"
        />
      </div>

      <Card className="cyh-verdict">
        <div className="cyh-verdict-b">
          <p className="cyh-verdict-t">{t(`history.verdict.${data.regularity}`)}</p>
          <p className="cyh-verdict-s">
            {data.range && data.regularity !== 'not_enough_data'
              ? t('history.verdict.range', {
                  inRange: data.inRange,
                  total: data.total,
                  min: data.range.min,
                  max: data.range.max,
                })
              : t('history.verdict.notEnough')}
          </p>
        </div>
        <Icon
          name={data.regularity === 'irregular' ? 'info' : 'check'}
          size={22}
          className={clsx('cyh-verdict-i', data.regularity === 'irregular' && 'is-warm')}
        />
      </Card>

      <Card padding="none" className="cyh-list" as="section" aria-label={t('history.listLabel')}>
        <ul className="cyh-rows">
          {data.cycles.map((row) => (
            <CycleRow key={row.start} row={row} scale={scale} predicted={data.predicted} />
          ))}
        </ul>
      </Card>

      <Link href="/cycle/symptoms" className="cyh-link">
        <span className="cyh-link-t">{t('history.symptomPattern')}</span>
        <Icon name={rtl ? 'chevronLeft' : 'chevronRight'} size={18} />
      </Link>

      <ul className="cyh-legend" aria-hidden>
        <li>
          <span className="cyh-dot is-period" />
          {t('history.legend.period')}
        </li>
        <li>
          <span className="cyh-dot is-cycle" />
          {t('history.legend.cycle')}
        </li>
        <li>
          <span className="cyh-dot is-out" />
          {t('history.legend.outOfRange')}
        </li>
      </ul>

      <SecondaryButton icon="plus" onClick={onAdd} className="cyh-add">
        {t('history.addPrevious')}
      </SecondaryButton>
    </>
  );
}

function Stat({ label, value, unit, tone }: { label: string; value: string; unit: string; tone: 'brand' | 'period' | 'data' }) {
  return (
    <div className="cyh-stat">
      <span className="cyh-stat-l">{label}</span>
      <b className={clsx('cyh-stat-v', `is-${tone}`)}>{value}</b>
      <span className="cyh-stat-u">{unit}</span>
    </div>
  );
}

function CycleRow({ row, scale, predicted }: { row: CycleHistoryRow; scale: number; predicted: number }) {
  const t = useTranslations('cycle');
  const format = useFormatter();
  const loc = useLocale() as Locale;
  const start = formatDayMonth(fromApiDate(row.start), loc);
  const geo = barGeometry(row, scale);
  // The current cycle also shows how far the prediction reaches, as the faint track tail.
  const predictedPct = row.isCurrent ? Math.min(100, (predicted / scale) * 100) : 0;
  const out = row.inRange === false;

  const title = row.isCurrent
    ? t('history.current')
    : monthName(toParts(fromApiDate(row.end ?? row.start), loc).month, loc);
  const dates = row.isCurrent
    ? t('history.toToday', { from: start })
    : t('history.range', { from: start, to: formatDayMonth(fromApiDate(row.end ?? row.start), loc) });

  return (
    <li className="cyh-row">
      <div className="cyh-row-head">
        <div className="cyh-row-b">
          <p className="cyh-row-t">{title}</p>
          <p className="cyh-row-s">{dates}</p>
        </div>
        <p className="cyh-row-n">
          <b className="cyh-row-num">{format.number(row.length)}</b>
          <span className="cyh-row-u">{t('history.days')}</span>
        </p>
      </div>
      <div
        className={clsx('cyh-bar', row.isCurrent && 'is-current')}
        role="img"
        aria-label={`${t('history.barLabel', { length: row.length, period: row.periodDays })}${out ? ` — ${t('history.outOfRange')}` : ''}`}
      >
        {row.isCurrent ? <span className="cyh-bar-pred" style={{ inlineSize: `${predictedPct}%` }} /> : null}
        <span className="cyh-bar-cycle" style={{ inlineSize: `${geo.cycle}%` }} />
        <span className="cyh-bar-period" style={{ inlineSize: `${geo.period}%` }} />
        {out ? <span className="cyh-bar-out" style={{ insetInlineStart: `${geo.cycle}%` }} /> : null}
      </div>
    </li>
  );
}

function HistorySkeleton({ label }: { label: string }) {
  return (
    <SkeletonGroup label={label} className="cyh-skel">
      <div className="cyh-stats">
        <Skeleton shape="block" className="cyh-skel-stat" />
        <Skeleton shape="block" className="cyh-skel-stat" />
        <Skeleton shape="block" className="cyh-skel-stat" />
      </div>
      <Skeleton shape="card" />
      <Skeleton shape="block" className="cyh-skel-list" />
    </SkeletonGroup>
  );
}

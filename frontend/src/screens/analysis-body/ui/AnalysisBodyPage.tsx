'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useState } from 'react';

import { REPORT_DEFAULT_RANGE, REPORT_RANGES, useBodyReport } from '@/entities/analysis';
import { useRouter, type Locale } from '@/shared/i18n';
import {
  formatDayMonth,
  formatDecimal,
  formatNumber,
  fromApiDate,
  weekdayKeys,
  weekdayLabels,
} from '@/shared/lib/date';
import { useMounted } from '@/shared/lib/use-mounted';
import { InfoNote } from '@/shared/ui';
import { BottomNav } from '@/widgets/bottom-nav';
import { ChartCard, ColumnChart, ReportFrame, TrendLine } from '@/widgets/charts';

import { headlineWeight, maxIndex, signed, weekdayValues } from '../model/body';

type Range = (typeof REPORT_RANGES.body)[number];

/**
 * `/analysis/body` (An_Body, B-N3-09): weight as a 7-day moving average with
 * its 7 / 30-day change and BMI, average sleep by weekday (+ the luteal-phase
 * average), active days per week, and the moving-average footnote.
 */
export function AnalysisBodyPage() {
  const t = useTranslations('analysis.reports');
  const tA = useTranslations('analysis');
  const loc = useLocale() as Locale;
  const router = useRouter();
  const mounted = useMounted();
  const [range, setRange] = useState<Range>(REPORT_DEFAULT_RANGE.body);
  const query = useBodyReport(range);
  const r = query.data;
  const num = (n: number) => formatNumber(n, loc);
  const dec = (n: number) => formatDecimal(String(Math.round(n * 10) / 10), loc);
  const delta = (v: number) => {
    const s = signed(v);
    return `${s.sign}${formatDecimal(s.abs, loc)}`;
  };
  const status =
    !mounted || (query.isPending && query.fetchStatus !== 'idle') ? 'loading' : query.isError || !r ? 'error' : 'ready';
  const sep = tA('format.list_separator');

  let content = null;
  if (r) {
    const w = r.weight;
    const head = headlineWeight(w.points, w.current);
    const line = w.points.map((p) => p.avg7 ?? p.value);
    const firstLine = line.find((v) => v !== null) ?? null;
    const lastLine = [...line].reverse().find((v) => v !== null) ?? null;

    const order = weekdayKeys(loc);
    const short = weekdayLabels(loc);
    const days = weekdayValues(r.sleep.byWeekday, order);
    const sleepMax = maxIndex(days.map((d) => d.value));
    const weeks = r.activity.weeks;

    content = (
      <>
        <ChartCard className="anr-body-card">
          <div className="anr-metric">
            <div className="anr-metric-main">
              <span className="anr-metric-label">{t('body.weight.title', { n: num(w.movingAverageDays) })}</span>
              <span className="anr-big">{head != null ? dec(head) : t('none')}</span>
            </div>
            {w.ready && w.deltaRange != null ? (
              <span className="anr-metric-delta">
                {t.rich('body.weight.delta', {
                  delta: delta(w.deltaRange),
                  n: (chunks) => <bdi dir="ltr">{chunks}</bdi>,
                })}
              </span>
            ) : null}
          </div>
          {w.ready && line.length > 1 && firstLine != null && lastLine != null ? (
            <TrendLine
              label={t('body.weight.chart', {
                n: num(w.movingAverageDays),
                from: dec(firstLine),
                to: dec(lastLine),
              })}
              table={{
                caption: t('body.weight.title', {
                  n: num(w.movingAverageDays),
                }),
                columns: [t('body.weight.colDate'), t('body.weight.colAvg')],
                rows: w.points
                  .filter((p) => (p.avg7 ?? p.value) !== null)
                  .map((p) => [formatDayMonth(fromApiDate(p.date), loc), dec((p.avg7 ?? p.value) as number)]),
              }}
              values={line}
              tone="data"
            />
          ) : (
            <p className="anr-note">{t('body.weight.notReady')}</p>
          )}
          <dl className="anr-minis" aria-label={t('body.weight.statsLabel')}>
            <div className="anr-mini">
              <dt>{t('body.weight.d7')}</dt>
              <dd>{w.delta7d != null ? <bdi dir="ltr">{delta(w.delta7d)}</bdi> : t('none')}</dd>
            </div>
            <div className="anr-mini">
              <dt>{t('body.weight.d30')}</dt>
              <dd>{w.delta30d != null ? <bdi dir="ltr">{delta(w.delta30d)}</bdi> : t('none')}</dd>
            </div>
            <div className="anr-mini">
              <dt>{t('body.weight.bmi')}</dt>
              <dd>{w.bmi != null ? dec(w.bmi) : t('none')}</dd>
            </div>
          </dl>
        </ChartCard>

        <ChartCard className="anr-body-card">
          <div className="anr-metric">
            <div className="anr-metric-main">
              <span className="anr-metric-label">{t('body.sleep.title')}</span>
              <span className="anr-big-row">
                <span className="anr-big">{r.sleep.avgHours != null ? dec(r.sleep.avgHours) : t('none')}</span>
                <span className="anr-big-unit">{t('body.sleep.avg')}</span>
              </span>
            </div>
            {r.sleep.lutealAvgHours != null ? (
              <span className="anr-chip-out">{t('body.sleep.luteal', { n: dec(r.sleep.lutealAvgHours) })}</span>
            ) : null}
          </div>
          {r.sleep.ready ? (
            <ColumnChart
              label={t('body.sleep.chart', {
                list: days
                  .map((d) => `${t(`body.weekdays.${d.key}`)} ${d.value != null ? dec(d.value) : t('none')}`)
                  .join(sep),
              })}
              table={{
                caption: t('body.sleep.title'),
                columns: [t('body.sleep.colDay'), t('body.sleep.colHours')],
                rows: days.map((d) => [t(`body.weekdays.${d.key}`), d.value != null ? dec(d.value) : t('none')]),
              }}
              columns={days.map((d, i) => ({
                key: d.key,
                label: short[i],
                value: d.value,
                tone: 'brand',
                solid: i === sleepMax,
              }))}
              baseline={3}
              floor={0.25}
              height={120}
            />
          ) : (
            <p className="anr-note">{t('body.sleep.notReady')}</p>
          )}
        </ChartCard>

        <ChartCard className="anr-body-card">
          <div className="anr-metric">
            <div className="anr-metric-main">
              <span className="anr-metric-label">{t('body.activity.title')}</span>
              <span className="anr-big-row">
                <span className="anr-big">{num(r.activity.activeDays)}</span>
                <span className="anr-big-unit">{t('body.activity.sub', { n: num(r.activity.days) })}</span>
              </span>
            </div>
          </div>
          {weeks.length > 1 ? (
            <ColumnChart
              label={t('body.activity.chart', {
                list: weeks
                  .map((wk, i) => `${t('body.activity.week', { n: num(i + 1) })}: ${num(wk.activeDays)}`)
                  .join(sep),
              })}
              table={{
                caption: t('body.activity.title'),
                columns: [t('body.activity.colWeek'), t('body.activity.colDays')],
                rows: weeks.map((wk) => [
                  formatDayMonth(fromApiDate(wk.start), loc),
                  `${num(wk.activeDays)} / ${num(wk.days)}`,
                ]),
              }}
              columns={weeks.map((wk, i) => ({
                key: wk.start,
                label: i === 0 ? t('body.activity.week', { n: num(1) }) : weeks.length > 12 && i % 4 ? '' : num(i + 1),
                value: wk.activeDays,
                tone: 'data',
                solid: i === weeks.length - 1,
              }))}
              max={7}
              floor={0.15}
              gap={weeks.length > 20 ? 3 : 8}
              height={110}
            />
          ) : null}
        </ChartCard>

        <InfoNote>{t('body.footnote')}</InfoNote>
      </>
    );
  }

  return (
    <ReportFrame
      title={t('body.title')}
      subtitle={t(`body.ranges.${range}`)}
      backLabel={t('back')}
      onBack={() => router.push('/analysis')}
      tabs={{
        label: t('rangeLabel'),
        value: range,
        options: REPORT_RANGES.body.map((v) => ({
          value: v,
          label: t(`body.ranges.${v}`),
        })),
        onChange: setRange,
      }}
      status={status}
      updating={query.isPlaceholderData && query.isFetching}
      loadingLabel={t('loading')}
      error={{
        title: t('error.title'),
        body: t('error.body'),
        retry: t('error.retry'),
        onRetry: () => void query.refetch(),
        retrying: query.isFetching,
      }}
      after={<BottomNav />}
    >
      {content}
    </ReportFrame>
  );
}

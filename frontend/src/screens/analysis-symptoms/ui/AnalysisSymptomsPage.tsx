'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useState } from 'react';

import { REPORT_DEFAULT_RANGE, REPORT_RANGES, useSymptomsReport } from '@/entities/analysis';
import { useRouter, type Locale } from '@/shared/i18n';
import { formatDayMonth, formatDecimal, formatNumber, fromApiDate } from '@/shared/lib/date';
import { useMounted } from '@/shared/lib/use-mounted';
import { openSheet } from '@/shared/sheet';
import { EmptyState, InfoNote, PillChip, PrimaryButton } from '@/shared/ui';
import { BottomNav } from '@/widgets/bottom-nav';
import { ChartCard, ReportFrame, SeriesBars, StripRows } from '@/widgets/charts';

import { activeKeys, highlightKey, labelsOf, patternTone, toggleKey, toneOf } from '../model/symptoms';

type Range = (typeof REPORT_RANGES.symptoms)[number];

/**
 * `/analysis/symptoms` (An_Symptoms, B-N3-09) — also the menopause «علائم»
 * tab: the trend of the top symptoms (pick up to three lanes), their pattern
 * on the typical cycle (≥ 3 cycles) with the same «N روز قبل از پریود»
 * sentence as the hub, and the min-cycles footnote.
 */
export function AnalysisSymptomsPage() {
  const t = useTranslations('analysis.reports');
  const tA = useTranslations('analysis');
  const loc = useLocale() as Locale;
  const router = useRouter();
  const mounted = useMounted();
  const [range, setRange] = useState<Range>(REPORT_DEFAULT_RANGE.symptoms);
  const [chosen, setChosen] = useState<string[] | null>(null);
  const query = useSymptomsReport(range);
  const r = query.data;
  const num = (n: number) => formatNumber(n, loc);
  const dec = (n: number) => formatDecimal(String(Math.round(n * 10) / 10), loc);
  const status =
    !mounted || (query.isPending && query.fetchStatus !== 'idle') ? 'loading' : query.isError || !r ? 'error' : 'ready';
  const rangeLabel = t(`ranges.${range}`);

  let content = null;
  if (r) {
    const labels = labelsOf(r);
    const keys = r.trend.keys;
    const active = activeKeys(keys, chosen);
    const points = r.trend.points;
    const pat = r.pattern;
    const h = r.highlight;
    const missing = Math.max(0, pat.cyclesNeeded - pat.cyclesCounted);
    const list = (ks: readonly string[]) => ks.map((k) => labels[k] ?? k).join(tA('format.list_separator'));

    content =
      r.top.length === 0 ? (
        <EmptyState
          icon="symptom"
          title={t('empty.title')}
          body={t('empty.body')}
          action={
            <PrimaryButton icon="plus" block={false} onClick={() => openSheet('log')}>
              {t('empty.cta')}
            </PrimaryButton>
          }
        />
      ) : (
        <>
          <ChartCard className="anr-trend-card">
            {r.trend.ready && points.length ? (
              <SeriesBars
                label={t('symptoms.chart', {
                  list: list(active),
                  range: rangeLabel,
                })}
                table={{
                  caption: t('symptoms.title'),
                  columns: [t('symptoms.colDate'), ...active.map((k) => labels[k] ?? k)],
                  rows: points.map((p) => [
                    formatDayMonth(fromApiDate(p.start), loc),
                    ...active.map((k) => dec(p.values[keys.indexOf(k)] ?? 0)),
                  ]),
                }}
                series={active.map((k) => ({
                  key: k,
                  tone: toneOf(keys, k),
                  values: points.map((p) => p.values[keys.indexOf(k)] ?? 0),
                }))}
                from={formatDayMonth(fromApiDate(points[0].start), loc)}
                to={formatDayMonth(fromApiDate(points[points.length - 1].start), loc)}
              />
            ) : (
              <p className="anr-note">{t('symptoms.trendNotReady')}</p>
            )}
          </ChartCard>

          {r.trend.ready && keys.length > 1 ? (
            <div className="anr-chips" role="group" aria-label={t('symptoms.pick')}>
              {keys.map((k) => (
                <PillChip
                  key={k}
                  mode="multi"
                  tone={toneOf(keys, k)}
                  pressed={active.includes(k)}
                  onPressedChange={() => setChosen(toggleKey(keys, active, k))}
                  className="anr-chip"
                >
                  {labels[k] ?? k}
                </PillChip>
              ))}
            </div>
          ) : null}

          <ChartCard title={t('symptoms.pattern.title')}>
            {h ? (
              <p className="anr-highlight">
                {tA.rich(`hub.symptoms.highlight.${highlightKey(h)}`, {
                  symptom: h.label,
                  days: num(h.days ?? 0),
                  start: num(h.startDay),
                  end: num(h.endDay),
                  b: (chunks) => <b>{chunks}</b>,
                })}
              </p>
            ) : null}
            {!pat.ready ? (
              <p className="anr-note">{t('symptoms.pattern.notReady', { n: num(missing) })}</p>
            ) : pat.items.length ? (
              <StripRows
                label={t('symptoms.pattern.chart', {
                  list: pat.items.map((i) => i.label).join(tA('format.list_separator')),
                  length: num(pat.typical.cycleLength),
                })}
                table={{
                  caption: t('symptoms.pattern.title'),
                  columns: [t('symptoms.pattern.colSymptom'), t('symptoms.pattern.colWindow')],
                  rows: pat.items.map((i) => [
                    i.label,
                    i.window
                      ? i.window.startDay === i.window.endDay
                        ? t('symptoms.pattern.day', {
                            n: num(i.window.startDay),
                          })
                        : `${t('symptoms.pattern.day', { n: num(i.window.startDay) })}–${num(i.window.endDay)}`
                      : t('none'),
                  ]),
                }}
                rows={pat.items.map((i, j) => ({
                  key: i.key,
                  label: i.label,
                  tone: patternTone(keys, i.key, j),
                  values: i.strip,
                }))}
                dayLabel={(d) => (d === 1 ? t('symptoms.pattern.day', { n: num(1) }) : num(d))}
              />
            ) : (
              <p className="anr-note">{t('symptoms.pattern.none')}</p>
            )}
          </ChartCard>

          <InfoNote>{t('symptoms.footnote', { n: num(pat.cyclesNeeded) })}</InfoNote>
        </>
      );
  }

  return (
    <ReportFrame
      title={t('symptoms.title')}
      subtitle={t('recent', { range: rangeLabel })}
      backLabel={t('back')}
      onBack={() => router.push('/analysis')}
      tabs={{
        label: t('rangeLabel'),
        value: range,
        options: REPORT_RANGES.symptoms.map((v) => ({
          value: v,
          label: t(`ranges.${v}`),
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

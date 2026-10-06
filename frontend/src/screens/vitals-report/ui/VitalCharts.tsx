'use client';

import { clsx } from 'clsx';

import type { Tone } from '@/shared/ui';
import { ChartFigure, type DataTable } from '@/widgets/charts';

import { CHART_W, PAD_X, labelIndexes, linePath, makeScale, slotX, type Scale } from '../model/chart';

function Grid({ scale, fmt }: { scale: Scale; fmt: (n: number) => string }) {
  return (
    <g className="vtc-grid">
      {scale.ticks.map((v) => (
        <g key={v}>
          <line x1={PAD_X} x2={CHART_W} y1={scale.y(v)} y2={scale.y(v)} />
          <text x={2} y={scale.y(v) + 3}>
            {fmt(v)}
          </text>
        </g>
      ))}
    </g>
  );
}

function Band({ scale, band }: { scale: Scale; band: readonly [number, number] }) {
  const top = scale.y(band[1]);
  return <rect className="vtc-band" x={PAD_X} y={top} width={CHART_W - PAD_X} height={Math.max(0, scale.y(band[0]) - top)} />;
}

function XLabels({ labels }: { labels: readonly string[] }) {
  const idx = labelIndexes(labels.length, 7);
  return (
    <div className="vtc-x" dir="ltr" aria-hidden>
      {idx.map((i) => (
        <span key={i} className="vtc-x-label" style={{ insetInlineStart: `${(slotX(i, labels.length) / CHART_W) * 100}%` }}>
          {labels[i]}
        </span>
      ))}
    </div>
  );
}

export interface BpPoint {
  systolic: number;
  diastolic: number;
  tone: Tone;
}

/** Daily systolic–diastolic bars over the normal band (nbl_Vitals_BPReport): ○ systolic, ● diastolic. */
export function BpChart({
  points,
  labels,
  band,
  label,
  table,
  fmt,
  height = 170,
}: {
  points: readonly BpPoint[];
  labels: readonly string[];
  band: readonly [number, number];
  label: string;
  table: DataTable;
  fmt: (n: number) => string;
  height?: number;
}) {
  const scale = makeScale(points.flatMap((p) => [p.systolic, p.diastolic]), band, height);
  const n = points.length;
  return (
    <ChartFigure label={label} table={table} className="vtc">
      <svg viewBox={`0 0 ${CHART_W} ${height}`} width="100%" className="axc-svg vtc-svg" direction="ltr" aria-hidden focusable="false">
        <Band scale={scale} band={band} />
        <Grid scale={scale} fmt={fmt} />
        {points.map((p, i) => {
          const x = slotX(i, n);
          return (
            <g key={i} className={clsx('vtc-bp', `nb-tone-${p.tone}`)}>
              <line className="vtc-bp-bar" x1={x} x2={x} y1={scale.y(p.systolic)} y2={scale.y(p.diastolic)} />
              <circle className="vtc-dot is-hollow" cx={x} cy={scale.y(p.systolic)} r={n > 31 ? 2.5 : 5} />
              <circle className="vtc-dot" cx={x} cy={scale.y(p.diastolic)} r={n > 31 ? 2 : 4} />
            </g>
          );
        })}
      </svg>
      <XLabels labels={labels} />
    </ChartFigure>
  );
}

export interface LineSeriesData {
  key: string;
  values: ReadonlyArray<number | null>;
  hollow?: boolean;
  tone: Tone;
}

/** Daily averages as lines + dots over the target band (nbl_Vitals_GlucoseReport, heart rate). */
export function VitalLineChart({
  series,
  labels,
  band,
  label,
  table,
  fmt,
  height = 170,
}: {
  series: readonly LineSeriesData[];
  labels: readonly string[];
  band: readonly [number, number];
  label: string;
  table: DataTable;
  fmt: (n: number) => string;
  height?: number;
}) {
  const values = series.flatMap((s) => s.values.filter((v): v is number => v !== null));
  const scale = makeScale(values, band, height);
  const n = labels.length;
  return (
    <ChartFigure label={label} table={table} className="vtc">
      <svg viewBox={`0 0 ${CHART_W} ${height}`} width="100%" className="axc-svg vtc-svg" direction="ltr" aria-hidden focusable="false">
        <Band scale={scale} band={band} />
        <Grid scale={scale} fmt={fmt} />
        {series.map((s) => (
          <g key={s.key} className={`nb-tone-${s.tone}`}>
            {/* One line per series (fasting, after meal…), skipping the days it has no reading. */}
            <path className="vtc-line" d={linePath(s.values.map((v, i) => (v === null ? undefined : { x: slotX(i, n), y: scale.y(v) })).filter((p): p is { x: number; y: number } => !!p))} />
            {s.values.map((v, i) =>
              v === null ? null : (
                <circle key={i} className={clsx('vtc-dot', s.hollow && 'is-hollow')} cx={slotX(i, n)} cy={scale.y(v)} r={n > 31 ? 2.5 : 5} />
              ),
            )}
          </g>
        ))}
      </svg>
      <XLabels labels={labels} />
    </ChartFigure>
  );
}

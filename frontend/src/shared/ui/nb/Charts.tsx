import { clsx } from 'clsx';

import { bandPath, barRects, domainOf, linePath, xAt, yAt, type PlotBox } from './chart-geometry';
import { toneClass, type Tone } from './tone';

const WIDTH = 320;

export interface LineSeries {
  values: ReadonlyArray<number | null>;
  tone?: Tone;
  dashed?: boolean;
  /** Draw a dot on every value. */
  points?: boolean;
}

interface LineChartProps {
  /** Accessible name — what the chart shows, in words («دمای پایه بدن، ۳۰ روز اخیر»). */
  label: string;
  series: readonly LineSeries[];
  /** x-axis labels, oldest first; drawn under evenly spaced points. */
  xLabels?: readonly string[];
  /** Horizontal grid lines + their labels (already locale-formatted in `yLabels`). */
  yTicks?: readonly number[];
  yLabels?: readonly string[];
  min?: number;
  max?: number;
  /** Shaded range, e.g. growth P3–P97 or a normal band. */
  band?: { lower: readonly number[]; upper: readonly number[]; tone?: Tone };
  /** Index of «today» — glowing data dot on the first series. */
  highlightIndex?: number;
  height?: number;
  className?: string;
}

/**
 * Simple SVG line chart. The plot is `direction: ltr` in both locales (time
 * runs left → right); labels are passed in already localized.
 */
export function LineChart({
  label,
  series,
  xLabels,
  yTicks,
  yLabels,
  min,
  max,
  band,
  highlightIndex,
  height = 160,
  className,
}: LineChartProps) {
  const box: PlotBox = { width: WIDTH, height, pad: [10, 8, xLabels ? 20 : 8, yTicks ? 30 : 8] };
  const all = series.map((s) => s.values);
  if (band) all.push(band.lower, band.upper);
  const domain = domainOf(all, { min, max });
  const first = series[0]?.values ?? [];
  const hi = highlightIndex !== undefined ? first[highlightIndex] : null;

  return (
    <svg
      role="img"
      aria-label={label}
      viewBox={`0 0 ${WIDTH} ${height}`}
      width="100%"
      className={clsx('nb-chart', className)}
    >
      {yTicks?.map((tick, i) => {
        const y = yAt(tick, domain, box);
        return (
          <g key={tick}>
            <line className="nb-chart-grid" x1={box.pad[3]} x2={WIDTH - box.pad[1]} y1={y} y2={y} />
            {yLabels?.[i] ? (
              <text className="nb-chart-axis" x={box.pad[3] - 6} y={y + 3} textAnchor="end">
                {yLabels[i]}
              </text>
            ) : null}
          </g>
        );
      })}
      {band ? (
        <path className={clsx('nb-chart-band', toneClass(band.tone ?? 'data'))} d={bandPath(band.lower, band.upper, domain, box)} />
      ) : null}
      {series.map((s, si) => (
        <g key={si} className={toneClass(s.tone ?? 'data')}>
          <path className={clsx('nb-chart-line', s.dashed && 'is-dashed')} d={linePath(s.values, domain, box)} />
          {s.points
            ? s.values.map((v, i) =>
                v === null ? null : (
                  <circle
                    key={i}
                    className="nb-chart-point"
                    cx={xAt(i, s.values.length, box)}
                    cy={yAt(v, domain, box)}
                    r={3}
                  />
                ),
              )
            : null}
        </g>
      ))}
      {hi !== null && hi !== undefined && highlightIndex !== undefined ? (
        <circle
          className="nb-chart-today"
          cx={xAt(highlightIndex, first.length, box)}
          cy={yAt(hi, domain, box)}
          r={5}
        />
      ) : null}
      {xLabels?.map((text, i) =>
        text ? (
          <text
            key={i}
            className="nb-chart-axis"
            x={xAt(i, xLabels.length, box)}
            y={height - 5}
            textAnchor="middle"
          >
            {text}
          </text>
        ) : null,
      )}
    </svg>
  );
}

export interface Bar {
  /** Axis label under the bar (already localized). */
  label: string;
  value: number;
  tone?: Tone;
  /** Solid instead of the soft tint — the current / selected bar. */
  highlight?: boolean;
  /** Text above the bar, e.g. «۲۹». */
  valueLabel?: string;
}

interface BarChartProps {
  label: string;
  bars: readonly Bar[];
  max?: number;
  height?: number;
  className?: string;
}

/** Simple SVG bar chart (cycle lengths, feeds, kicks), `direction: ltr`. */
export function BarChart({ label, bars, max, height = 150, className }: BarChartProps) {
  const box: PlotBox = { width: WIDTH, height, pad: [16, 4, 20, 4] };
  const top = max ?? Math.max(1, ...bars.map((b) => b.value));
  const rects = barRects(
    bars.map((b) => b.value),
    top,
    box,
  );
  return (
    <svg role="img" aria-label={label} viewBox={`0 0 ${WIDTH} ${height}`} width="100%" className={clsx('nb-chart', className)}>
      <line className="nb-chart-grid" x1={box.pad[3]} x2={WIDTH - box.pad[1]} y1={height - box.pad[2]} y2={height - box.pad[2]} />
      {bars.map((bar, i) => {
        const rect = rects[i];
        return (
          <g key={i} className={toneClass(bar.tone ?? 'brand')}>
            <rect
              className={clsx('nb-chart-bar', bar.highlight && 'is-highlight')}
              x={rect.x}
              y={rect.y}
              width={rect.width}
              height={rect.height}
              rx={Math.min(6, rect.width / 2)}
            />
            {bar.valueLabel ? (
              <text className="nb-chart-value" x={rect.x + rect.width / 2} y={rect.y - 4} textAnchor="middle">
                {bar.valueLabel}
              </text>
            ) : null}
            <text className="nb-chart-axis" x={rect.x + rect.width / 2} y={height - 5} textAnchor="middle">
              {bar.label}
            </text>
          </g>
        );
      })}
    </svg>
  );
}

"use client";

import clsx from "clsx";
import { useLocale, useTranslations } from "next-intl";
import { useId } from "react";

import { type BbtCycle, formatBbt } from "@/entities/fertility";
import type { Locale } from "@/shared/i18n";
import { formatDayMonth, formatNumber, fromApiDate } from "@/shared/lib/date";

import {
  areaPath,
  buildScales,
  coverlineY,
  fertileBandX,
  linePath,
} from "../model/scale";

interface Props {
  /** `cycles[0]` is the current cycle; the rest are overlaid muted. */
  cycles: readonly BbtCycle[];
  /** `Y-m-d` of today — the matching point is highlighted. */
  today: string;
}

/**
 * «دمای پایه» chart (`v19_TTC_BBT`): inline SVG, token colours only, drawn LTR
 * (cycle day grows rightward in both directions, like every clinical BBT chart).
 * Identity is never colour-alone: a legend and a visually hidden table ship with it.
 *
 * Privacy (§11): temperatures are rendered, never logged.
 */
export function BbtChart({ cycles, today }: Props) {
  const t = useTranslations("fertility.bbt");
  const locale = useLocale() as Locale;
  const gradId = useId();
  const current = cycles[0];
  if (!current) return null;
  const scales = buildScales(cycles);
  const { frame, domain } = scales;
  const band = fertileBandX(scales, current.fertileWindow);
  const cover = coverlineY(scales, current.coverline);
  const plotRight = frame.width - frame.right;
  const floor = frame.height - frame.bottom;
  const previous = cycles.slice(1);
  const points = [...current.points].sort((a, b) => a.cycleDay - b.cycleDay);

  return (
    <figure className="flex flex-col gap-3">
      <div dir="ltr">
        <svg
          viewBox={`0 0 ${frame.width} ${frame.height}`}
          className="h-auto w-full"
          role="img"
          aria-label={t("chart.label")}
        >
          <defs>
            <linearGradient id={gradId} x1="0" y1="0" x2="0" y2="1">
              <stop
                offset="0%"
                className="[stop-color:var(--fert-teal)] [stop-opacity:0.28]"
              />
              <stop
                offset="100%"
                className="[stop-color:var(--fert-teal)] [stop-opacity:0]"
              />
            </linearGradient>
          </defs>
          {band && (
            <rect
              x={band.x}
              y={frame.top}
              width={band.width}
              height={floor - frame.top}
              className="fill-(--fert-amber-band)"
            />
          )}
          {domain.ticks.map((v) => (
            <g key={v}>
              <line
                x1={frame.left}
                x2={plotRight}
                y1={scales.y(v)}
                y2={scales.y(v)}
                className="stroke-(--fert-chart-grid)"
                strokeWidth={1}
              />
              <text
                x={frame.left - 6}
                y={scales.y(v)}
                textAnchor="end"
                dominantBaseline="middle"
                className="fill-(--fert-chart-label) text-[9px]"
              >
                {formatNumber(v.toFixed(1), locale)}
              </text>
            </g>
          ))}
          {scales.xTicks.map((d) => (
            <text
              key={d}
              x={scales.x(d)}
              y={floor + 16}
              textAnchor="middle"
              className="fill-(--fert-chart-label) text-[9px]"
            >
              {formatNumber(d, locale)}
            </text>
          ))}
          {previous.map((c, i) => (
            <path
              key={c.startDate ?? `prev-${i}`}
              d={linePath(scales, c.points)}
              fill="none"
              className="stroke-(--fert-chart-label) opacity-40"
              strokeWidth={1.5}
              strokeLinejoin="round"
            />
          ))}
          {points.length > 0 && (
            <path d={areaPath(scales, points)} fill={`url(#${gradId})`} />
          )}
          {cover !== null && (
            <line
              x1={frame.left}
              x2={plotRight}
              y1={cover}
              y2={cover}
              className="stroke-(--fert-amber)"
              strokeWidth={1.5}
              strokeDasharray="5 4"
            />
          )}
          <path
            d={linePath(scales, points)}
            fill="none"
            className="stroke-(--fert-teal)"
            strokeWidth={2}
            strokeLinejoin="round"
            strokeLinecap="round"
          />
          {points.map((p) => {
            const isToday = p.date === today;
            return (
              <circle
                key={p.cycleDay}
                cx={scales.x(p.cycleDay)}
                cy={scales.y(p.value)}
                r={isToday ? 6 : 4}
                className={clsx(
                  isToday
                    ? "fill-(--fert-teal) stroke-(--surface)"
                    : "fill-(--fert-chart-point) stroke-(--fert-teal)",
                )}
                strokeWidth={2}
              >
                <title>{`${t("chart.dayColumn")} ${formatNumber(p.cycleDay, locale)}: ${formatBbt(p.value, locale)}°C`}</title>
              </circle>
            );
          })}
        </svg>
      </div>

      <figcaption className="flex flex-wrap gap-x-4 gap-y-1.5 text-[11px] text-(--ink-3)">
        <LegendItem
          swatch="h-2.5 w-2.5 rounded-full bg-(--fert-teal)"
          label={t("legend.logged")}
        />
        <LegendItem
          swatch="h-0 w-4 border-t-2 border-dashed border-(--fert-amber)"
          label={t("legend.baseline")}
        />
        <LegendItem
          swatch="h-2.5 w-4 rounded-sm bg-(--fert-amber-band)"
          label={t("legend.window")}
        />
        {previous.length > 0 && (
          <LegendItem
            swatch="h-0 w-4 border-t-2 border-(--fert-chart-label) opacity-40"
            label={t("legend.previous")}
          />
        )}
      </figcaption>

      <table className="sr-only">
        <caption>{t("chart.tableCaption")}</caption>
        <thead>
          <tr>
            <th scope="col">{t("chart.dayColumn")}</th>
            <th scope="col">{t("chart.dateColumn")}</th>
            <th scope="col">{t("chart.valueColumn")}</th>
          </tr>
        </thead>
        <tbody>
          {points.map((p) => (
            <tr key={p.cycleDay}>
              <td>{formatNumber(p.cycleDay, locale)}</td>
              <td>{formatDayMonth(fromApiDate(p.date), locale)}</td>
              <td>{formatBbt(p.value, locale)}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </figure>
  );
}

function LegendItem({ swatch, label }: { swatch: string; label: string }) {
  return (
    <span className="flex items-center gap-1.5">
      <span className={clsx("inline-block shrink-0", swatch)} aria-hidden />
      {label}
    </span>
  );
}

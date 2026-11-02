"use client";

import { useLocale, useTranslations } from "next-intl";

import { formatNumber } from "@/shared/lib";

/** Cycle lengths (days, oldest first) as bars with a table-free text alternative; lengths only, no dates. */
export function CycleLengthChart({ lengths }: { lengths: number[] }) {
  const t = useTranslations("users");
  const locale = useLocale();
  const shown = lengths.slice(-12);
  const max = Math.max(45, ...shown);
  const W = 320;
  const H = 120;
  const gap = 6;
  const bw = (W - gap * (shown.length + 1)) / Math.max(shown.length, 1);
  return (
    <figure className="m-0">
      <svg
        viewBox={`0 0 ${W} ${H + 18}`}
        role="img"
        aria-label={t("cycleChartLabel", {
          list: shown.map((n) => formatNumber(n, locale)).join(", "),
        })}
        className="h-auto w-full max-w-md"
      >
        <line x1="0" x2={W} y1={H} y2={H} stroke="var(--line)" />
        {shown.map((n, i) => {
          const h = (n / max) * (H - 14);
          const x = gap + i * (bw + gap);
          return (
            <g key={i}>
              <rect
                x={x}
                y={H - h}
                width={bw}
                height={h}
                rx="3"
                fill="var(--data-deep)"
              />
              <text
                x={x + bw / 2}
                y={H + 13}
                textAnchor="middle"
                fontSize="10"
                fill="var(--ink-3)"
              >
                {formatNumber(n, locale)}
              </text>
            </g>
          );
        })}
      </svg>
      <figcaption className="mt-1 text-[13px] text-ink-3">
        {t("cycleChartCaption", { count: shown.length })}
      </figcaption>
    </figure>
  );
}

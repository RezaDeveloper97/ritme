"use client";

import clsx from "clsx";
import { useLocale, useTranslations } from "next-intl";
import { type ReactNode, useState } from "react";

import { useCycleToday } from "@/entities/cycle";
import {
  type FertilityInsights,
  useFertilityInsights,
} from "@/entities/fertility";
import { type Locale, Link, useDirection } from "@/shared/i18n";
import {
  formatDayMonth,
  formatMonthLabel,
  formatNumber,
  fromApiDate,
  toApiDate,
  today,
  toParts,
  weekdayLabels,
} from "@/shared/lib/date";
import { AppSheet } from "@/shared/sheet";
import { Icon } from "@/shared/ui";

import {
  CONFIDENCE_TONE,
  STRENGTH_TONE,
  calendarMark,
  evidenceIcon,
  evidenceTone,
  historyStrip,
  isLowData,
  nextStarts,
  windowMonths,
  windowWeeks,
} from "../model/view";

const BACK_HREF = "/home";
const LOG_HREF = "/fertility/log";
/** Engine default when the cycle view has no period length yet. */
const DEFAULT_PERIOD_DAYS = 5;

/**
 * «پیش‌بینی‌ها» — fertility insights (`v19_TTC_Insights` / `nb2_TTC_Insights`),
 * route `/fertility/insights`. Privacy (§11): nothing here is logged.
 */
export function FertilityInsightsPage() {
  const t = useTranslations("fertility");
  const locale = useLocale() as Locale;
  const dir = useDirection();
  const query = useFertilityInsights();
  const [tipsOpen, setTipsOpen] = useState(false);
  const data = query.data;

  let body: ReactNode;
  if (query.isPending) {
    body = (
      <p className="py-10 text-center text-[14px] text-(--muted)">
        {t("loading")}
      </p>
    );
  } else if (query.isError || !data) {
    body = (
      <div className="flex flex-col items-center gap-3 py-10">
        <p className="text-center text-[14px] text-(--muted)">
          {t("loadError")}
        </p>
        <button
          type="button"
          className="rounded-full bg-(--fert-chip-on-bg) px-5 py-2 text-[14px] font-bold text-(--brand)"
          onClick={() => void query.refetch()}
        >
          {t("retry")}
        </button>
      </div>
    );
  } else if (isLowData(data)) {
    body = (
      <>
        <section className="fert-card flex flex-col items-center gap-3 px-4 py-8 text-center">
          <span className="fert-disc fert-tone-violet grid size-14 place-items-center rounded-full">
            <Icon name="sparkle" size={26} />
          </span>
          <h2 className="text-[16px] font-extrabold text-(--ink)">
            {t("insights.lowData.title")}
          </h2>
          <p className="text-[13px] leading-6 text-(--muted)">
            {t("insights.lowData.body")}
          </p>
          <p className="text-[12px] text-(--muted)">
            {t("insights.window.disclaimer")}
          </p>
          <Link href={LOG_HREF} className="btn btn-primary mt-1">
            {t("insights.logCta")}
          </Link>
        </section>
        {data.evidence.length > 0 && <EvidenceCard data={data} />}
      </>
    );
  } else {
    body = (
      <>
        <WindowCard data={data} locale={locale} />
        <EvidenceCard data={data} />
        {data.history.length > 0 && <HistoryCard data={data} locale={locale} />}
        <button
          type="button"
          className="fert-improve"
          onClick={() => setTipsOpen(true)}
        >
          <Icon name="sparkle" size={18} />
          {t("insights.improve")}
        </button>
      </>
    );
  }

  return (
    <div className="view">
      <div className="scroll">
        <header className="rmd-hdr">
          <Link href={BACK_HREF} className="rmd-hdr-btn" aria-label={t("back")}>
            <Icon
              name={dir === "rtl" ? "chevronRight" : "chevronLeft"}
              size={20}
              strokeWidth={1.8}
            />
          </Link>
          <div className="rmd-hdr-text">
            <h1 className="rmd-hdr-title">{t("insights.title")}</h1>
            {data && (
              <p className="rmd-hdr-sub">
                {data.cyclesUsed === 0
                  ? t("insights.noCycles")
                  : t("insights.basedOn", {
                      count: formatNumber(data.cyclesUsed, locale),
                    })}
              </p>
            )}
          </div>
          <span className="rmd-hdr-btn invisible" aria-hidden />
        </header>
        <div className="flex flex-col gap-3.5 px-4 pt-1 pb-32">{body}</div>
      </div>

      <AppSheet
        open={tipsOpen}
        onClose={() => setTipsOpen(false)}
        size="half"
        title={t("insights.improve")}
      >
        <div className="flex flex-col gap-4 pb-2">
          {data && data.tips.length > 0 && (
            <ul className="flex list-disc flex-col gap-1 ps-5 text-start text-[14px] leading-7 text-(--ink-3)">
              {data.tips.map((tip) => (
                <li key={tip}>{tip}</li>
              ))}
            </ul>
          )}
          <Link href={LOG_HREF} className="btn btn-ghost w-full">
            {t("insights.logCta")}
          </Link>
        </div>
      </AppSheet>
    </div>
  );
}

function WindowCard({
  data,
  locale,
}: {
  data: FertilityInsights;
  locale: Locale;
}) {
  const t = useTranslations("fertility.insights");
  const w = data.window;
  if (!w) return null;
  const start = formatDayMonth(fromApiDate(w.start), locale);
  const end = formatDayMonth(fromApiDate(w.end), locale);
  const summary = w.ovulation
    ? t("window.range", {
        start,
        end,
        ovulation: formatDayMonth(fromApiDate(w.ovulation), locale),
      })
    : t("window.rangeNoOvulation", { start, end });
  const todayIso = toApiDate(today());
  const weeks = windowWeeks(w, locale);
  const caption = windowMonths(w, locale)
    .map(({ year, month }) => formatMonthLabel(year, month, locale))
    .join(" · ");
  const showsToday = weeks.some((week) => week.some((d) => toApiDate(d) === todayIso));

  return (
    <section className="fert-card flex flex-col gap-3.5">
      <div className="flex items-center justify-between gap-2">
        <h2 className="fert-overline">{t("window.title")}</h2>
        {data.confidence && (
          <span
            className={clsx(
              "fert-pill",
              `fert-tone-${CONFIDENCE_TONE[data.confidence]}`,
            )}
          >
            {t(`confidence.${data.confidence}`)}
          </span>
        )}
      </div>
      <p className="text-start text-[19px] leading-[1.7] font-extrabold text-(--ink)">
        {summary}
      </p>

      <div aria-label={t("calendarLabel")} className="flex flex-col gap-1.5">
        <p className="text-center text-[12px] font-bold text-(--muted)">
          {caption}
        </p>
        <div className="grid grid-cols-7 text-center text-[10.5px] font-bold text-(--muted)">
          {weekdayLabels(locale).map((d) => (
            <span key={d}>{d}</span>
          ))}
        </div>
        {weeks.map((week) => (
          <div key={toApiDate(week[0])} className="grid grid-cols-7 gap-y-1.5">
            {week.map((date) => {
              const iso = toApiDate(date);
              const mark = calendarMark(iso, w, todayIso);
              return (
                <span
                  key={iso}
                  className={clsx("fert-cal-day", mark && `is-${mark}`)}
                >
                  {formatNumber(toParts(date, locale).day, locale)}
                </span>
              );
            })}
          </div>
        ))}
        <div className="mt-1 flex flex-wrap items-center justify-center gap-4 text-[11px] text-(--muted)">
          <span className="flex items-center gap-1.5">
            <span className="fert-cal-swatch is-window" />
            {t("window.legendWindow")}
          </span>
          <span className="flex items-center gap-1.5">
            <span className="fert-cal-swatch is-ovulation" />
            {t("window.legendOvulation")}
          </span>
          {showsToday && (
            <span className="flex items-center gap-1.5">
              <span className="fert-cal-swatch is-today" />
              {t("window.legendToday")}
            </span>
          )}
        </div>
      </div>

      <p className="fert-disclaimer">{t("window.disclaimer")}</p>
    </section>
  );
}

function EvidenceCard({ data }: { data: FertilityInsights }) {
  const t = useTranslations("fertility.insights");
  if (data.evidence.length === 0) return null;
  return (
    <section className="fert-card flex flex-col gap-3.5">
      <h2 className="text-start text-[15px] font-extrabold text-(--ink)">
        {t("evidence.title")}
      </h2>
      <ul className="flex flex-col gap-3.5">
        {data.evidence.map((row) => (
          <li key={row.key} className="flex items-center gap-3">
            <span
              className={clsx(
                "fert-disc grid size-10 shrink-0 place-items-center rounded-full",
                `fert-tone-${evidenceTone(row.key)}`,
              )}
            >
              <Icon name={evidenceIcon(row.key)} size={18} strokeWidth={2.2} />
            </span>
            <div className="flex min-w-0 flex-1 flex-col text-start">
              <span className="text-[13.5px] font-extrabold text-(--ink)">
                {row.title}
              </span>
              {row.detail && (
                <span className="text-[12px] font-semibold text-(--ink-3)">
                  {row.detail}
                </span>
              )}
            </div>
            {row.strength && (
              <span
                className={clsx(
                  "fert-pill",
                  `fert-tone-${STRENGTH_TONE[row.strength]}`,
                )}
              >
                {t(`evidence.strength.${row.strength}`)}
              </span>
            )}
          </li>
        ))}
      </ul>
    </section>
  );
}

/**
 * «تخمک‌گذاری در سیکل‌های قبل»: one day strip per finished cycle (audit #2) —
 * period red, fertile days amber, ovulation turquoise, PMS violet — plus the
 * ovulation day. A row the strip can't be drawn for keeps the plain day.
 */
function HistoryCard({
  data,
  locale,
}: {
  data: FertilityInsights;
  locale: Locale;
}) {
  const t = useTranslations("fertility.insights");
  const cycle = useCycleToday().data?.cycleView ?? null;
  const currentStart = cycle?.anchors?.currentPeriodStart ?? null;
  // Only for a cycle without a logged period end: each row carries its own `periodDays`.
  const fallbackPeriodLength =
    cycle?.metrics?.effectivePeriodLength ??
    cycle?.effectiveValues.periodDuration ??
    DEFAULT_PERIOD_DAYS;
  const next = nextStarts(data.history, currentStart);

  return (
    <section className="fert-card flex flex-col gap-3">
      <h2 className="text-start text-[15px] font-extrabold text-(--ink)">
        {t("history.title")}
      </h2>
      <ul className="flex flex-col gap-3">
        {data.history.map((row, i) => {
          const strip = historyStrip(row, next[i], fallbackPeriodLength);
          return (
            <li
              key={`${row.cycleStart ?? row.monthLabel}-${i}`}
              className="flex items-center gap-2.5"
            >
              <span className="w-15 shrink-0 text-start text-[12px] font-bold text-(--ink-3)">
                {row.monthLabel}
              </span>
              {strip ? (
                <span className="fert-strip" aria-hidden>
                  {strip.map((kind, d) => (
                    <span key={d} className={clsx("fert-strip-dot", `is-${kind}`)} />
                  ))}
                </span>
              ) : (
                <span className="flex-1" />
              )}
              <span className="shrink-0 text-[12px] font-extrabold whitespace-nowrap text-(--fert-teal)">
                {row.ovulationDay === null
                  ? t("history.unknown")
                  : t("history.day", {
                      day: formatNumber(row.ovulationDay, locale),
                    })}
              </span>
            </li>
          );
        })}
      </ul>
    </section>
  );
}

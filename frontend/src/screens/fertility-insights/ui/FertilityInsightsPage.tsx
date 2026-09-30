"use client";

import clsx from "clsx";
import { useLocale, useTranslations } from "next-intl";
import { type ReactNode, useState } from "react";

import { useCycleToday } from "@/entities/cycle";
import {
  type FertilityInsights,
  useFertilityInsights,
} from "@/entities/fertility";
import { type Locale, Link, useRouter } from "@/shared/i18n";
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
} from "@/shared/ui";

import {
  CONFIDENCE_TONE,
  STRENGTH_TONE,
  calendarMark,
  evidenceIcon,
  evidenceTone,
  historyStrip,
  isLowData,
  nextStarts,
  gridMonths,
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
  const router = useRouter();
  const query = useFertilityInsights();
  const [tipsOpen, setTipsOpen] = useState(false);
  const data = query.data;

  let body: ReactNode;
  if (query.isPending) {
    body = (
      <SkeletonGroup label={t("loading")}>
        <Skeleton shape="card" className="ttc-skel-tall" />
        <Skeleton shape="card" />
        <Skeleton shape="card" />
      </SkeletonGroup>
    );
  } else if (query.isError || !data) {
    body = (
      <Card className="ttc-state" role="alert">
        <span className="ttc-state-disc" aria-hidden>
          <Icon name="warning" size={24} />
        </span>
        <p className="ttc-state-text">{t("loadError")}</p>
        <SecondaryButton
          icon="refresh"
          block={false}
          onClick={() => void query.refetch()}
        >
          {t("retry")}
        </SecondaryButton>
      </Card>
    );
  } else if (isLowData(data)) {
    body = (
      <>
        <Card>
          <EmptyState
            icon="sparkle"
            className="ttc-empty"
            title={t("insights.lowData.title")}
            body={
              <>
                {t("insights.lowData.body")}
                <span className="ttc-empty-note">{t("insights.window.disclaimer")}</span>
              </>
            }
            action={
              <PrimaryButton onClick={() => router.push(LOG_HREF)}>
                {t("insights.logCta")}
              </PrimaryButton>
            }
          />
        </Card>
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
          className="nb-btn is-block fert-improve"
          onClick={() => setTipsOpen(true)}
        >
          <Icon name="sparkle" size={18} />
          {t("insights.improve")}
        </button>
      </>
    );
  }

  return (
    <div className="view fert-page">
      <div className="scroll ttc-screen">
        <SkyLayer />
        <ScreenHeader
          title={t("insights.title")}
          subtitle={
            data
              ? data.cyclesUsed === 0
                ? t("insights.noCycles")
                : t("insights.basedOn", {
                    count: formatNumber(data.cyclesUsed, locale),
                  })
              : undefined
          }
          onBack={() => router.push(BACK_HREF)}
          backLabel={t("back")}
        />
        <div className="ttc-body">{body}</div>
      </div>

      <AppSheet
        open={tipsOpen}
        onClose={() => setTipsOpen(false)}
        size="half"
        title={t("insights.improve")}
      >
        <div className="flex flex-col gap-4 pb-2">
          {data && data.tips.length > 0 && (
            <ul className="ttc-sheet-list">
              {data.tips.map((tip) => (
                <li key={tip}>{tip}</li>
              ))}
            </ul>
          )}
          <Link href={LOG_HREF} className="nb-btn is-outline is-block">
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
  const caption = gridMonths(weeks, locale)
    .map(({ year, month }) => formatMonthLabel(year, month, locale))
    .join(" · ");
  const showsToday = weeks.some((week) => week.some((d) => toApiDate(d) === todayIso));

  return (
    <Card as="section" padding="lg" className="ttc-stack">
      <div className="ttc-row-between">
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
      <p className="ttc-window-summary">
        {summary}
      </p>

      <div role="group" aria-label={t("calendarLabel")} className="ttc-cal">
        <p className="ttc-cal-caption">
          {caption}
        </p>
        <div className="ttc-cal-week is-head">
          {weekdayLabels(locale).map((d) => (
            <span key={d}>{d}</span>
          ))}
        </div>
        {weeks.map((week) => (
          <div key={toApiDate(week[0])} className="ttc-cal-week">
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
        <div className="ttc-cal-legend">
          <span className="ttc-legend-item">
            <span className="fert-cal-swatch is-window" />
            {t("window.legendWindow")}
          </span>
          <span className="ttc-legend-item">
            <span className="fert-cal-swatch is-ovulation" />
            {t("window.legendOvulation")}
          </span>
          {showsToday && (
            <span className="ttc-legend-item">
              <span className="fert-cal-swatch is-today" />
              {t("window.legendToday")}
            </span>
          )}
        </div>
      </div>

      <p className="fert-disclaimer">{t("window.disclaimer")}</p>
    </Card>
  );
}

function EvidenceCard({ data }: { data: FertilityInsights }) {
  const t = useTranslations("fertility.insights");
  if (data.evidence.length === 0) return null;
  return (
    <Card as="section" padding="lg" className="ttc-stack">
      <h2 className="ttc-card-title">{t("evidence.title")}</h2>
      <ul className="ttc-evidence">
        {data.evidence.map((row) => (
          <li key={row.key} className="ttc-evidence-row">
            <span
              className={clsx(
                "fert-disc ttc-evidence-disc",
                `fert-tone-${evidenceTone(row.key)}`,
              )}
            >
              <Icon name={evidenceIcon(row.key)} size={18} strokeWidth={2.2} />
            </span>
            <div className="ttc-evidence-text">
              <span className="ttc-evidence-title">{row.title}</span>
              {row.detail && (
                <span className="ttc-evidence-detail">
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
    </Card>
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
    <Card as="section" padding="lg" className="ttc-stack">
      <h2 className="ttc-card-title">{t("history.title")}</h2>
      <ul className="ttc-history">
        {data.history.map((row, i) => {
          const strip = historyStrip(row, next[i], fallbackPeriodLength);
          return (
            <li
              key={`${row.cycleStart ?? row.monthLabel}-${i}`}
              className="ttc-history-row"
            >
              <span className="ttc-history-month">
                {row.monthLabel}
              </span>
              {strip ? (
                <span className="fert-strip" aria-hidden>
                  {strip.map((kind, d) => (
                    <span key={d} className={clsx("fert-strip-dot", `is-${kind}`)} />
                  ))}
                </span>
              ) : (
                <span className="ttc-grow" />
              )}
              <span className="ttc-history-day">
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
    </Card>
  );
}

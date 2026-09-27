"use client";

import clsx from "clsx";
import { useLocale, useTranslations } from "next-intl";
import type { ReactNode } from "react";

import {
  type EvidenceStrength,
  type FertilityInsights,
  useFertilityInsights,
} from "@/entities/fertility";
import { type Locale, Link, useDirection, useRouter } from "@/shared/i18n";
import {
  formatDayMonth,
  formatMonthLabel,
  formatNumber,
  fromApiDate,
  monthMatrix,
  toApiDate,
  weekdayLabels,
} from "@/shared/lib/date";
import { Icon } from "@/shared/ui";

import { evidenceIcon, isLowData, markFor, windowMonths } from "../model/view";

const BACK_HREF = "/home";
const LOG_HREF = "/fertility/log";

const STRENGTH_CLASS: Record<EvidenceStrength, string> = {
  strong: "fert-tone-green",
  medium: "fert-tone-amber",
  none: "fert-tone-rose",
};

/**
 * «پیش‌بینی‌ها» — fertility insights (`v19_TTC_Insights` / `nb2_TTC_Insights`),
 * route `/fertility/insights`. Privacy (§11): nothing here is logged.
 */
export function FertilityInsightsPage() {
  const t = useTranslations("fertility");
  const locale = useLocale() as Locale;
  const dir = useDirection();
  const router = useRouter();
  const query = useFertilityInsights();
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
        <section className="card flex flex-col items-center gap-3 px-4 py-8 text-center">
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
        {data.history.length > 0 && (
          <section className="card flex flex-col gap-2 p-4">
            <h2 className="text-start text-[15px] font-extrabold text-(--ink)">
              {t("insights.history.title")}
            </h2>
            <ul className="flex flex-col">
              {data.history.map((row, i) => (
                <li
                  key={`${row.monthLabel}-${i}`}
                  className="flex items-center justify-between border-b border-(--line) py-2 text-[13px] last:border-b-0"
                >
                  <span className="text-(--ink-3)">{row.monthLabel}</span>
                  <span className="font-bold text-(--ink)">
                    {row.ovulationDay === null
                      ? t("insights.history.unknown")
                      : t("insights.history.day", {
                          day: formatNumber(row.ovulationDay, locale),
                        })}
                  </span>
                </li>
              ))}
            </ul>
          </section>
        )}
        <TipsCard tips={data.tips} />
      </>
    );
  }

  return (
    <div className="view">
      <div className="scroll">
        <header className="flex items-center gap-3 px-4 pt-4 pb-3">
          <button
            type="button"
            className="iconbtn"
            aria-label={t("back")}
            onClick={() => router.push(BACK_HREF)}
          >
            <Icon
              name={dir === "rtl" ? "chevronRight" : "chevronLeft"}
              size={22}
            />
          </button>
          <div className="flex min-w-0 flex-1 flex-col text-start">
            <h1 className="text-[18px] font-extrabold text-(--ink)">
              {t("insights.title")}
            </h1>
            {data && (
              <p className="text-[13px] text-(--muted)">
                {data.cyclesUsed === 0
                  ? t("insights.noCycles")
                  : t("insights.basedOn", {
                      count: formatNumber(data.cyclesUsed, locale),
                    })}
              </p>
            )}
          </div>
        </header>
        <div className="flex flex-col gap-3 px-4 pb-32">{body}</div>
      </div>
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
  const months = windowMonths(w, locale);

  return (
    <section className="card flex flex-col gap-3 p-4">
      <div className="flex items-center justify-between gap-2">
        <h2 className="text-[15px] font-extrabold text-(--ink)">
          {t("window.title")}
        </h2>
        {data.confidence && (
          <span className="rounded-full bg-(--fert-chip-on-bg) px-3 py-1 text-[12px] font-bold text-(--brand)">
            {t(`confidence.${data.confidence}`)}
          </span>
        )}
      </div>
      <p className="text-start text-[13px] leading-6 text-(--ink-3)">
        {summary}
      </p>

      <div aria-label={t("calendarLabel")} className="flex flex-col gap-3">
        {months.map(({ year, month }) => (
          <div key={`${year}-${month}`} className="flex flex-col gap-1">
            <p className="text-center text-[13px] font-bold text-(--ink)">
              {formatMonthLabel(year, month, locale)}
            </p>
            <div className="grid grid-cols-7 gap-1 text-center text-[11px] text-(--muted)">
              {weekdayLabels(locale).map((d) => (
                <span key={d}>{d}</span>
              ))}
            </div>
            {monthMatrix(year, month, locale).map((week, wi) => (
              <div key={wi} className="grid grid-cols-7 gap-1">
                {week.map((cell, ci) => {
                  if (!cell) return <span key={ci} />;
                  const mark = markFor(toApiDate(cell.date), w);
                  return (
                    <span
                      key={ci}
                      className={clsx(
                        "grid aspect-square place-items-center rounded-full text-[12px]",
                        mark === "ovulation" &&
                          "bg-(--fert-teal) font-extrabold text-(--on-brand)",
                        mark === "window" &&
                          "bg-(--fert-teal-soft) font-bold text-(--fert-teal)",
                        mark === null && "text-(--ink-3)",
                      )}
                    >
                      {formatNumber(cell.day, locale)}
                    </span>
                  );
                })}
              </div>
            ))}
          </div>
        ))}
        <div className="mt-1 flex flex-wrap items-center justify-center gap-4 text-[11px] text-(--muted)">
          <span className="flex items-center gap-1">
            <span className="size-3 rounded-full bg-(--fert-teal-soft)" />
            {t("window.legendWindow")}
          </span>
          <span className="flex items-center gap-1">
            <span className="size-3 rounded-full bg-(--fert-teal)" />
            {t("window.legendOvulation")}
          </span>
        </div>
      </div>

      <p className="flex items-start gap-2 rounded-xl bg-(--fert-note-bg) p-3 text-start text-[12px] leading-5 text-(--ink-3)">
        <Icon name="info" size={16} />
        <span>{t("window.disclaimer")}</span>
      </p>
    </section>
  );
}

function EvidenceCard({ data }: { data: FertilityInsights }) {
  const t = useTranslations("fertility.insights");
  if (data.evidence.length === 0) return null;
  return (
    <section className="card flex flex-col gap-2 p-4">
      <h2 className="text-start text-[15px] font-extrabold text-(--ink)">
        {t("evidence.title")}
      </h2>
      <ul className="flex flex-col gap-2">
        {data.evidence.map((row) => (
          <li key={row.key} className="flex items-center gap-3">
            <span className="fert-disc fert-tone-teal grid size-9 shrink-0 place-items-center rounded-full">
              <Icon name={evidenceIcon(row.key)} size={18} />
            </span>
            <div className="flex min-w-0 flex-1 flex-col text-start">
              <span className="text-[13px] font-bold text-(--ink)">
                {row.title}
              </span>
              {row.detail && (
                <span className="text-[12px] text-(--muted)">
                  {row.detail}
                </span>
              )}
            </div>
            {row.strength && (
              <span
                className={clsx(
                  "fert-disc rounded-full px-2.5 py-0.5 text-[11px] font-bold",
                  STRENGTH_CLASS[row.strength],
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

function TipsCard({ tips }: { tips: string[] }) {
  const t = useTranslations("fertility.insights");
  return (
    <section className="card flex flex-col gap-2 p-4">
      <h2 className="text-start text-[15px] font-extrabold text-(--ink)">
        {t("improve")}
      </h2>
      {tips.length > 0 && (
        <ul className="flex list-disc flex-col gap-1 ps-5 text-start text-[13px] leading-6 text-(--ink-3)">
          {tips.map((tip) => (
            <li key={tip}>{tip}</li>
          ))}
        </ul>
      )}
      <Link href={LOG_HREF} className="btn btn-primary mt-1">
        {t("logCta")}
      </Link>
    </section>
  );
}

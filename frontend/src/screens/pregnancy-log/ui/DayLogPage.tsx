"use client";

import clsx from "clsx";
import { useLocale, useTranslations } from "next-intl";
import { useEffect, useMemo, useState } from "react";

import {
  DAY_SYMPTOMS,
  moodGlyph,
  MOODS_DISPLAY_ORDER,
  type PregnancyAlertV2,
  SEVERITIES,
  usePregnancyDay,
  WATER_MAX,
  WATER_MIN,
} from "@/entities/pregnancy";
import { type Locale, Link, useRouter } from "@/shared/i18n";
import {
  type DateParts,
  formatDayMonth,
  formatNumber,
  formatWeekdayDayMonth,
  fromApiDate,
  partsToDate,
  toApiDate,
  toParts,
  today,
} from "@/shared/lib/date";
import { AppSheet } from "@/shared/sheet";
import { CalendarPicker, Icon } from "@/shared/ui";

import {
  type DayDraft,
  draftFromDay,
  EMPTY_DRAFT,
  selectedSymptoms,
  setSeverity,
  stepWater,
  toggleSymptom,
  inputFromDraft,
} from "../model/draft";
import { isConfirmedSave } from "../model/save-status";
import { useSaveDay } from "../model/use-save-day";

const CARD = "rounded-2xl border border-(--line) bg-(--surface) p-4";
const H2 = "m-0 text-[15px] font-extrabold text-(--ink)";

/** A valid, non-future `YYYY-MM-DD`, else today. */
function resolveDate(raw: string | undefined): string {
  const now = toApiDate(today());
  return raw && /^\d{4}-\d{2}-\d{2}$/.test(raw) && raw <= now ? raw : now;
}

/** «ثبت علائم» v2 — `/pregnancy/log?date=` (Log.dc.html, T-M7-12). */
export function DayLogPage({ date: rawDate }: { date?: string }) {
  const t = useTranslations("pregnancyV2");
  const locale = useLocale() as Locale;
  const router = useRouter();
  const date = resolveDate(rawDate);
  const isToday = date === toApiDate(today());
  const query = usePregnancyDay(date);
  const { submit, reset, status, alerts, pending } = useSaveDay();
  const [draft, setDraft] = useState<DayDraft>(EMPTY_DRAFT);
  const [picking, setPicking] = useState(false);

  useEffect(() => {
    if (query.data !== undefined) setDraft(draftFromDay(query.data));
  }, [query.data]);

  const edit = (patch: Partial<DayDraft>) => {
    setDraft((d) => ({ ...d, ...patch }));
    reset();
  };

  const selected = selectedSymptoms(draft.symptoms);
  const day = query.data;
  const dateLabel = formatWeekdayDayMonth(fromApiDate(date), locale);
  const subtitle =
    day?.week != null
      ? t("log.sub", { date: dateLabel, week: day.week })
      : dateLabel;
  const pickerValue = useMemo(
    () => toParts(fromApiDate(date), locale),
    [date, locale],
  );

  const pickDay = (parts: DateParts) => {
    const next = toApiDate(partsToDate(parts, locale));
    if (next > toApiDate(today())) return;
    setPicking(false);
    router.replace(`/pregnancy/log?date=${next}`, { scroll: false });
  };

  // «queued» is only on the device: the button stays a save button until the
  // outbox has actually sent the day (the status line says it is queued).
  const saved = isConfirmedSave(status);

  return (
    <div className="view preg-page">
      <div className="scroll">
        <header className="rmd-hdr">
          <Link
            href="/pregnancy"
            className="rmd-hdr-btn"
            aria-label={t("common.close")}
          >
            <Icon name="x" size={18} />
          </Link>
          <div className="rmd-hdr-text">
            <h1 className="rmd-hdr-title">
              {isToday ? t("log.title") : t("log.titleOtherDay")}
            </h1>
            <p className="rmd-hdr-sub">{subtitle}</p>
          </div>
          <button
            type="button"
            className="rmd-hdr-btn"
            aria-label={t("log.pickDay")}
            onClick={() => setPicking(true)}
          >
            <Icon name="calendar" size={18} strokeWidth={1.8} />
          </button>
        </header>

        {pending > 0 && (
          <p className="mx-4 mb-2 flex items-center gap-2 rounded-xl bg-(--data-soft) px-3 py-2 text-xs font-bold text-(--data-deep)">
            <Icon name="refresh" size={14} />
            {t("log.pending", { count: pending })}
          </p>
        )}

        <div className="flex flex-col gap-3.5 px-4 pt-1 pb-6">
          {query.isPending ? (
            <p className="m-0 py-10 text-center text-sm font-semibold text-(--ink-3)">
              {t("common.loading")}
            </p>
          ) : query.isError ? (
            <div className="flex flex-col items-center gap-3 py-10 text-center">
              <p
                role="alert"
                className="m-0 text-sm font-semibold text-(--ink-3)"
              >
                {t("common.loadError")}
              </p>
              <button
                type="button"
                className="btn btn-primary"
                onClick={() => void query.refetch()}
              >
                {t("common.retry")}
              </button>
            </div>
          ) : query.data === null ? (
            <div className="flex flex-col items-center gap-3 py-10 text-center">
              <p className="m-0 text-sm font-semibold text-(--ink-3)">
                {t("common.notActive")}
              </p>
              <Link href="/pregnancy" className="btn btn-primary no-underline">
                {t("common.back")}
              </Link>
            </div>
          ) : (
            <>
              <section className={CARD}>
                <h2 id="plog-mood" className={H2}>
                  {t("log.moodTitle")}
                </h2>
                <div
                  role="radiogroup"
                  aria-labelledby="plog-mood"
                  className="mt-3 grid grid-cols-5 gap-2"
                >
                  {MOODS_DISPLAY_ORDER.map((m) => {
                    const on = draft.mood === m;
                    return (
                      <button
                        key={m}
                        type="button"
                        role="radio"
                        aria-checked={on}
                        onClick={() => edit({ mood: on ? null : m })}
                        className={clsx(
                          "flex flex-col items-center gap-1 rounded-2xl py-2.5 text-[11.5px] font-bold",
                          on
                            ? "border-2 border-(--brand-fill) bg-(--surface-2) text-(--brand-deep)"
                            : "border border-(--line) bg-(--surface) text-(--ink-2)",
                        )}
                      >
                        <Icon name={moodGlyph(m)} size={26} strokeWidth={1.8} />
                        {t(`log.moods.${m}`)}
                      </button>
                    );
                  })}
                </div>
              </section>

              <section className={CARD}>
                <div className="flex items-center justify-between">
                  <h2 className={H2}>{t("log.symptomsTitle")}</h2>
                  {selected.length > 0 && (
                    <span className="text-xs font-bold text-(--ink-3)">
                      {t("log.symptomsCount", { count: selected.length })}
                    </span>
                  )}
                </div>
                <div className="mt-3 flex flex-wrap gap-2">
                  {DAY_SYMPTOMS.map((s) => {
                    const on = !!draft.symptoms[s];
                    return (
                      <button
                        key={s}
                        type="button"
                        aria-pressed={on}
                        onClick={() =>
                          edit({ symptoms: toggleSymptom(draft.symptoms, s) })
                        }
                        className={clsx(
                          "inline-flex h-9 items-center gap-1 rounded-full border px-3.5 text-[13px] font-bold",
                          on
                            ? "border-(--brand-fill) bg-(--brand-fill) text-(--on-accent)"
                            : "border-(--line) bg-(--surface) text-(--ink)",
                        )}
                      >
                        {on && <Icon name="check" size={14} strokeWidth={3} />}
                        {t(`log.symptoms.${s}`)}
                      </button>
                    );
                  })}
                </div>
                {selected.length > 0 && (
                  <div className="mt-3 flex flex-col gap-2 border-t border-(--line) pt-3">
                    {selected.map((s) => (
                      <div
                        key={s}
                        className="flex items-center justify-between gap-2"
                      >
                        <span className="text-[13px] font-bold text-(--ink)">
                          {t(`log.symptoms.${s}`)}
                        </span>
                        <div
                          role="radiogroup"
                          aria-label={`${t("log.severityLabel")} — ${t(`log.symptoms.${s}`)}`}
                          className="flex gap-1.5"
                        >
                          {SEVERITIES.map((lv) => {
                            const on = draft.symptoms[s] === lv;
                            return (
                              <button
                                key={lv}
                                type="button"
                                role="radio"
                                aria-checked={on}
                                onClick={() =>
                                  edit({
                                    symptoms: setSeverity(
                                      draft.symptoms,
                                      s,
                                      lv,
                                    ),
                                  })
                                }
                                className={clsx(
                                  "h-8 rounded-full border px-3 text-xs font-bold",
                                  on
                                    ? "border-(--brand-fill) bg-(--brand-fill) text-(--on-accent)"
                                    : "border-(--line) bg-(--surface) text-(--ink)",
                                )}
                              >
                                {t(`log.severities.${lv}`)}
                              </button>
                            );
                          })}
                        </div>
                      </div>
                    ))}
                  </div>
                )}
              </section>

              <section className={clsx(CARD, "flex items-center gap-3")}>
                <span className="flex size-11 shrink-0 items-center justify-center rounded-xl bg-(--data-soft) text-(--data-deep)">
                  <Icon name="drop" size={22} strokeWidth={1.8} />
                </span>
                <div className="min-w-0 flex-1">
                  <b className="block text-[15px] text-(--ink)">
                    {t("log.water")}
                  </b>
                  <span
                    className="text-xs font-semibold text-(--ink-3)"
                    aria-live="polite"
                  >
                    {t("log.waterCount", { count: draft.water })}
                  </span>
                </div>
                <button
                  type="button"
                  className="flex size-10 items-center justify-center rounded-full border border-(--line) bg-(--surface) text-(--ink) disabled:opacity-40"
                  aria-label={t("log.waterLess")}
                  disabled={draft.water <= WATER_MIN}
                  onClick={() => edit({ water: stepWater(draft.water, -1) })}
                >
                  <span aria-hidden className="text-lg leading-none font-black">
                    −
                  </span>
                </button>
                <button
                  type="button"
                  className="flex size-10 items-center justify-center rounded-full bg-(--brand-fill) text-(--on-accent) disabled:opacity-40"
                  aria-label={t("log.waterMore")}
                  disabled={draft.water >= WATER_MAX}
                  onClick={() => edit({ water: stepWater(draft.water, 1) })}
                >
                  <Icon name="plus" size={18} strokeWidth={2.4} />
                </button>
              </section>

              <section className={clsx(CARD, "flex items-center gap-3")}>
                <span className="flex size-11 shrink-0 items-center justify-center rounded-xl bg-(--surface-2) text-(--brand)">
                  <Icon name="scale" size={22} strokeWidth={1.8} />
                </span>
                <div className="min-w-0 flex-1">
                  <label
                    htmlFor="plog-weight"
                    className="block text-[15px] font-bold text-(--ink)"
                  >
                    {t("log.weight")}
                  </label>
                  <span className="text-xs font-semibold text-(--ink-3)">
                    {day?.lastWeight
                      ? t("log.weightLast", {
                          value: day.lastWeight.value,
                          date: formatDayMonth(
                            fromApiDate(day.lastWeight.date),
                            locale,
                          ),
                        })
                      : t("log.weightNone")}
                  </span>
                </div>
                <div className="flex items-center gap-1.5 rounded-xl border border-(--field-border) bg-(--surface) px-2.5">
                  <input
                    id="plog-weight"
                    inputMode="decimal"
                    autoComplete="off"
                    value={formatNumber(draft.weight, locale)}
                    onChange={(e) => edit({ weight: e.target.value })}
                    className="h-10 w-16 bg-transparent text-center text-[15px] font-bold text-(--ink) outline-none"
                  />
                  <span className="text-xs font-semibold text-(--ink-3)">
                    {t("log.kg")}
                  </span>
                </div>
              </section>

              <section className={CARD}>
                <label htmlFor="plog-note" className={clsx(H2, "block")}>
                  {t("log.note")}
                </label>
                <textarea
                  id="plog-note"
                  rows={3}
                  value={draft.visitNote}
                  placeholder={t("log.notePlaceholder")}
                  onChange={(e) => edit({ visitNote: e.target.value })}
                  className="mt-2 w-full resize-none rounded-xl border border-(--field-border) bg-(--surface) p-3 text-sm text-(--ink)"
                />
              </section>

              {draft.symptoms.spotting && (
                <section
                  className="pg2-warn flex items-start gap-3 rounded-2xl p-3.5"
                  role="note"
                >
                  <Icon name="warning" size={20} className="pg2-warn-icon" />
                  <p className="m-0 text-[12.5px] leading-[1.9]">
                    <b>{t("log.spottingTitle")}</b> {t("log.spottingBody")}
                  </p>
                </section>
              )}

              {alerts.length > 0 && <RaisedAlerts alerts={alerts} />}

              <nav
                aria-label={t("log.otherLogs")}
                className="flex flex-col gap-2"
              >
                <h2 className="m-0 px-1 text-xs font-bold text-(--ink-3)">
                  {t("log.otherLogs")}
                </h2>
                <div className="flex gap-2">
                  <Link
                    href="/pregnancy/log?tab=weekly"
                    className={clsx(
                      CARD,
                      "flex-1 py-3 text-center text-[13px] font-bold text-(--ink) no-underline",
                    )}
                  >
                    {t("log.weeklyCheckup")}
                  </Link>
                  <Link
                    href="/pregnancy/log?tab=movement"
                    className={clsx(
                      CARD,
                      "flex-1 py-3 text-center text-[13px] font-bold text-(--ink) no-underline",
                    )}
                  >
                    {t("log.movement")}
                  </Link>
                </div>
              </nav>
            </>
          )}
        </div>

        {day && (
          <div className="sticky bottom-0 z-10 flex flex-col gap-1.5 border-t border-(--line) bg-(--page) px-4 pt-3 pb-[max(12px,env(safe-area-inset-bottom))]">
            <button
              type="button"
              className={clsx(
                "flex h-12 w-full items-center justify-center gap-2 rounded-2xl text-[15px] font-extrabold text-(--on-accent) disabled:opacity-70",
                saved ? "bg-(--success)" : "bg-(image:--gradient-brand)",
              )}
              disabled={status === "saving"}
              onClick={() => void submit(date, inputFromDraft(draft))}
            >
              {saved && <Icon name="check" size={18} strokeWidth={2.6} />}
              {status === "saving"
                ? t("log.saving")
                : saved
                  ? t("log.saved")
                  : t("log.save")}
            </button>
            <span
              role="status"
              className="text-center text-[11.5px] font-semibold text-(--ink-3)"
            >
              {status === "error" ? (
                <span className="text-(--danger-deep)">
                  {t("common.saveError")}
                </span>
              ) : status === "queued" ? (
                <span className="inline-flex items-center gap-1">
                  <Icon name="clock" size={13} />
                  {t("log.queued")}
                </span>
              ) : (
                `${t("log.offlineNote")} ${t("log.editNote")}`
              )}
            </span>
          </div>
        )}
      </div>

      <AppSheet
        open={picking}
        onClose={() => setPicking(false)}
        size="half"
        title={t("log.pickDay")}
      >
        <CalendarPicker value={pickerValue} onSelect={pickDay} />
      </AppSheet>
    </div>
  );
}

function RaisedAlerts({ alerts }: { alerts: PregnancyAlertV2[] }) {
  const t = useTranslations("pregnancyV2");
  return (
    <section className={clsx(CARD, "flex flex-col gap-2")} aria-live="polite">
      <h2 className={H2}>{t("log.alertsRaised", { count: alerts.length })}</h2>
      <ul className="m-0 flex list-none flex-col gap-1.5 p-0">
        {alerts.map((a) => (
          <li
            key={a.id}
            className="flex items-start gap-2 text-[13px] text-(--ink)"
          >
            <span className="mt-1 rounded-full bg-(--surface-2) px-2 py-0.5 text-[11px] font-bold text-(--brand-deep)">
              {t(`alerts.levelsShort.${a.level}`)}
            </span>
            <span className="min-w-0 flex-1 font-semibold">{a.title}</span>
          </li>
        ))}
      </ul>
      <Link
        href="/pregnancy/alerts"
        className="text-[13px] font-extrabold text-(--brand) no-underline"
      >
        {t("log.seeAlerts")}
      </Link>
    </section>
  );
}

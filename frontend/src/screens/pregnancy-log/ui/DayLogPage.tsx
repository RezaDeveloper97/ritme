"use client";

import clsx from "clsx";
import { useLocale, useTranslations } from "next-intl";
import { useEffect, useMemo, useState } from "react";

import {
  type AlertLevelV2,
  DAY_SYMPTOMS,
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
  formatWeekdayDayMonth,
  fromApiDate,
  partsToDate,
  toApiDate,
  toParts,
  today,
} from "@/shared/lib/date";
import { AppSheet } from "@/shared/sheet";
import {
  CalendarPicker,
  Card,
  EmptyState,
  HeaderButton,
  Icon,
  IconCircle,
  PillChip,
  PrimaryButton,
  ScreenHeader,
  SecondaryButton,
  Skeleton,
  SkeletonGroup,
  SkyLayer,
  type Tone,
} from "@/shared/ui";

import {
  type DayDraft,
  displayWeight,
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

/** The Log artboard's glass row; more than eight still counts through −/+. */
const WATER_GLASSES = 8;

const LEVEL_TONE: Record<AlertLevelV2, Tone> = {
  info: "data",
  suggestion: "brand",
  follow_up: "warm",
  urgent: "danger",
};

/** A valid, non-future `YYYY-MM-DD`, else today. */
function resolveDate(raw: string | undefined): string {
  const now = toApiDate(today());
  return raw && /^\d{4}-\d{2}-\d{2}$/.test(raw) && raw <= now ? raw : now;
}

/** «ثبت علائم» v2 — `/pregnancy/log?date=` (PregFull_Log). */
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
    router.replace(`/pregnancy/log?tab=day&date=${next}`, { scroll: false });
  };

  // «queued» is only on the device: the button stays a save button until the
  // outbox has actually sent the day (the status line says it is queued).
  const saved = isConfirmedSave(status);

  const header = (
    <ScreenHeader
      title={isToday ? t("log.title") : t("log.titleOtherDay")}
      subtitle={subtitle}
      onBack={() => router.push("/pregnancy")}
      backLabel={t("common.back")}
      action={<HeaderButton icon="calendar" label={t("log.pickDay")} onClick={() => setPicking(true)} />}
    />
  );

  let body: React.ReactNode;
  if (query.isPending) {
    body = (
      <SkeletonGroup label={t("common.loading")} className="pgn-skel">
        <Skeleton shape="card" />
        <Skeleton shape="card" className="pgn-skel-hero" />
        <Skeleton shape="block" />
        <Skeleton shape="block" />
      </SkeletonGroup>
    );
  } else if (query.isError) {
    body = (
      <Card className="pgn-state" role="alert">
        <span className="pgn-state-disc" aria-hidden>
          <Icon name="warning" size={24} />
        </span>
        <p className="pgn-state-text">{t("common.loadError")}</p>
        <SecondaryButton icon="refresh" block={false} onClick={() => void query.refetch()}>
          {t("common.retry")}
        </SecondaryButton>
      </Card>
    );
  } else if (query.data === null) {
    body = (
      <Card>
        <EmptyState
          icon="heart"
          title={t("common.notActive")}
          action={
            <Link href="/pregnancy/setup" className="nb-btn is-primary is-block">
              {t("today.setupCta")}
            </Link>
          }
        />
      </Card>
    );
  } else {
    const chipSymptoms = DAY_SYMPTOMS.filter((s) => s !== "spotting");
    const spotting = !!draft.symptoms.spotting;
    body = (
      <>
        <Card as="section" className="pgn-sect" aria-labelledby="plog-mood">
          <h2 id="plog-mood" className="pgn-sect-title">
            {t("log.moodTitle")}
          </h2>
          <div role="group" aria-labelledby="plog-mood" className="nb-chips pgn-chips">
            {MOODS_DISPLAY_ORDER.map((m) => (
              <PillChip
                key={m}
                mode="multi"
                tone="brand"
                className="pgn-chip"
                pressed={draft.mood === m}
                onPressedChange={(on) => edit({ mood: on ? m : null })}
              >
                {t(`log.moods.${m}`)}
              </PillChip>
            ))}
          </div>
        </Card>

        <Card as="section" className="pgn-sect" aria-labelledby="plog-symptoms">
          <div className="pgn-sect-head">
            <h2 id="plog-symptoms" className="pgn-sect-title">
              {t("log.symptomsTitle")}
            </h2>
            {selected.length > 0 && (
              <span className="pgn-meta">{t("log.symptomsCount", { count: selected.length })}</span>
            )}
          </div>
          <div role="group" aria-labelledby="plog-symptoms" className="nb-chips pgn-chips">
            {chipSymptoms.map((s) => (
              <PillChip
                key={s}
                mode="multi"
                tone="warm"
                className="pgn-chip"
                pressed={!!draft.symptoms[s]}
                onPressedChange={() => edit({ symptoms: toggleSymptom(draft.symptoms, s) })}
              >
                {t(`log.symptoms.${s}`)}
              </PillChip>
            ))}
          </div>
          {selected.length > 0 && (
            <div className="pgn-severity">
              {selected.map((s) => (
                <div key={s} className="pgn-severity-row">
                  <span className="pgn-row-title">{t(`log.symptoms.${s}`)}</span>
                  <div
                    role="radiogroup"
                    aria-label={`${t("log.severityLabel")} — ${t(`log.symptoms.${s}`)}`}
                    className="pgn-severity-opts"
                  >
                    {SEVERITIES.map((lv) => (
                      <button
                        key={lv}
                        type="button"
                        role="radio"
                        aria-checked={draft.symptoms[s] === lv}
                        onClick={() => edit({ symptoms: setSeverity(draft.symptoms, s, lv) })}
                        className="pgn-severity-opt"
                      >
                        {t(`log.severities.${lv}`)}
                      </button>
                    ))}
                  </div>
                </div>
              ))}
            </div>
          )}
        </Card>

        <Card as="section" className="pgn-sect" aria-labelledby="plog-water">
          <h2 id="plog-water" className="pgn-sect-title">
            {t("log.water")}
          </h2>
          <div className="pgn-glasses" role="group" aria-labelledby="plog-water">
            {Array.from({ length: WATER_GLASSES }, (_, i) => i + 1).map((n) => (
              <button
                key={n}
                type="button"
                aria-pressed={draft.water >= n}
                aria-label={t("log.waterGlass", { count: n })}
                className="pgn-glass"
                onClick={() => edit({ water: draft.water === n ? n - 1 : n })}
              />
            ))}
          </div>
          <div className="pgn-water-foot">
            <span className="pgn-caption" aria-live="polite">
              {t("log.waterCount", { count: draft.water })}
            </span>
            <button
              type="button"
              className="pgn-step"
              aria-label={t("log.waterLess")}
              disabled={draft.water <= WATER_MIN}
              onClick={() => edit({ water: stepWater(draft.water, -1) })}
            >
              <Icon name="minus" size={18} strokeWidth={2.2} />
            </button>
            <button
              type="button"
              className="pgn-step"
              aria-label={t("log.waterMore")}
              disabled={draft.water >= WATER_MAX}
              onClick={() => edit({ water: stepWater(draft.water, 1) })}
            >
              <Icon name="plus" size={18} strokeWidth={2.2} />
            </button>
          </div>
        </Card>

        <Card as="section" className="pgn-sect">
          <label htmlFor="plog-weight" className="pgn-sect-title">
            {t("log.weight")}
          </label>
          <div className="pgn-weight">
            <input
              id="plog-weight"
              inputMode="decimal"
              autoComplete="off"
              placeholder="—"
              value={displayWeight(draft.weight, locale)}
              onChange={(e) => edit({ weight: e.target.value })}
              className="pgn-weight-input"
              aria-describedby="plog-weight-last"
            />
            <span id="plog-weight-last" className="pgn-caption">
              {t("log.kg")}
              {t("common.separator")}
              {day?.lastWeight
                ? t("log.weightLast", {
                    value: day.lastWeight.value,
                    date: formatDayMonth(fromApiDate(day.lastWeight.date), locale),
                  })
                : t("log.weightNone")}
            </span>
          </div>
        </Card>

        <button
          type="button"
          aria-pressed={spotting}
          className="pgn-spotting"
          onClick={() => edit({ symptoms: toggleSymptom(draft.symptoms, "spotting") })}
        >
          <Icon name="drop" size={20} strokeWidth={1.8} className="pgn-spotting-icon" />
          <span className="pgn-row-text">
            <b className="pgn-row-title">{t("log.spottingCard.title")}</b>
            <span className="pgn-row-desc">{t("log.spottingCard.body")}</span>
          </span>
          <Icon name={spotting ? "check" : "plus"} size={16} strokeWidth={2.4} className="pgn-spotting-icon" />
        </button>

        {spotting && (
          <section className="pgn-warn" role="note">
            <Icon name="warning" size={20} className="pgn-warn-icon" />
            <p className="pgn-warn-text">
              <b>{t("log.spottingTitle")}</b> {t("log.spottingBody")}
            </p>
          </section>
        )}

        <Card as="section" className="pgn-sect">
          <label htmlFor="plog-note" className="pgn-sect-title">
            {t("log.note")}
          </label>
          <textarea
            id="plog-note"
            rows={3}
            value={draft.visitNote}
            placeholder={t("log.notePlaceholder")}
            onChange={(e) => edit({ visitNote: e.target.value })}
            className="pgn-textarea"
          />
        </Card>

        {alerts.length > 0 && <RaisedAlerts alerts={alerts} />}

        <nav aria-labelledby="plog-other" className="pgn-other">
          <h2 id="plog-other" className="pgn-other-title">
            {t("log.otherLogs")}
          </h2>
          <div className="pgn-other-row">
            <Link href="/pregnancy/log?tab=weekly" className="pgn-tile is-row">
              <IconCircle icon="doctor" tone="data" size="sm" />
              <b className="pgn-tile-label">{t("log.weeklyCheckup")}</b>
            </Link>
            <Link href="/pregnancy/log?tab=movement" className="pgn-tile is-row">
              <IconCircle icon="heartLine" tone="bloom" size="sm" />
              <b className="pgn-tile-label">{t("log.movement")}</b>
            </Link>
          </div>
        </nav>
      </>
    );
  }

  return (
    <div className="view preg-page pgn-screen">
      <SkyLayer />
      <div className="scroll">
        {header}
        {pending > 0 && (
          <p className="pgn-pending">
            <Icon name="refresh" size={14} />
            {t("log.pending", { count: pending })}
          </p>
        )}
        <div className="pgn-body is-form">{body}</div>

        {day && (
          <div className="pgn-footer">
            <PrimaryButton
              className={clsx(saved && "pgn-saved")}
              icon={saved ? "check" : undefined}
              loading={status === "saving"}
              onClick={() => void submit(date, inputFromDraft(draft))}
            >
              {status === "saving" ? t("log.saving") : saved ? t("log.saved") : t("log.save")}
            </PrimaryButton>
            <span role="status" className="pgn-footer-note">
              {status === "error" ? (
                <span className="pgn-error-text">{t("common.saveError")}</span>
              ) : status === "queued" ? (
                <span className="pgn-inline">
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

      <AppSheet open={picking} onClose={() => setPicking(false)} size="half" title={t("log.pickDay")}>
        <CalendarPicker value={pickerValue} onSelect={pickDay} />
      </AppSheet>
    </div>
  );
}

function RaisedAlerts({ alerts }: { alerts: PregnancyAlertV2[] }) {
  const t = useTranslations("pregnancyV2");
  return (
    <Card as="section" className="pgn-sect" aria-live="polite" aria-labelledby="plog-raised">
      <h2 id="plog-raised" className="pgn-sect-title">
        {t("log.alertsRaised", { count: alerts.length })}
      </h2>
      <ul className="pgn-rows">
        {alerts.map((a) => (
          <li key={a.id} className="pgn-row">
            {/* The level pill of the Alerts screen and its legend (QA 2026-09-29-c L6). */}
            <span className={clsx("pgn-opill", `nb-tone-${LEVEL_TONE[a.level]}`)}>
              {t(`alerts.levelsShort.${a.level}`)}
            </span>
            <span className="pgn-row-title">{a.title}</span>
          </li>
        ))}
      </ul>
      <Link href="/pregnancy/alerts" className="pgn-link">
        {t("log.seeAlerts")}
      </Link>
    </Card>
  );
}

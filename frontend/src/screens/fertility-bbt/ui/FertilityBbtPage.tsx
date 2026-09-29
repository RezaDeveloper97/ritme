"use client";

import clsx from "clsx";
import { useLocale, useTranslations } from "next-intl";
import { type ReactNode, useState } from "react";

import {
  BBT_RANGES,
  type BbtRange,
  BbtNumber,
  formatBbt,
  useFertilityBbt,
} from "@/entities/fertility";
import { useReminders } from "@/entities/reminder";
import {
  useCreateReminder,
  useDeleteReminder,
} from "@/features/manage-reminders";
import { BbtChart } from "@/widgets/bbt-chart";
import { type Locale, Link, useDirection, useRouter } from "@/shared/i18n";
import { formatNumber, toApiDate, today } from "@/shared/lib/date";
import { AppSheet } from "@/shared/sheet";
import { Icon, type IconName } from "@/shared/ui";

import {
  findBbtReminder,
  hasEnoughReadings,
  parseRange,
  REMINDER_TIME,
  todayReading,
} from "../model/view";

const BACK_HREF = "/home";
const INSIGHTS_HREF = "/fertility/insights";
const LOG_HREF = "/fertility/log?focus=bbt";

/**
 * «دمای پایه» — BBT chart screen (`v19_TTC_BBT` / `nb2_TTC_BBT`), route
 * `/fertility/bbt?range=1|3|6`. Privacy (§11): readings are shown, never logged.
 */
export function FertilityBbtPage({ range: rawRange }: { range?: string }) {
  const t = useTranslations("fertility");
  const locale = useLocale() as Locale;
  const dir = useDirection();
  const router = useRouter();
  const range = parseRange(rawRange);
  const query = useFertilityBbt(range);
  const [explainer, setExplainer] = useState(false);
  const todayStr = toApiDate(today());

  const data = query.data;
  const current = data?.cycles[0];
  const day = data?.stats.cycleDaysSoFar ?? 0;

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
  } else if (!hasEnoughReadings(current)) {
    body = (
      <section className="fert-card flex flex-col items-center gap-3 px-4 py-8 text-center">
        <span className="fert-disc fert-tone-teal grid size-14 place-items-center rounded-full">
          <Icon name="thermo" size={26} />
        </span>
        <h2 className="text-[16px] font-extrabold text-(--ink)">
          {t("bbt.empty.title")}
        </h2>
        <p className="text-[13px] text-(--muted)">{t("bbt.empty.body")}</p>
        <Link href={LOG_HREF} className="btn btn-primary mt-1">
          {t("bbt.empty.cta")}
        </Link>
      </section>
    );
  } else {
    const reading = todayReading(current, todayStr);
    const stats = data.stats;
    body = (
      <>
        <section className="fert-card flex flex-col gap-3 px-3">
          <div className="flex items-start justify-between gap-3 px-1">
            <div className="flex flex-col text-start">
              <span className="text-[12px] font-bold text-(--ink-3)">
                {reading ? t("bbt.todayMorning") : t("bbt.noReadingToday")}
              </span>
              {reading && (
                <span dir="ltr" className="flex items-baseline gap-1">
                  <BbtNumber
                    text={formatBbt(reading.value, locale)}
                    className="font-['Lalezar'] text-[44px] leading-none text-(--ink)"
                  />
                  <span className="text-[16px] font-bold text-(--ink-3)">
                    {t("celsius")}
                  </span>
                </span>
              )}
            </div>
            {current?.phase && (
              <span className="fert-pill fert-tone-teal">
                {t(`bbt.phases.${current.phase}`)}
              </span>
            )}
          </div>
          <BbtChart cycles={data.cycles} today={todayStr} />
        </section>

        <div className="grid grid-cols-2 gap-2.5">
          <StatCard
            icon="thermo"
            tone="teal"
            href={INSIGHTS_HREF}
            label={t("bbt.stats.preOvulationAvg")}
            value={
              stats.preOvulationAvg === null
                ? "—"
                : t("degrees", { value: formatBbt(stats.preOvulationAvg, locale) })
            }
            hint={t("bbt.preOvulationHint")}
          />
          <StatCard
            icon="check"
            tone={stats.gaps === 0 ? "green" : "amber"}
            href={INSIGHTS_HREF}
            label={t("bbt.stats.loggedDays")}
            value={t("bbt.stats.loggedOfTotal", {
              logged: formatNumber(stats.loggedDays, locale),
              total: formatNumber(stats.cycleDaysSoFar, locale),
            })}
            hint={
              stats.gaps === 0
                ? t("bbt.stats.noGaps")
                : t("bbt.stats.gaps", {
                    count: formatNumber(stats.gaps, locale),
                  })
            }
          />
        </div>

        {data.tip && (
          <section className="fert-card flex items-start gap-3.5 p-4">
            <span className="grid size-11 shrink-0 place-items-center rounded-full bg-(--fert-teal-soft) text-(--fert-teal)">
              <Icon name="sparkle" size={20} fill="currentColor" strokeWidth={0} />
            </span>
            <div className="flex min-w-0 flex-col gap-1 text-start">
              <h2 className="text-[14.5px] font-extrabold text-(--ink)">
                {data.tip.title ?? t("bbt.tip.title")}
              </h2>
              <p className="text-[13px] leading-[1.9] text-(--ink-3)">
                {data.tip.body}
              </p>
              <Link
                href={INSIGHTS_HREF}
                className="mt-1 inline-flex items-center gap-1 self-start text-[13px] font-extrabold text-(--brand)"
              >
                {t("bbt.openInsights")}
                <Icon
                  name={dir === "rtl" ? "chevronLeft" : "chevronRight"}
                  size={15}
                  strokeWidth={2.2}
                />
              </Link>
            </div>
          </section>
        )}

        <ReminderButton />
      </>
    );
  }

  return (
    <div className="view fert-page">
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
            <h1 className="rmd-hdr-title">{t("bbt.title")}</h1>
            {data && (
              <p className="rmd-hdr-sub">
                {t("bbt.subtitle", { day: formatNumber(day, locale) })}
              </p>
            )}
          </div>
          <button
            type="button"
            className="rmd-hdr-btn"
            aria-label={t("bbt.help")}
            onClick={() => setExplainer(true)}
          >
            <Icon name="info" size={20} strokeWidth={1.8} />
          </button>
        </header>

        <div className="flex flex-col gap-3.5 px-4 pt-1 pb-32">
          <div role="tablist" aria-label={t("bbt.tabsLabel")} className="rmd-tabs">
            {BBT_RANGES.map((r: BbtRange) => (
              <button
                key={r}
                type="button"
                role="tab"
                aria-selected={r === range}
                onClick={() =>
                  router.replace(
                    r === 1 ? "/fertility/bbt" : `/fertility/bbt?range=${r}`,
                  )
                }
                className={clsx("rmd-tab", r === range && "on")}
              >
                {t(`bbt.ranges.${r}`)}
              </button>
            ))}
          </div>
          {body}
        </div>
      </div>

      <AppSheet
        open={explainer}
        onClose={() => setExplainer(false)}
        size="full"
        title={t("bbt.explainer.title")}
      >
        <div className="flex flex-col gap-3 text-start text-[14px] leading-7 text-(--ink-3)">
          <p>{t("bbt.explainer.p1")}</p>
          <p>{t("bbt.explainer.p2")}</p>
          <p>{t("bbt.explainer.p3")}</p>
        </div>
      </AppSheet>
    </div>
  );
}

const STAT_TONE = {
  teal: "fert-tone-teal",
  green: "fert-tone-green",
  amber: "fert-tone-amber",
} as const;

/** `v19_TTC_BBT` stat card: tone disc, label, Lalezar value, a toned hint. */
function StatCard({
  icon,
  tone,
  href,
  label,
  value,
  hint,
}: {
  icon: IconName;
  tone: keyof typeof STAT_TONE;
  href: string;
  label: string;
  value: string;
  hint: string;
}) {
  return (
    <Link
      href={href}
      className={clsx("fert-card flex flex-col gap-1.5 p-4 text-start", STAT_TONE[tone])}
    >
      <span className="fert-disc size-9.5 rounded-full" aria-hidden>
        <Icon name={icon} size={19} />
      </span>
      <span className="text-[12px] font-bold text-(--ink-3)">{label}</span>
      <BbtNumber
        text={value}
        className="font-['Lalezar'] text-[26px] leading-[1.1] text-(--ink)"
      />
      <span className="text-[11px] font-bold text-(--fert-ink)">{hint}</span>
    </Link>
  );
}

/** Creates the daily 07:00 custom reminder, or turns it off when it exists. */
function ReminderButton() {
  const t = useTranslations("fertility.bbt");
  const reminders = useReminders("custom");
  const create = useCreateReminder();
  const remove = useDeleteReminder();
  const title = t("reminderTitle");
  const existing = findBbtReminder(reminders.data, title);
  const busy = create.isPending || remove.isPending || reminders.isPending;
  const failed = create.isError || remove.isError;

  const toggle = () => {
    if (existing) remove.mutate(existing.id);
    else
      create.mutate({
        type: "custom",
        title,
        recurrence: "daily",
        recurrenceTime: REMINDER_TIME,
      });
  };

  return (
    <div className="flex flex-col gap-1.5">
      <button
        type="button"
        aria-pressed={existing !== null}
        disabled={busy}
        onClick={toggle}
        className="flex h-13 items-center justify-center gap-2 rounded-full border border-(--line) bg-(--surface) text-[14.5px] font-extrabold text-(--ink) disabled:opacity-60 focus-visible:shadow-(--ring) focus-visible:outline-none"
      >
        <Icon name={existing ? "bellRing" : "moon"} size={18} />
        {busy && !reminders.isPending
          ? t("reminderSaving")
          : existing
            ? t("reminderOff")
            : t("remindTomorrow")}
      </button>
      {failed && (
        <p role="alert" className="text-center text-[12px] text-(--danger)">
          {t("reminderError")}
        </p>
      )}
    </div>
  );
}

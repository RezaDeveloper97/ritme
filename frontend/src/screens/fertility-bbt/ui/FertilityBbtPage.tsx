"use client";

import clsx from "clsx";
import { useLocale, useTranslations } from "next-intl";
import { type ReactNode, useState } from "react";

import {
  BBT_RANGES,
  type BbtRange,
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
import { Icon } from "@/shared/ui";

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
      <section className="card flex flex-col items-center gap-3 px-4 py-8 text-center">
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
        <section className="card flex flex-col gap-4 p-4">
          <div className="flex items-end justify-between gap-3">
            <div className="flex flex-col text-start">
              <span className="text-[12px] text-(--muted)">
                {reading ? t("bbt.todayMorning") : t("bbt.noReadingToday")}
              </span>
              {reading && (
                <span
                  dir="ltr"
                  className="font-['Lalezar'] text-[44px] leading-none text-(--ink)"
                >
                  {formatBbt(reading.value, locale)}
                  <span className="ms-1 text-[18px] text-(--muted)">
                    {t("celsius")}
                  </span>
                </span>
              )}
            </div>
            {current?.phase && (
              <span className="fert-disc fert-tone-teal rounded-full px-3 py-1 text-[12px] font-bold">
                {t(`bbt.phases.${current.phase}`)}
              </span>
            )}
          </div>
          <BbtChart cycles={data.cycles} today={todayStr} />
        </section>

        <div className="grid grid-cols-2 gap-3">
          <StatCard
            label={t("bbt.stats.preOvulationAvg")}
            value={
              stats.preOvulationAvg === null
                ? "—"
                : `${formatBbt(stats.preOvulationAvg, locale)}${t("celsius")}`
            }
            hint={t("bbt.preOvulationHint")}
          />
          <StatCard
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
          <section className="card flex items-start gap-3 p-4">
            <span className="grid size-10 shrink-0 place-items-center rounded-full bg-(--fert-teal-soft) text-(--fert-teal)">
              <Icon name="info" size={20} />
            </span>
            <div className="flex min-w-0 flex-col gap-1 text-start">
              <h2 className="text-[14px] font-bold text-(--ink)">
                {data.tip.title ?? t("bbt.tip.title")}
              </h2>
              <p className="text-[13px] leading-6 text-(--ink-3)">
                {data.tip.body}
              </p>
            </div>
          </section>
        )}

        <ReminderButton />
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
              {t("bbt.title")}
            </h1>
            {data && (
              <p className="text-[13px] text-(--muted)">
                {t("bbt.subtitle", { day: formatNumber(day, locale) })}
              </p>
            )}
          </div>
          <Link
            href={INSIGHTS_HREF}
            className="text-[13px] font-bold text-(--brand)"
          >
            {t("bbt.openInsights")}
          </Link>
          <button
            type="button"
            className="iconbtn"
            aria-label={t("bbt.help")}
            onClick={() => setExplainer(true)}
          >
            <Icon name="info" size={20} />
          </button>
        </header>

        <div className="flex flex-col gap-3 px-4 pb-32">
          <div
            role="tablist"
            aria-label={t("bbt.tabsLabel")}
            className="grid grid-cols-3 gap-1 rounded-full bg-(--surface) p-1"
          >
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
                className={clsx(
                  "rounded-full py-2 text-[13px] font-bold transition-colors",
                  r === range
                    ? "bg-(--fert-chip-on-bg) text-(--brand)"
                    : "text-(--ink-3)",
                )}
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

function StatCard({
  label,
  value,
  hint,
}: {
  label: string;
  value: string;
  hint: string;
}) {
  return (
    <div className="card flex flex-col gap-1 text-start p-4">
      <span className="text-[12px] text-(--muted)">{label}</span>
      <span
        dir="ltr"
        className="text-start text-[20px] font-extrabold text-(--ink) rtl:text-end"
      >
        {value}
      </span>
      <span className="text-[11px] text-(--muted)">{hint}</span>
    </div>
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
        className="flex items-center justify-center gap-2 rounded-full border border-(--fert-chip-on-line) bg-(--surface) py-3 text-[14px] font-bold text-(--brand) disabled:opacity-60"
      >
        <Icon name={existing ? "bellRing" : "bell"} size={18} />
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

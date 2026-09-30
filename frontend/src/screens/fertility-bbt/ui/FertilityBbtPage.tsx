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
import {
  Card,
  EmptyState,
  HeaderButton,
  Icon,
  type IconName,
  ScreenHeader,
  SecondaryButton,
  type SegmentedTab,
  SegmentedTabs,
  Skeleton,
  SkeletonGroup,
  SkyLayer,
} from "@/shared/ui";

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
  const rangeTabs: SegmentedTab<`${BbtRange}`>[] = BBT_RANGES.map((r) => ({
    value: `${r}` as `${BbtRange}`,
    label: t(`bbt.ranges.${r}`),
  }));

  const data = query.data;
  const current = data?.cycles[0];
  const day = data?.stats.cycleDaysSoFar ?? 0;

  let body: ReactNode;
  if (query.isPending) {
    body = (
      <SkeletonGroup label={t("loading")}>
        <Skeleton shape="card" className="ttc-skel-chart" />
        <div className="ttc-grid-2">
          <Skeleton shape="card" />
          <Skeleton shape="card" />
        </div>
        <Skeleton shape="block" />
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
  } else if (!hasEnoughReadings(current)) {
    body = (
      <Card>
        <EmptyState
          icon="thermo"
          className="ttc-empty"
          title={t("bbt.empty.title")}
          body={t("bbt.empty.body")}
          action={
            <Link href={LOG_HREF} className="nb-btn is-primary is-block">
              {t("bbt.empty.cta")}
            </Link>
          }
        />
      </Card>
    );
  } else {
    const reading = todayReading(current, todayStr);
    const stats = data.stats;
    body = (
      <>
        <Card as="section" className="ttc-chart-card" aria-label={t("bbt.title")}>
          <div className="ttc-chart-head">
            <div className="ttc-chart-now">
              <span className="ttc-chart-over">
                {reading ? t("bbt.todayMorning") : t("bbt.noReadingToday")}
              </span>
              {reading && (
                <span dir="ltr" className="ttc-chart-value">
                  <BbtNumber
                    text={formatBbt(reading.value, locale)}
                    className="ttc-chart-num"
                  />
                  <span className="ttc-chart-unit">
                    {t("celsius")}
                  </span>
                </span>
              )}
            </div>
            {current?.phase && (
              <span className="fert-pill fert-tone-teal ttc-phase-pill">
                {t(`bbt.phases.${current.phase}`)}
              </span>
            )}
          </div>
          <BbtChart cycles={data.cycles} today={todayStr} />
        </Card>

        <div className="ttc-grid-2">
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
          <Card as="section" className="ttc-tip-card">
            <span className="ttc-tip-card-disc" aria-hidden>
              <Icon name="sparkle" size={20} fill="currentColor" strokeWidth={0} />
            </span>
            <div className="ttc-tip-card-text">
              <h2 className="ttc-tip-card-title">
                {data.tip.title ?? t("bbt.tip.title")}
              </h2>
              <p className="ttc-tip-card-body">{data.tip.body}</p>
              <Link href={INSIGHTS_HREF} className="ttc-link">
                {t("bbt.openInsights")}
                <Icon
                  name={dir === "rtl" ? "chevronLeft" : "chevronRight"}
                  size={15}
                  strokeWidth={2.2}
                />
              </Link>
            </div>
          </Card>
        )}

        <ReminderButton />
      </>
    );
  }

  return (
    <div className="view fert-page">
      <div className="scroll ttc-screen">
        <SkyLayer />
        <ScreenHeader
          title={t("bbt.title")}
          subtitle={
            data ? t("bbt.subtitle", { day: formatNumber(day, locale) }) : undefined
          }
          onBack={() => router.push(BACK_HREF)}
          backLabel={t("back")}
          action={
            <HeaderButton
              icon="info"
              label={t("bbt.help")}
              onClick={() => setExplainer(true)}
            />
          }
        />

        <div className="ttc-body">
          <SegmentedTabs
            label={t("bbt.tabsLabel")}
            value={String(range) as `${BbtRange}`}
            tabs={rangeTabs}
            onChange={(v) =>
              router.replace(v === "1" ? "/fertility/bbt" : `/fertility/bbt?range=${v}`)
            }
          />
          {body}
        </div>
      </div>

      <AppSheet
        open={explainer}
        onClose={() => setExplainer(false)}
        size="full"
        title={t("bbt.explainer.title")}
      >
        <div className="ttc-sheet-copy">
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
      className={clsx("nb-card ttc-stat", STAT_TONE[tone])}
    >
      <span className="fert-disc ttc-stat-disc" aria-hidden>
        <Icon name={icon} size={19} />
      </span>
      <span className="ttc-stat-label">{label}</span>
      <BbtNumber text={value} className="ttc-stat-num" />
      <span className="ttc-stat-hint">{hint}</span>
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
    <div className="ttc-reminder">
      <button
        type="button"
        aria-pressed={existing !== null}
        disabled={busy}
        onClick={toggle}
        className="nb-btn is-block ttc-reminder-btn"
      >
        <Icon name={existing ? "bellRing" : "moon"} size={18} />
        {busy && !reminders.isPending
          ? t("reminderSaving")
          : existing
            ? t("reminderOff")
            : t("remindTomorrow")}
      </button>
      {failed && (
        <p role="alert" className="ttc-save-error">
          {t("reminderError")}
        </p>
      )}
    </div>
  );
}

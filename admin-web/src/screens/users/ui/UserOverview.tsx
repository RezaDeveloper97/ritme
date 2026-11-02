"use client";

import Link from "next/link";
import { useLocale, useTranslations } from "next-intl";
import { useState, type ReactNode } from "react";

import { formatDate, formatNumber } from "@/shared/lib";
import { Badge, ErrorState, Panel, Skeleton } from "@/shared/ui";

import { useUserOverview, type UserOverview } from "../api/directory";
import { CycleLengthChart } from "./CycleLengthChart";

export const OVERVIEW_TABS = [
  "summary",
  "engine",
  "events",
  "messages",
  "subscription",
  "consents",
  "support",
] as const;
type Tab = (typeof OVERVIEW_TABS)[number];

/** Detail tabs of /users/:id (B-N9-04). Health-gated content shows a «request access» CTA until a grant is live. */
export function UserOverviewTabs({
  id,
  children,
}: {
  id: number;
  children: (tab: Tab | "account") => ReactNode;
}) {
  const t = useTranslations("users");
  const [tab, setTab] = useState<Tab | "account">("summary");
  const query = useUserOverview(id);
  return (
    <div className="flex flex-col gap-4">
      <div className="tabs" role="tablist" aria-label={t("tabs.label")}>
        {[...OVERVIEW_TABS, "account" as const].map((k) => (
          <button
            key={k}
            type="button"
            role="tab"
            id={`user-tab-${k}`}
            aria-selected={tab === k}
            aria-current={tab === k ? "page" : undefined}
            className="tab min-h-[44px]"
            onClick={() => setTab(k)}
          >
            {t(`tabs.${k}` as "tabs.summary")}
          </button>
        ))}
      </div>
      <div
        role="tabpanel"
        aria-labelledby={`user-tab-${tab}`}
        className="flex flex-col gap-4"
      >
        {tab === "account" ? (
          children("account")
        ) : query.error && !query.data ? (
          <Panel>
            <ErrorState error={query.error} onRetry={() => query.refetch()} />
          </Panel>
        ) : !query.data ? (
          <div aria-busy="true">
            <Skeleton className="h-40 w-full" />
          </div>
        ) : (
          <TabBody tab={tab} data={query.data} id={id} />
        )}
      </div>
    </div>
  );
}

function RequestAccess({ id }: { id: number }) {
  const t = useTranslations("users");
  return (
    <Panel title={t("health.limitedTitle")}>
      <p className="m-0 mb-3 text-ink-3">{t("health.limitedBody")}</p>
      <Link href={`/health-access?user=${id}`} className="btn btn-primary">
        {t("health.request")}
      </Link>
    </Panel>
  );
}

type RuleInfo = UserOverview["messages"][number]["rule"];

/** «rule key · v3 · kind» under a message or challenge; the title is already null without a health grant. */
function RuleRef({ rule }: { rule: RuleInfo }) {
  const t = useTranslations("users");
  if (!rule) return null;
  return (
    <span dir="ltr" className="text-[13px] text-ink-3">
      {rule.key ?? `#${rule.id}`}
      {rule.version != null
        ? ` · ${t("ruleVersion", { v: rule.version })}`
        : ""}
      {` · ${rule.kind}`}
    </span>
  );
}

function Empty() {
  const t = useTranslations("users");
  return <p className="m-0 p-4 text-ink-3 sm:p-5">{t("tabEmpty")}</p>;
}

function TabBody({
  tab,
  data,
  id,
}: {
  tab: Tab;
  data: UserOverview;
  id: number;
}) {
  const t = useTranslations("users");
  const locale = useLocale();
  const d = (v: string | null) => formatDate(v, locale) || "—";
  const label = (ns: string, key: string) =>
    t.has(`${ns}.${key}` as "goals.ttc")
      ? t(`${ns}.${key}` as "goals.ttc")
      : key;

  switch (tab) {
    case "summary":
      return (
        <>
          <Panel title={t("tabs.summary")} bodyClassName="">
            <dl className="facts">
              <div>
                <dt>{t("ref")}</dt>
                <dd dir="ltr">{data.ref ?? "—"}</dd>
              </div>
              <div>
                <dt>{t("joined")}</dt>
                <dd>{d(data.summary.signed_up_at)}</dd>
              </div>
              <div>
                <dt>{t("lastActivity")}</dt>
                <dd>{d(data.summary.last_activity_at)}</dd>
              </div>
              <div>
                <dt>{t("loggedDays")}</dt>
                <dd>{formatNumber(data.summary.logged_days_30, locale)}</dd>
              </div>
              <div>
                <dt>{t("companions")}</dt>
                <dd>{formatNumber(data.summary.companions, locale)}</dd>
              </div>
              <div>
                <dt>{t("courses")}</dt>
                <dd>{formatNumber(data.summary.courses, locale)}</dd>
              </div>
            </dl>
          </Panel>
          {!data.health_access.granted && <RequestAccess id={id} />}
        </>
      );
    case "engine":
      return (
        <>
          <Panel title={t("engineToday")}>
            {data.engine ? (
              <>
                <dl className="facts">
                  <div>
                    <dt>{t("mode")}</dt>
                    <dd>{label("modes", data.engine.mode)}</dd>
                  </div>
                  <div>
                    <dt>{t("mainPhase")}</dt>
                    <dd>{label("phases", data.engine.main_phase)}</dd>
                  </div>
                  <div>
                    <dt>{t("confidence")}</dt>
                    <dd>
                      {data.engine.confidence != null
                        ? label("confidences", String(data.engine.confidence))
                        : "—"}
                    </dd>
                  </div>
                  <div>
                    <dt>{t("dataQuality")}</dt>
                    <dd>
                      {data.engine.data_quality
                        ? label("dataQualities", data.engine.data_quality)
                        : "—"}
                    </dd>
                  </div>
                  <div>
                    <dt>{t("daysLate")}</dt>
                    <dd>
                      {data.engine.days_late != null
                        ? formatNumber(data.engine.days_late, locale)
                        : "—"}
                    </dd>
                  </div>
                </dl>
                <div className="mt-3 flex flex-wrap gap-2">
                  {data.engine.warnings.length === 0 ? (
                    <span className="text-ink-3">{t("noWarnings")}</span>
                  ) : (
                    data.engine.warnings.map((w) => (
                      <Badge
                        key={w}
                        tone={
                          w === data.engine?.main_warning ? "amber" : "neutral"
                        }
                      >
                        {label("warnings", w)}
                      </Badge>
                    ))
                  )}
                </div>
              </>
            ) : (
              <p className="m-0 text-ink-3">{t("engineNone")}</p>
            )}
          </Panel>
          <Panel title={t("cycleHistory")}>
            {data.cycle_lengths && data.cycle_lengths.length > 0 ? (
              <CycleLengthChart lengths={data.cycle_lengths} />
            ) : (
              <p className="m-0 text-ink-3">{t("cycleHistoryEmpty")}</p>
            )}
          </Panel>
          {data.health ? (
            <Panel title={t("tabs.engine")}>
              <dl className="facts">
                <div>
                  <dt>{t("goal")}</dt>
                  <dd>
                    {data.health.user_goal
                      ? label("goals", data.health.user_goal)
                      : "—"}
                  </dd>
                </div>
              </dl>
              <p className="m-0 mt-3 text-[13px] text-ink-3">
                {t("health.engineNote")}
              </p>
            </Panel>
          ) : (
            <RequestAccess id={id} />
          )}
        </>
      );
    case "events":
      return (
        <Panel title={t("tabs.events")} bodyClassName="">
          {data.events.length === 0 ? (
            <Empty />
          ) : (
            <ul className="m-0 list-none p-0">
              {data.events.map((e, i) => (
                <li
                  key={`${e.kind}-${e.at}-${i}`}
                  className="flex justify-between gap-3 border-b border-line px-4 py-3 last:border-b-0"
                >
                  <span>{label("eventKinds", e.kind)}</span>
                  <span className="cell-num text-ink-3">{d(e.at)}</span>
                </li>
              ))}
            </ul>
          )}
        </Panel>
      );
    case "messages":
      return (
        <>
          <Panel title={t("messagesShown")} bodyClassName="">
            {data.messages.length === 0 ? (
              <Empty />
            ) : (
              <ul className="m-0 list-none p-0">
                {data.messages.map((m, i) => (
                  <li
                    key={i}
                    className="flex flex-wrap justify-between gap-3 border-b border-line px-4 py-3 last:border-b-0"
                  >
                    <span className="flex flex-col">
                      <span>{m.title ?? label("placements", m.placement)}</span>
                      <RuleRef rule={m.rule} />
                    </span>
                    <span className="flex gap-2">
                      <Badge tone="neutral">
                        {label("messageEvents", m.event)}
                      </Badge>
                      <span className="cell-num text-ink-3">
                        {d(m.created_at)}
                      </span>
                    </span>
                  </li>
                ))}
              </ul>
            )}
          </Panel>
          <Panel title={t("challengesJoined")} bodyClassName="">
            {data.challenges.length === 0 ? (
              <Empty />
            ) : (
              <ul className="m-0 list-none p-0">
                {data.challenges.map((c, i) => (
                  <li
                    key={i}
                    className="flex justify-between gap-3 border-b border-line px-4 py-3 last:border-b-0"
                  >
                    <span className="flex flex-col">
                      <span>{c.title ?? "—"}</span>
                      <RuleRef rule={c.rule} />
                    </span>
                    <span className="flex gap-2">
                      <Badge tone="neutral">{c.status}</Badge>
                      <span className="cell-num text-ink-3">
                        {d(c.started_on)}
                      </span>
                    </span>
                  </li>
                ))}
              </ul>
            )}
          </Panel>
          {data.messages.some((m) => m.title === null) &&
            !data.health_access.granted && <RequestAccess id={id} />}
        </>
      );
    case "subscription":
      return (
        <Panel title={t("tabs.subscription")} bodyClassName="">
          {data.subscriptions.length === 0 ? (
            <Empty />
          ) : (
            <ul className="m-0 list-none p-0">
              {data.subscriptions.map((s, i) => (
                <li
                  key={i}
                  className="flex flex-wrap justify-between gap-3 border-b border-line px-4 py-3 last:border-b-0"
                >
                  <span>{s.plan ?? s.source}</span>
                  <span className="flex gap-2">
                    <Badge tone={s.status === "active" ? "green" : "neutral"}>
                      {s.status}
                    </Badge>
                    <span className="cell-num text-ink-3">
                      {d(s.starts_at)} – {d(s.ends_at)}
                    </span>
                  </span>
                </li>
              ))}
            </ul>
          )}
        </Panel>
      );
    case "consents":
      return (
        <Panel title={t("tabs.consents")} bodyClassName="">
          {data.consents.length === 0 ? (
            <Empty />
          ) : (
            <ul className="m-0 list-none p-0">
              {data.consents.map((c) => (
                <li
                  key={c.consent}
                  className="flex justify-between gap-3 border-b border-line px-4 py-3 last:border-b-0"
                >
                  <span dir="ltr">
                    {c.consent}
                    {c.version ? ` v${c.version}` : ""}
                  </span>
                  <span className="flex gap-2">
                    <Badge tone={c.granted ? "green" : "red"}>
                      {c.granted ? t("consentGranted") : t("consentRevoked")}
                    </Badge>
                    <span className="cell-num text-ink-3">
                      {d(c.revoked_at ?? c.granted_at)}
                    </span>
                  </span>
                </li>
              ))}
            </ul>
          )}
        </Panel>
      );
    case "support":
      return (
        <Panel title={t("tabs.support")} bodyClassName="">
          {data.support_reports.length === 0 ? (
            <Empty />
          ) : (
            <ul className="m-0 list-none p-0">
              {data.support_reports.map((r) => (
                <li
                  key={r.id}
                  className="flex justify-between gap-3 border-b border-line px-4 py-3 last:border-b-0"
                >
                  <span>
                    #{formatNumber(r.id, locale)}{" "}
                    {r.app_version ? (
                      <span dir="ltr" className="text-ink-3">
                        {r.app_version}
                      </span>
                    ) : null}
                  </span>
                  <span className="flex gap-2">
                    <Badge tone={r.status === "open" ? "amber" : "neutral"}>
                      {r.status}
                    </Badge>
                    <span className="cell-num text-ink-3">
                      {d(r.created_at)}
                    </span>
                  </span>
                </li>
              ))}
            </ul>
          )}
          <p className="m-0 p-4 text-[13px] text-ink-3 sm:p-5">
            {t("supportNote")}
          </p>
        </Panel>
      );
  }
}

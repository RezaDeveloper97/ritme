"use client";

import { useLocale, useTranslations } from "next-intl";
import { useState } from "react";

import { cn, formatDateTime, formatNumber } from "@/shared/lib";
import {
  Badge,
  DataTable,
  ErrorState,
  Panel,
  Skeleton,
  type Column,
} from "@/shared/ui";

import {
  useSelfTestRun,
  useSelfTestRuns,
  type SelfTestResult,
  type SelfTestRun,
} from "../api/safety";
import { failureLines } from "../lib/runs";

/**
 * The nightly safety self-test (T-M10-24): the last 30 persisted runs, and for the selected run each scenario's
 * status, threshold and failure detail (synthetic inputs only — never user data).
 */
export function SelfTestRuns() {
  const t = useTranslations("safety.selftest");
  const runs = useSelfTestRuns();
  const [picked, setPicked] = useState<string | null>(null);
  const list = runs.data?.runs ?? [];
  const selected = picked ?? list[0]?.run_id ?? null;
  const run = useSelfTestRun(selected);

  return (
    <Panel title={t("title")} bodyClassName="">
      <p className="panel-note">{t("note")}</p>
      {runs.error && !runs.data ? (
        <ErrorState error={runs.error} onRetry={() => runs.refetch()} />
      ) : (
        <div className="grid gap-4 p-3 lg:grid-cols-[minmax(0,18rem)_1fr]">
          <RunsList
            runs={runs.data?.runs}
            selected={selected}
            onSelect={setPicked}
          />
          <div className="min-w-0">
            {run.error && !run.data ? (
              <ErrorState error={run.error} onRetry={() => run.refetch()} />
            ) : (
              <RunResults
                results={run.data?.results}
                loading={selected != null && run.isPending}
              />
            )}
          </div>
        </div>
      )}
    </Panel>
  );
}

/** The runs, newest first: date and passed/total, red when any scenario failed. */
export function RunsList({
  runs,
  selected,
  onSelect,
}: {
  runs: SelfTestRun[] | undefined;
  selected: string | null;
  onSelect: (runId: string) => void;
}) {
  const t = useTranslations("safety.selftest");
  const locale = useLocale();
  if (!runs) {
    return (
      <div className="flex flex-col gap-2">
        {Array.from({ length: 4 }, (_, i) => (
          <Skeleton key={i} className="h-11 w-full" />
        ))}
      </div>
    );
  }
  if (runs.length === 0)
    return <p className="text-ink-3 m-0 text-sm">{t("empty")}</p>;
  return (
    <ul
      className="m-0 flex list-none flex-col gap-1 p-0"
      aria-label={t("runsLabel")}
    >
      {runs.map((r) => (
        <li key={r.run_id}>
          <button
            type="button"
            aria-pressed={selected === r.run_id}
            onClick={() => onSelect(r.run_id)}
            className={cn(
              "btn btn-sm min-h-11 w-full justify-between gap-2",
              selected === r.run_id ? "btn-primary" : "btn-ghost",
            )}
          >
            <span>
              {r.started_at ? formatDateTime(r.started_at, locale) : "—"}
            </span>
            <Badge tone={r.failed > 0 ? "red" : "data"}>
              {r.failed > 0
                ? t("failedCount", { n: formatNumber(r.failed, locale) })
                : t("allPassed")}
              {" · "}
              <span dir="ltr">{`${formatNumber(r.passed, locale)}/${formatNumber(r.scenarios, locale)}`}</span>
            </Badge>
          </button>
        </li>
      ))}
    </ul>
  );
}

/** One run's scenarios (failures first): status, kind, threshold and the failing probes / problems. */
export function RunResults({
  results,
  loading,
}: {
  results: SelfTestResult[] | undefined;
  loading: boolean;
}) {
  const t = useTranslations("safety.selftest");
  const locale = useLocale();
  const columns: Column<SelfTestResult>[] = [
    {
      key: "scenario",
      header: t("scenario"),
      cell: (r) => (
        <span className="flex flex-col gap-0.5">
          <span className="font-mono text-xs font-semibold" dir="ltr">
            {r.scenario}
          </span>
          {r.detail?.threshold ? (
            <span className="text-ink-3 text-xs" dir="ltr">
              {r.detail.threshold}
            </span>
          ) : null}
        </span>
      ),
    },
    { key: "kind", header: t("kind"), cell: (r) => t(`kinds.${r.kind}`) },
    {
      key: "status",
      header: t("status"),
      cell: (r) => (
        <Badge tone={r.passed ? "data" : "red"}>
          {t(r.passed ? "passed" : "failed")}
        </Badge>
      ),
    },
    {
      key: "detail",
      header: t("detail"),
      cell: (r) => {
        const lines = failureLines(r);
        if (lines.length === 0) {
          return (
            <span className="text-ink-3 text-xs">
              {t("probes", {
                n: formatNumber(r.detail?.probes?.length ?? 0, locale),
              })}
            </span>
          );
        }
        return (
          <ul
            className="text-danger-deep m-0 flex list-none flex-col gap-0.5 p-0 text-xs"
            dir="ltr"
          >
            {lines.map((l) => (
              <li key={l}>{l}</li>
            ))}
          </ul>
        );
      },
    },
    {
      key: "duration",
      header: t("duration"),
      cell: (r) => t("ms", { n: formatNumber(r.duration_ms, locale) }),
      className: "num whitespace-nowrap",
    },
  ];
  return (
    <DataTable
      columns={columns}
      items={results}
      rowKey={(r) => r.scenario}
      loading={loading}
      emptyText={t("pick")}
      caption={t("resultsCaption")}
    />
  );
}

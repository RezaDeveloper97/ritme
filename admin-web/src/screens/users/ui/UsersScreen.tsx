"use client";

import Link from "next/link";
import { useLocale, useTranslations } from "next-intl";
import { useRouter } from "next/navigation";
import { useState } from "react";

import { useCan } from "@/features/auth";
import { formatDate, formatNumber, useListParams } from "@/shared/lib";
import {
  Badge,
  Button,
  DataTable,
  Pagination,
  Panel,
  SearchInput,
  Select,
  toast,
  type BadgeTone,
  type Column,
} from "@/shared/ui";

import {
  ACTIVITIES,
  CHURNS,
  downloadDirectoryCsv,
  PLANS,
  useDirectory,
  type DirectoryRow,
} from "../api/directory";
import type { Segment } from "../api/segments";
import {
  filtersToSegment,
  isFiltered,
  LIST_DEFAULTS,
  segmentToSearch,
} from "../model/segment-filters";
import { SegmentsPanel } from "./SegmentsPanel";

/** The last 12 signup months as YYYY-MM (cohort filter). */
function cohortOptions(): string[] {
  const now = new Date();
  return Array.from({ length: 12 }, (_, i) => {
    const d = new Date(now.getFullYear(), now.getMonth() - i, 1);
    return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}`;
  });
}

type T = ReturnType<typeof useTranslations<"users">>;
/** A label from users.<ns>.<value>; unknown values show as-is. */
function enumLabel(t: T, ns: string, value: string): string {
  const key = `${ns}.${value}` as "plans.free";
  return t.has(key) ? t(key) : value;
}

const CHURN_TONE: Record<string, BadgeTone> = {
  new: "data",
  low: "green",
  medium: "amber",
  high: "red",
};

/** /users — v2 directory: masked mobile, plan, activity, churn risk; filters, saved segments, anonymised export (B-N9-04). */
export function UsersScreen() {
  const t = useTranslations("users");
  const tc = useTranslations("common");
  const locale = useLocale();
  const router = useRouter();
  const canExport = useCan("export");
  const list = useListParams(LIST_DEFAULTS);
  const query = useDirectory(list.query);
  const [exporting, setExporting] = useState(false);

  const f = list.params.filters;
  const filtered = isFiltered(list.params.q, f);
  const segmentFilters = filtersToSegment(f);
  const hasSegmentFilters = Object.keys(segmentFilters).length > 0;
  const options = query.data?.options ?? {};
  const opt = (value: string, label: string) => ({ value, label });
  /** Option list: server vocabulary when present, else the static fallback. */
  const vocab = (key: string, fallback: readonly string[], ns: string) => [
    opt("", t("statusAll")),
    ...(options[key] ?? fallback).map((v) => opt(v, enumLabel(t, ns, v))),
  ];

  const applySegment = (seg: Segment) => {
    const qs = segmentToSearch(seg.filters);
    router.replace(qs ? `?${qs}` : "?", { scroll: false });
  };
  const onExport = async () => {
    setExporting(true);
    try {
      await downloadDirectoryCsv(list.query);
    } catch {
      toast.error(t("exportFailed"));
    } finally {
      setExporting(false);
    }
  };

  const columns: Column<DirectoryRow>[] = [
    {
      key: "ref",
      header: t("ref"),
      cell: (u) => <span dir="ltr">{u.ref ?? "—"}</span>,
      className: "cell-num whitespace-nowrap",
    },
    {
      key: "mobile",
      header: t("mobile"),
      cell: (u) => <span dir="ltr">{u.mobile_masked}</span>,
      className: "cell-num",
    },
    {
      key: "mode",
      header: t("mode"),
      cell: (u) => (u.mode ? enumLabel(t, "modes", u.mode) : "—"),
      className: "whitespace-nowrap",
    },
    {
      key: "plan",
      header: t("plan"),
      cell: (u) => (
        <Badge tone={u.plan === "free" ? "neutral" : "brand"}>
          {enumLabel(t, "plans", u.plan)}
        </Badge>
      ),
    },
    {
      key: "last",
      header: t("lastActivity"),
      cell: (u) => formatDate(u.last_activity_at, locale) || "—",
      className: "cell-num whitespace-nowrap",
    },
    {
      key: "days",
      header: t("loggedDays"),
      cell: (u) => formatNumber(u.logged_days_30, locale),
      className: "cell-num",
    },
    {
      key: "quality",
      header: t("dataQuality"),
      cell: (u) =>
        u.data_quality ? enumLabel(t, "dataQualities", u.data_quality) : "—",
      className: "whitespace-nowrap",
    },
    {
      key: "warning",
      header: t("mainWarning"),
      cell: (u) =>
        u.main_warning ? (
          <Badge tone="amber">{enumLabel(t, "warnings", u.main_warning)}</Badge>
        ) : (
          "—"
        ),
    },
    {
      key: "churn",
      header: t("churnRisk"),
      cell: (u) => (
        <Badge tone={CHURN_TONE[u.churn_risk] ?? "neutral"}>
          {enumLabel(t, "churn", u.churn_risk)}
        </Badge>
      ),
    },
    {
      key: "status",
      header: t("status"),
      cell: (u) =>
        u.is_blocked ? (
          <Badge tone="red">{tc("blocked")}</Badge>
        ) : (
          <Badge tone="green">{tc("active")}</Badge>
        ),
    },
    {
      key: "joined",
      header: t("joined"),
      cell: (u) => formatDate(u.created_at, locale),
      className: "cell-num whitespace-nowrap text-ink-3",
    },
    {
      key: "open",
      header: <span className="sr-only">{t("actions")}</span>,
      cell: (u) => (
        <Link
          href={`/users/${u.id}`}
          className="btn btn-sm"
          onClick={(e) => e.stopPropagation()}
        >
          {t("view")}
        </Link>
      ),
      className: "cell-actions",
    },
  ];

  return (
    <div className="flex flex-col gap-4">
      <SegmentsPanel
        current={segmentFilters}
        activeId={f.segment ? Number(f.segment) : null}
        canSave={hasSegmentFilters}
        onApply={applySegment}
      />
      <Panel
        title={t("title")}
        bodyClassName=""
        actions={
          <>
            {canExport && (
              <Button size="sm" onClick={onExport} loading={exporting}>
                {t("exportAnonymised")}
              </Button>
            )}
          </>
        }
      >
        <div className="filter-bar">
          <SearchInput
            value={list.params.q}
            onChange={list.setSearch}
            placeholder={t("searchPlaceholder")}
          />
          <Select
            label={t("status")}
            className="w-36"
            value={f.status}
            onChange={(e) => list.setFilter("status", e.target.value)}
            options={[
              opt("all", t("statusAll")),
              opt("active", tc("active")),
              opt("blocked", tc("blocked")),
            ]}
          />
          <Select
            label={t("plan")}
            className="w-36"
            value={f.plan}
            onChange={(e) => list.setFilter("plan", e.target.value)}
            options={[
              opt("", t("statusAll")),
              ...PLANS.map((p) => opt(p, enumLabel(t, "plans", p))),
            ]}
          />
          <Select
            label={t("lastActivity")}
            className="w-40"
            value={f.activity}
            onChange={(e) => list.setFilter("activity", e.target.value)}
            options={[
              opt("", t("statusAll")),
              ...ACTIVITIES.map((a) => opt(a, enumLabel(t, "activities", a))),
            ]}
          />
          <Select
            label={t("cohort")}
            className="w-36"
            value={f.cohort}
            onChange={(e) => list.setFilter("cohort", e.target.value)}
            options={[
              opt("", t("statusAll")),
              ...cohortOptions().map((c) => opt(c, c)),
            ]}
          />
          <Select
            label={t("churnRisk")}
            className="w-36"
            value={f.churn}
            onChange={(e) => list.setFilter("churn", e.target.value)}
            options={vocab("churn", CHURNS, "churn")}
          />
          <Select
            label={t("mode")}
            className="w-40"
            value={f.mode}
            onChange={(e) => list.setFilter("mode", e.target.value)}
            options={vocab("mode", [], "modes")}
          />
          <Select
            label={t("dataQuality")}
            className="w-40"
            value={f.data_quality}
            onChange={(e) => list.setFilter("data_quality", e.target.value)}
            options={vocab("data_quality", [], "dataQualities")}
          />
          <Select
            label={t("mainWarning")}
            className="w-52"
            value={f.main_warning}
            onChange={(e) => list.setFilter("main_warning", e.target.value)}
            options={vocab("main_warning", [], "warnings")}
          />
        </div>
        <DataTable
          columns={columns}
          items={query.data?.items}
          rowKey={(u) => u.id}
          loading={query.isPending}
          error={query.error}
          onRetry={() => query.refetch()}
          onRowClick={(u) => router.push(`/users/${u.id}`)}
          emptyText={filtered ? t("emptyFiltered") : t("empty")}
          caption={t("title")}
        />
        <Pagination meta={query.data?.meta} onPage={list.setPage} />
        <p className="m-0 p-4 text-[13px] text-ink-3 sm:p-5">
          {t("privacyNote")}
        </p>
      </Panel>
    </div>
  );
}

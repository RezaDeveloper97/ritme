"use client";

import Link from "next/link";
import { useLocale, useTranslations } from "next-intl";
import { usePathname, useRouter, useSearchParams } from "next/navigation";
import { useEffect, useState } from "react";

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

const SEGMENTS_KEY = "ritme-admin-user-segments";
interface Segment {
  name: string;
  qs: string;
}

function readSegments(): Segment[] {
  try {
    const raw: unknown = JSON.parse(localStorage.getItem(SEGMENTS_KEY) ?? "[]");
    return Array.isArray(raw)
      ? (raw as Segment[]).filter(
          (s) => typeof s.name === "string" && typeof s.qs === "string",
        )
      : [];
  } catch {
    return [];
  }
}

function writeSegments(list: Segment[]): void {
  try {
    localStorage.setItem(SEGMENTS_KEY, JSON.stringify(list));
  } catch {
    // private mode / blocked storage: segments just don't persist
  }
}

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
  const pathname = usePathname();
  const search = useSearchParams();
  const canExport = useCan("export");
  const list = useListParams({
    status: "all",
    plan: "",
    activity: "",
    cohort: "",
    churn: "",
  });
  const query = useDirectory(list.query);
  const [segments, setSegments] = useState<Segment[]>([]);
  const [exporting, setExporting] = useState(false);
  useEffect(() => setSegments(readSegments()), []);

  const f = list.params.filters;
  const filtered =
    Boolean(list.params.q) ||
    f.status !== "all" ||
    Boolean(f.plan || f.activity || f.cohort || f.churn);
  const opt = (value: string, label: string) => ({ value, label });

  const saveSegment = () => {
    const name = window.prompt(t("segmentName"))?.trim();
    if (!name) return;
    const qs = search.toString();
    const next = [
      ...segments.filter((s) => s.name !== name),
      { name, qs },
    ].slice(-12);
    setSegments(next);
    writeSegments(next);
  };
  const applySegment = (name: string) => {
    const seg = segments.find((s) => s.name === name);
    if (seg)
      router.replace(seg.qs ? `${pathname}?${seg.qs}` : pathname, {
        scroll: false,
      });
  };
  const removeSegment = (name: string) => {
    const next = segments.filter((s) => s.name !== name);
    setSegments(next);
    writeSegments(next);
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
      cell: (u) => <span dir="ltr">{u.ref}</span>,
      className: "cell-num whitespace-nowrap",
    },
    {
      key: "mobile",
      header: t("mobile"),
      cell: (u) => <span dir="ltr">{u.mobile_masked}</span>,
      className: "cell-num",
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
    <Panel
      title={t("title")}
      bodyClassName=""
      actions={
        <>
          <Button size="sm" onClick={saveSegment} disabled={!filtered}>
            {t("saveSegment")}
          </Button>
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
          options={[
            opt("", t("statusAll")),
            ...CHURNS.map((c) => opt(c, enumLabel(t, "churn", c))),
          ]}
        />
        {segments.length > 0 && (
          <Select
            label={t("segments")}
            className="w-44"
            value=""
            onChange={(e) => applySegment(e.target.value)}
            options={[
              opt("", t("pickSegment")),
              ...segments.map((s) => opt(s.name, s.name)),
            ]}
          />
        )}
      </div>
      {segments.length > 0 && (
        <ul className="m-0 flex list-none flex-wrap gap-2 px-4 pb-3 sm:px-5">
          {segments.map((s) => (
            <li key={s.name}>
              <Badge tone="neutral">
                {s.name}{" "}
                <button
                  type="button"
                  className="min-h-[24px] min-w-[24px]"
                  aria-label={t("removeSegment", { name: s.name })}
                  onClick={() => removeSegment(s.name)}
                >
                  ×
                </button>
              </Badge>
            </li>
          ))}
        </ul>
      )}
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
  );
}

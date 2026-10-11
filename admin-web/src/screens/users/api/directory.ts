"use client";

import { keepPreviousData, useQuery } from "@tanstack/react-query";
import { z } from "zod";

import {
  api,
  ApiError,
  buildUrl,
  pagedSchema,
  type QueryValue,
} from "@/shared/api";
import { ADMIN_API_BASE_URL } from "@/shared/config";

/** Users list v2 + detail overview (admin-api.md §6c, B-N9-04). No health value in any of these payloads without a grant. */

export const PLANS = ["free", "plus", "trial"] as const;
export const ACTIVITIES = ["today", "7d", "30d", "inactive"] as const;
export const CHURNS = ["low", "medium", "high"] as const;
export type Churn = (typeof CHURNS)[number];

const directoryRowSchema = z.object({
  id: z.number(),
  ref: z.string(),
  mobile_masked: z.string(),
  plan: z.string(),
  is_blocked: z.boolean(),
  last_activity_at: z.string().nullable(),
  logged_days_30: z.number(),
  churn_risk: z.string(),
  created_at: z.string().nullable(),
});
export type DirectoryRow = z.infer<typeof directoryRowSchema>;

export function useDirectory(query: Record<string, QueryValue>) {
  return useQuery({
    queryKey: ["users", "directory", query],
    queryFn: ({ signal }) =>
      api.get("/users/directory", {
        query,
        schema: pagedSchema(directoryRowSchema, {}),
        signal,
      }),
    placeholderData: keepPreviousData,
  });
}

/** Anonymised CSV of the filtered list (needs the export capability; audited server-side). */
export async function downloadDirectoryCsv(
  query: Record<string, QueryValue>,
): Promise<void> {
  const rest = { ...query };
  delete rest.page;
  delete rest.per_page;
  const res = await fetch(
    buildUrl(ADMIN_API_BASE_URL, "/users/directory/export", rest),
    { credentials: "include" },
  );
  if (!res.ok)
    throw new ApiError({
      status: res.status,
      code: res.status === 403 ? "forbidden" : "http_error",
    });
  const url = URL.createObjectURL(await res.blob());
  const a = document.createElement("a");
  a.href = url;
  a.download = "users-anonymised.csv";
  a.click();
  URL.revokeObjectURL(url);
}

const at = z.string().nullable();
const overviewSchema = z.object({
  ref: z.string(),
  summary: z.object({
    signed_up_at: at,
    last_activity_at: at,
    logged_days_30: z.number(),
    companions: z.number(),
    courses: z.number(),
    is_blocked: z.boolean(),
  }),
  events: z.array(z.object({ kind: z.string(), at: z.string() })),
  messages: z.array(
    z.object({
      title: z.string().nullable(),
      placement: z.string(),
      event: z.string(),
      created_at: at,
    }),
  ),
  challenges: z.array(
    z.object({
      title: z.string().nullable(),
      status: z.string(),
      started_on: at,
    }),
  ),
  subscriptions: z.array(
    z.object({
      status: z.string(),
      source: z.string(),
      starts_at: at,
      ends_at: at,
      auto_renew: z.boolean(),
      plan: z.string().nullable(),
    }),
  ),
  consents: z.array(
    z.object({
      consent: z.string(),
      granted: z.boolean(),
      version: z.number().nullable(),
      granted_at: at,
      revoked_at: at,
    }),
  ),
  support_reports: z.array(
    z.object({
      id: z.number(),
      status: z.string(),
      app_version: z.string().nullable(),
      created_at: at,
    }),
  ),
  health_access: z.object({ granted: z.boolean(), expires_at: at }),
  /** Health-gated: null unless the admin holds an approved grant for this user. */
  health: z.object({ user_goal: z.string().nullable() }).nullable(),
});
export type UserOverview = z.infer<typeof overviewSchema>;

export function useUserOverview(id: number) {
  return useQuery({
    queryKey: ["users", "overview", id],
    queryFn: ({ signal }) =>
      api.get(`/users/${id}/overview`, { schema: overviewSchema, signal }),
    staleTime: 0,
    gcTime: 0, // a grant can expire: never serve a cached unlocked view
  });
}

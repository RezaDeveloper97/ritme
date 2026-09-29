'use client';

import { useQuery } from '@tanstack/react-query';

import { type ApiEnvelope, apiClient, getApiErrorStatus } from '@/shared/api';
import { isAuthenticated } from '@/shared/session';

import { datingPreviewBody, isDatingPreviewReady, isV2Week } from '../model/v2';
import type {
  DatingPreview,
  DatingPreviewInput,
  PregnancyAlertsV2,
  PregnancyCalendar,
  PregnancyDay,
  PregnancyReport,
  PregnancyToday,
  PregnancyWeek,
  ReportRange,
  SetupCopy,
} from '../model/v2-types';
import { pregnancyKeys } from './keys';
import {
  datingPreviewSchema,
  pregnancyAlertsV2Schema,
  pregnancyCalendarSchema,
  pregnancyDaySchema,
  pregnancyReportSchema,
  pregnancyTodaySchema,
  pregnancyWeekSchema,
  setupCopySchema,
} from './v2-schema';

/*
 * Reads for `/api/v1/pregnancy/v2/*` (M7, Go-only — docs/pregnancy-v2/README.md
 * § API). Server state lives in TanStack Query (§8); every hook is disabled
 * until a token exists. The v2 mutations live in `features/track-pregnancy`.
 *
 * "Not in pregnancy mode" (409 `pregnancy_not_active`) resolves to `null`
 * instead of an error, so a screen can send the user to Setup rather than show
 * a failure. Privacy (§11): never log what these return.
 */

const V2 = '/pregnancy/v2';

/** 409 = pregnancy mode is off; 404 = nothing there (e.g. a week out of range). */
function isAbsent(error: unknown): boolean {
  const status = getApiErrorStatus(error);
  return status === 404 || status === 409;
}

async function getOrNull<T>(path: string, parse: (raw: unknown) => T, params?: Record<string, string>) {
  try {
    const { data } = await apiClient.get<ApiEnvelope<unknown>>(path, params ? { params } : undefined);
    return parse(data.data);
  } catch (error) {
    if (isAbsent(error)) return null;
    throw error;
  }
}

// ── Today ──────────────────────────────────────────────────────

/** GET /pregnancy/v2/today — the whole Today screen in one payload. */
export function fetchPregnancyToday(): Promise<PregnancyToday | null> {
  return getOrNull(`${V2}/today`, (raw) => pregnancyTodaySchema.parse(raw));
}

export function usePregnancyToday(options: { enabled?: boolean } = {}) {
  return useQuery({
    queryKey: pregnancyKeys.v2.today(),
    queryFn: fetchPregnancyToday,
    enabled: isAuthenticated() && options.enabled !== false,
    staleTime: 60_000,
    retry: false,
  });
}

// ── Week ───────────────────────────────────────────────────────

/** GET /pregnancy/v2/weeks/{n} (1–42) — details, tasks with done state, tip. */
export function fetchPregnancyWeek(week: number): Promise<PregnancyWeek | null> {
  return getOrNull(`${V2}/weeks/${week}`, (raw) => pregnancyWeekSchema.parse(raw));
}

export function usePregnancyWeek(week: number | null) {
  return useQuery({
    queryKey: pregnancyKeys.v2.week(week ?? 0),
    queryFn: () => fetchPregnancyWeek(week as number),
    enabled: isAuthenticated() && week != null && isV2Week(week),
    // Admin-authored content: it changes rarely; the task state is refreshed by
    // the week-state mutation.
    staleTime: 5 * 60_000,
    retry: false,
  });
}

// ── Day log ────────────────────────────────────────────────────

/** GET /pregnancy/v2/days/{date} — the merged day the Log screen edits. */
export function fetchPregnancyDay(date: string): Promise<PregnancyDay | null> {
  return getOrNull(`${V2}/days/${date}`, (raw) => pregnancyDaySchema.parse(raw));
}

export function usePregnancyDay(date: string) {
  return useQuery({
    queryKey: pregnancyKeys.v2.day(date),
    queryFn: () => fetchPregnancyDay(date),
    enabled: isAuthenticated() && date.length > 0,
    staleTime: 60_000,
    retry: false,
  });
}

// ── Calendar ───────────────────────────────────────────────────

/** GET /pregnancy/v2/calendar?month=YYYY-MM — markers, visits, care plan. */
export function fetchPregnancyCalendar(month: string): Promise<PregnancyCalendar | null> {
  return getOrNull(`${V2}/calendar`, (raw) => pregnancyCalendarSchema.parse(raw), { month });
}

export function usePregnancyCalendar(month: string) {
  return useQuery({
    queryKey: pregnancyKeys.v2.calendar(month),
    queryFn: () => fetchPregnancyCalendar(month),
    enabled: isAuthenticated() && /^\d{4}-\d{2}$/.test(month),
    staleTime: 60_000,
    retry: false,
  });
}

// ── Alerts ─────────────────────────────────────────────────────

/** GET /pregnancy/v2/alerts — the last 7 days + the level legend. */
export function fetchPregnancyAlertsV2(): Promise<PregnancyAlertsV2 | null> {
  return getOrNull(`${V2}/alerts`, (raw) => pregnancyAlertsV2Schema.parse(raw));
}

export function usePregnancyAlertsV2() {
  return useQuery({
    queryKey: pregnancyKeys.v2.alerts(),
    queryFn: fetchPregnancyAlertsV2,
    enabled: isAuthenticated(),
    staleTime: 60_000,
    retry: false,
  });
}

// ── Setup copy ─────────────────────────────────────────────────

/** GET /pregnancy/v2/setup-copy — admin-edited `pregnancy_setup` texts (no pregnancy needed). */
export async function fetchSetupCopy(): Promise<SetupCopy> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>(`${V2}/setup-copy`);
  return setupCopySchema.parse(data.data);
}

/**
 * The Setup screens' copy. Until it loads (or when it fails) `data` is
 * undefined and the screens show their bundled text — the seeded rows say the
 * same, so there is no visible jump unless an admin changed them.
 */
export function useSetupCopy() {
  return useQuery({
    queryKey: pregnancyKeys.v2.setupCopy(),
    queryFn: fetchSetupCopy,
    enabled: isAuthenticated(),
    staleTime: 10 * 60_000,
    retry: false,
  });
}

// ── Dating preview (POST, but a pure read — nothing is written) ─

/** POST /pregnancy/v2/dating-preview — weeks, due date, range, basis sentence. */
export async function fetchDatingPreview(input: DatingPreviewInput): Promise<DatingPreview> {
  const { data } = await apiClient.post<ApiEnvelope<unknown>>(`${V2}/dating-preview`, datingPreviewBody(input));
  return datingPreviewSchema.parse(data.data);
}

/**
 * The Setup result step. Runs whenever `input` is complete; the key is the
 * serialized body, so flipping back to an earlier answer is instant. A 422
 * surfaces as the query error (the server's localized message).
 */
export function useDatingPreview(input: DatingPreviewInput | null) {
  const ready = isDatingPreviewReady(input);
  const body = ready ? datingPreviewBody(input) : null;
  return useQuery({
    queryKey: pregnancyKeys.v2.datingPreview(body ? JSON.stringify(body) : ''),
    queryFn: () => fetchDatingPreview(body as DatingPreviewInput),
    enabled: isAuthenticated() && ready,
    staleTime: 5 * 60_000,
    retry: false,
  });
}

// ── Doctor report ──────────────────────────────────────────────

/** GET /pregnancy/v2/report?from=&to= — data for the «گزارش علائم برای پزشک» PDF. */
export async function fetchPregnancyReport(range: ReportRange): Promise<PregnancyReport> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>(`${V2}/report`, {
    params: { from: range.from, to: range.to },
  });
  return pregnancyReportSchema.parse(data.data);
}

/** Usually fetched on demand (`queryClient.fetchQuery`) when the PDF is built. */
export function usePregnancyReport(range: ReportRange | null) {
  return useQuery({
    queryKey: range ? pregnancyKeys.v2.report(range) : pregnancyKeys.v2.reportAll(),
    queryFn: () => fetchPregnancyReport(range as ReportRange),
    enabled: isAuthenticated() && range != null,
    staleTime: 60_000,
    retry: false,
  });
}

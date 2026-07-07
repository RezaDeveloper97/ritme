'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { type ApiEnvelope, apiClient, getApiErrorStatus } from '@/shared/api';
import { isAuthenticated } from '@/shared/session';

import type {
  AlertSummary,
  FetalMovementInput,
  OnboardingInput,
  PregnancyActivation,
  PregnancyAlert,
  PregnancyEnums,
  PregnancyProfile,
  PregnancyStatus,
  SymptomEnums,
  SymptomLogInput,
  WeeklyContent,
  WeeklyEnums,
  WeeklyLogInput,
} from '../model/types';
import {
  alertSummarySchema,
  alertsEnvelopeSchema,
  pregnancyAlertSchema,
  pregnancyEnumsSchema,
  pregnancyProfileEnvelopeSchema,
  pregnancyProfileSchema,
  pregnancyStatusSchema,
  symptomEnumsSchema,
  weeklyContentEnvelopeSchema,
  weeklyEnumsSchema,
} from './schema';

/**
 * Query-key factory for pregnancy mode (CLAUDE.md §8). Weekly resources are
 * keyed by week, daily ones by API date, so each caches independently. All
 * reads and post-mutation invalidation go through here — never raw arrays.
 */
export const pregnancyKeys = {
  all: ['pregnancy'] as const,
  status: () => [...pregnancyKeys.all, 'status'] as const,
  profile: () => [...pregnancyKeys.all, 'profile'] as const,
  enums: () => [...pregnancyKeys.all, 'enums'] as const,
  symptomEnums: () => [...pregnancyKeys.all, 'symptom-enums'] as const,
  weeklyEnums: () => [...pregnancyKeys.all, 'weekly-enums'] as const,
  content: (week: number, locale: string) =>
    [...pregnancyKeys.all, 'content', week, locale] as const,
  symptom: (date: string) => [...pregnancyKeys.all, 'symptom', date] as const,
  weeklyLog: (week: number) => [...pregnancyKeys.all, 'weekly-log', week] as const,
  fetalMovement: (date: string) => [...pregnancyKeys.all, 'fetal-movement', date] as const,
  alerts: () => [...pregnancyKeys.all, 'alerts'] as const,
  alertSummary: () => [...pregnancyKeys.all, 'alert-summary'] as const,
};

/** Drop `undefined` values so a body only carries fields the user actually set. */
function compact<T extends object>(input: T): Record<string, unknown> {
  const out: Record<string, unknown> = {};
  for (const [k, v] of Object.entries(input)) {
    if (v !== undefined) out[k] = v;
  }
  return out;
}

// ── Status & profile ───────────────────────────────────────────
export async function fetchPregnancyStatus(): Promise<PregnancyStatus> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>('/pregnancy/status');
  return pregnancyStatusSchema.parse(data.data);
}

/** Reads pregnancy status — the tracker's single source of truth (§8). */
export function usePregnancyStatus() {
  return useQuery({
    queryKey: pregnancyKeys.status(),
    queryFn: fetchPregnancyStatus,
    enabled: isAuthenticated(),
    staleTime: 5 * 60_000,
    retry: false,
  });
}

/** GET /pregnancy/profile → `{ profile, status }`, or `null` before onboarding. */
export async function fetchPregnancyProfile(): Promise<{
  profile: PregnancyProfile;
  status: PregnancyStatus;
} | null> {
  try {
    const { data } = await apiClient.get<ApiEnvelope<unknown>>('/pregnancy/profile');
    return pregnancyProfileEnvelopeSchema.parse(data.data);
  } catch (error) {
    if (getApiErrorStatus(error) === 404) return null;
    throw error;
  }
}

export function usePregnancyProfile() {
  return useQuery({
    queryKey: pregnancyKeys.profile(),
    queryFn: fetchPregnancyProfile,
    enabled: isAuthenticated(),
    staleTime: 5 * 60_000,
    retry: false,
  });
}

// ── Enums ──────────────────────────────────────────────────────
function enumsQuery<T>(key: readonly unknown[], fetcher: () => Promise<T>) {
  return {
    queryKey: key,
    queryFn: fetcher,
    enabled: isAuthenticated(),
    staleTime: Infinity,
    gcTime: Infinity,
    retry: false,
  } as const;
}

export async function fetchPregnancyEnums(): Promise<PregnancyEnums> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>('/pregnancy/enums');
  return pregnancyEnumsSchema.parse(data.data);
}
export function usePregnancyEnums() {
  return useQuery(enumsQuery(pregnancyKeys.enums(), fetchPregnancyEnums));
}

export async function fetchSymptomEnums(): Promise<SymptomEnums> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>('/pregnancy/symptoms/enums');
  return symptomEnumsSchema.parse(data.data);
}
export function useSymptomEnums() {
  return useQuery(enumsQuery(pregnancyKeys.symptomEnums(), fetchSymptomEnums));
}

export async function fetchWeeklyEnums(): Promise<WeeklyEnums> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>('/pregnancy/weekly/enums');
  return weeklyEnumsSchema.parse(data.data);
}
export function useWeeklyEnums() {
  return useQuery(enumsQuery(pregnancyKeys.weeklyEnums(), fetchWeeklyEnums));
}

// ── Weekly educational content ─────────────────────────────────
/**
 * GET /pregnancy/content/{week}?locale=… — the week's educational modules.
 * Browsers can't override `Accept-Language`, so we pass the app locale as a
 * query param (the backend prefers it). Returns `null` when a week hasn't been
 * authored yet (the API answers 404) so the UI can degrade gracefully.
 */
export async function fetchWeeklyContent(
  week: number,
  locale: string,
): Promise<WeeklyContent | null> {
  try {
    const { data } = await apiClient.get<ApiEnvelope<unknown>>(`/pregnancy/content/${week}`, {
      params: { locale },
    });
    return weeklyContentEnvelopeSchema.parse(data.data);
  } catch (error) {
    if (getApiErrorStatus(error) === 404) return null;
    throw error;
  }
}

export function useWeeklyContent(week: number | null, locale: string) {
  return useQuery({
    queryKey: pregnancyKeys.content(week ?? 0, locale),
    queryFn: () => fetchWeeklyContent(week as number, locale),
    enabled: isAuthenticated() && week != null && week >= 1 && week <= 40,
    staleTime: 60 * 60_000,
    retry: false,
  });
}

// ── Daily symptom log ──────────────────────────────────────────
export async function fetchSymptomLog(date: string): Promise<SymptomLogInput | null> {
  try {
    const { data } = await apiClient.get<ApiEnvelope<{ log: unknown }>>(
      `/pregnancy/symptoms/${date}`,
    );
    // The endpoint wraps the row as `{ log, pregnancy_week }`.
    const raw = (data.data as { log: Record<string, unknown> } | undefined)?.log;
    return (raw ?? null) as SymptomLogInput | null;
  } catch (error) {
    if (getApiErrorStatus(error) === 404) return null;
    throw error;
  }
}

export function useSymptomLog(date: string) {
  return useQuery({
    queryKey: pregnancyKeys.symptom(date),
    queryFn: () => fetchSymptomLog(date),
    enabled: isAuthenticated() && date.length > 0,
    staleTime: 60_000,
    retry: false,
  });
}

/** POST /pregnancy/symptoms — upserts the day's symptoms; may return alerts. */
export function useSaveSymptomLog() {
  const queryClient = useQueryClient();
  return useMutation<PregnancyAlert[], unknown, SymptomLogInput>({
    mutationFn: async (input) => {
      const { data } = await apiClient.post<ApiEnvelope<{ alerts?: unknown[] }>>(
        '/pregnancy/symptoms',
        compact(input),
      );
      const alerts = (data.data as { alerts?: unknown[] } | undefined)?.alerts ?? [];
      return alerts.map((a) => pregnancyAlertSchema.parse(a));
    },
    onSuccess: (_alerts, input) => {
      void queryClient.invalidateQueries({ queryKey: pregnancyKeys.symptom(input.log_date) });
      void queryClient.invalidateQueries({ queryKey: pregnancyKeys.alerts() });
      void queryClient.invalidateQueries({ queryKey: pregnancyKeys.alertSummary() });
    },
  });
}

// ── Weekly checkup log ─────────────────────────────────────────
export async function fetchWeeklyLog(week: number): Promise<WeeklyLogInput | null> {
  try {
    const { data } = await apiClient.get<ApiEnvelope<{ log: unknown }>>(`/pregnancy/weekly/${week}`);
    const raw = (data.data as { log: Record<string, unknown> } | undefined)?.log;
    return (raw ?? null) as WeeklyLogInput | null;
  } catch (error) {
    if (getApiErrorStatus(error) === 404) return null;
    throw error;
  }
}

export function useWeeklyLog(week: number | null) {
  return useQuery({
    queryKey: pregnancyKeys.weeklyLog(week ?? 0),
    queryFn: () => fetchWeeklyLog(week as number),
    enabled: isAuthenticated() && week != null && week >= 1 && week <= 42,
    staleTime: 60_000,
    retry: false,
  });
}

export function useSaveWeeklyLog() {
  const queryClient = useQueryClient();
  return useMutation<PregnancyAlert[], unknown, WeeklyLogInput>({
    mutationFn: async (input) => {
      const { data } = await apiClient.post<ApiEnvelope<{ alerts?: unknown[] }>>(
        '/pregnancy/weekly',
        compact(input),
      );
      const alerts = (data.data as { alerts?: unknown[] } | undefined)?.alerts ?? [];
      return alerts.map((a) => pregnancyAlertSchema.parse(a));
    },
    onSuccess: (_alerts, input) => {
      void queryClient.invalidateQueries({ queryKey: pregnancyKeys.weeklyLog(input.pregnancy_week) });
      void queryClient.invalidateQueries({ queryKey: pregnancyKeys.alerts() });
      void queryClient.invalidateQueries({ queryKey: pregnancyKeys.alertSummary() });
    },
  });
}

// ── Fetal movement ─────────────────────────────────────────────
export async function fetchFetalMovement(date: string): Promise<FetalMovementInput | null> {
  try {
    const { data } = await apiClient.get<ApiEnvelope<{ logs?: unknown[] }>>(
      '/pregnancy/fetal-movement',
      { params: { from: date, to: date } },
    );
    const logs = (data.data as { logs?: unknown[] } | undefined)?.logs ?? [];
    return (logs[0] ?? null) as FetalMovementInput | null;
  } catch (error) {
    if (getApiErrorStatus(error) === 404) return null;
    throw error;
  }
}

export function useFetalMovement(date: string) {
  return useQuery({
    queryKey: pregnancyKeys.fetalMovement(date),
    queryFn: () => fetchFetalMovement(date),
    enabled: isAuthenticated() && date.length > 0,
    staleTime: 60_000,
    retry: false,
  });
}

export function useSaveFetalMovement() {
  const queryClient = useQueryClient();
  return useMutation<PregnancyAlert[], unknown, FetalMovementInput>({
    mutationFn: async (input) => {
      const { data } = await apiClient.post<ApiEnvelope<{ alerts?: unknown[] }>>(
        '/pregnancy/fetal-movement',
        compact(input),
      );
      const alerts = (data.data as { alerts?: unknown[] } | undefined)?.alerts ?? [];
      return alerts.map((a) => pregnancyAlertSchema.parse(a));
    },
    onSuccess: (_alerts, input) => {
      void queryClient.invalidateQueries({ queryKey: pregnancyKeys.fetalMovement(input.log_date) });
      void queryClient.invalidateQueries({ queryKey: pregnancyKeys.alerts() });
      void queryClient.invalidateQueries({ queryKey: pregnancyKeys.alertSummary() });
    },
  });
}

// ── Alerts ─────────────────────────────────────────────────────
export async function fetchPregnancyAlerts(): Promise<{
  alerts: PregnancyAlert[];
  counts: AlertSummary['counts'];
}> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>('/pregnancy/alerts');
  return alertsEnvelopeSchema.parse(data.data);
}

export function usePregnancyAlerts() {
  return useQuery({
    queryKey: pregnancyKeys.alerts(),
    queryFn: fetchPregnancyAlerts,
    enabled: isAuthenticated(),
    staleTime: 60_000,
    retry: false,
  });
}

export async function fetchAlertSummary(): Promise<AlertSummary> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>('/pregnancy/alerts/summary');
  return alertSummarySchema.parse(data.data);
}

export function useAlertSummary() {
  return useQuery({
    queryKey: pregnancyKeys.alertSummary(),
    queryFn: fetchAlertSummary,
    enabled: isAuthenticated(),
    staleTime: 60_000,
    retry: false,
  });
}

/** POST /pregnancy/alerts/{id}/read and /dismiss share this shape. */
export function useAlertAction() {
  const queryClient = useQueryClient();
  return useMutation<void, unknown, { id: number; action: 'read' | 'dismiss' }>({
    mutationFn: async ({ id, action }) => {
      await apiClient.post(`/pregnancy/alerts/${id}/${action}`);
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: pregnancyKeys.alerts() });
      void queryClient.invalidateQueries({ queryKey: pregnancyKeys.alertSummary() });
    },
  });
}

export function useMarkAllAlertsRead() {
  const queryClient = useQueryClient();
  return useMutation<void, unknown, void>({
    mutationFn: async () => {
      await apiClient.post('/pregnancy/alerts/read-all');
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: pregnancyKeys.alerts() });
      void queryClient.invalidateQueries({ queryKey: pregnancyKeys.alertSummary() });
    },
  });
}

// ── Mode & onboarding mutations ────────────────────────────────
/** POST /pregnancy/activate — switch the user into pregnancy mode. */
export function useActivatePregnancy() {
  const queryClient = useQueryClient();
  return useMutation<PregnancyActivation, unknown, void>({
    mutationFn: async () => {
      const { data } = await apiClient.post<ApiEnvelope<{
        pregnancy_mode: boolean;
        cycle_mode: boolean;
        onboarding_required: boolean;
      }>>('/pregnancy/activate');
      const d = data.data!;
      return {
        pregnancyMode: d.pregnancy_mode,
        cycleMode: d.cycle_mode,
        onboardingRequired: d.onboarding_required,
      };
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: pregnancyKeys.all });
      void queryClient.invalidateQueries({ queryKey: ['message'] });
    },
  });
}

/** POST /pregnancy/deactivate — return to cycle mode. */
export function useDeactivatePregnancy() {
  const queryClient = useQueryClient();
  return useMutation<void, unknown, void>({
    mutationFn: async () => {
      await apiClient.post('/pregnancy/deactivate');
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: pregnancyKeys.all });
      void queryClient.invalidateQueries({ queryKey: ['message'] });
    },
  });
}

/** POST /pregnancy/onboarding — supply dating + history; returns the profile. */
export function useCompleteOnboarding() {
  const queryClient = useQueryClient();
  return useMutation<PregnancyProfile, unknown, OnboardingInput>({
    mutationFn: async (input) => {
      const { data } = await apiClient.post<ApiEnvelope<{ profile: unknown }>>(
        '/pregnancy/onboarding',
        compact(input),
      );
      const raw = (data.data as { profile: unknown }).profile;
      return pregnancyProfileSchema.parse(raw);
    },
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: pregnancyKeys.all });
      void queryClient.invalidateQueries({ queryKey: ['message'] });
    },
  });
}

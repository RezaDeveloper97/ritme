'use client';

import { useQuery } from '@tanstack/react-query';

import { type ApiEnvelope, apiClient } from '@/shared/api';
import { isAuthenticated } from '@/shared/session';

import type { LogDay, LogDayChanges, LogDaysRange, LogPreferences, LogTaxonomy } from '../model/log-v2';
import { logDaySchema, logDaysRangeSchema, logPreferencesSchema, logTaxonomySchema } from './log-v2-schema';

/**
 * Query keys of the taxonomy v2 endpoints (`/logs/*`, Go only). `mode` = `undefined` means "the user's
 * current mode" (the server resolves it), which is its own cache entry.
 */
export const logKeys = {
  all: ['logs-v2'] as const,
  taxonomy: (mode?: string) => [...logKeys.all, 'taxonomy', mode ?? 'current'] as const,
  /** Every mode's preferences (and the current-mode entry) — invalidated after any preferences write. */
  preferencesAll: () => [...logKeys.all, 'preferences'] as const,
  preferences: (mode?: string) => [...logKeys.all, 'preferences', mode ?? 'current'] as const,
  days: () => [...logKeys.all, 'days'] as const,
  ranges: () => [...logKeys.days(), 'range'] as const,
  range: (from: string, to: string) => [...logKeys.ranges(), from, to] as const,
  day: (date: string) => [...logKeys.days(), 'day', date] as const,
};

/** GET /logs/taxonomy[?mode=] — categories, params and value sets, labelled in the request language. */
export async function fetchLogTaxonomy(mode?: string): Promise<LogTaxonomy> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>('/logs/taxonomy', {
    params: mode ? { mode } : undefined,
  });
  return logTaxonomySchema.parse(data.data);
}

export function useLogTaxonomy(mode?: string) {
  return useQuery({
    queryKey: logKeys.taxonomy(mode),
    queryFn: () => fetchLogTaxonomy(mode),
    enabled: isAuthenticated(),
    staleTime: 30 * 60_000,
    retry: 1,
  });
}

/** GET /logs/preferences[?mode=] — quick tiles, category order and visibility, custom items. */
export async function fetchLogPreferences(mode?: string): Promise<LogPreferences> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>('/logs/preferences', {
    params: mode ? { mode } : undefined,
  });
  return logPreferencesSchema.parse(data.data);
}

export function useLogPreferences(mode?: string) {
  return useQuery({
    queryKey: logKeys.preferences(mode),
    queryFn: () => fetchLogPreferences(mode),
    enabled: isAuthenticated(),
    staleTime: 5 * 60_000,
    retry: 1,
  });
}

/** GET /logs/days/{date} — the day in the PUT body shape (empty when nothing is logged). */
export async function fetchLogDay(date: string): Promise<LogDay> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>(`/logs/days/${date}`);
  return logDaySchema.parse(data.data);
}

export function useLogDay(date: string) {
  return useQuery({
    queryKey: logKeys.day(date),
    queryFn: () => fetchLogDay(date),
    enabled: isAuthenticated() && date.length > 0,
    staleTime: 60_000,
    retry: 1,
  });
}

/** GET /logs/days?from&to — the days in range that have entries (≤ 366 days). */
export async function fetchLogDays(from: string, to: string): Promise<LogDaysRange> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>('/logs/days', { params: { from, to } });
  return logDaysRangeSchema.parse(data.data);
}

export function useLogDays(from: string, to: string, enabled = true) {
  return useQuery({
    queryKey: logKeys.range(from, to),
    queryFn: () => fetchLogDays(from, to),
    enabled: enabled && isAuthenticated() && from.length > 0 && to.length > 0,
    staleTime: 60_000,
    retry: 1,
  });
}

/**
 * PUT /logs/days/{date} — partial: each param sent replaces the stored one, `null` clears it, params not
 * sent are kept. Returns the whole day. `voiceParams` (`category.param`, B-N3-05) marks the params whose
 * value came from a confirmed voice-log suggestion (stored with `source: voice`).
 */
export async function saveLogDay(date: string, changes: LogDayChanges, voiceParams?: readonly string[]): Promise<LogDay> {
  const body = voiceParams?.length ? { categories: changes, voice_params: voiceParams } : { categories: changes };
  const { data } = await apiClient.put<ApiEnvelope<unknown>>(`/logs/days/${date}`, body);
  return logDaySchema.parse(data.data);
}

'use client';

import { useQuery } from '@tanstack/react-query';

import { type ApiEnvelope, apiClient } from '@/shared/api';
import { isAuthenticated } from '@/shared/session';

import type { Reminder, ReminderEnums, ReminderType } from '../model/types';
import { reminderEnumsSchema, reminderListSchema } from './schema';

/**
 * Query-key factory for reminders (CLAUDE.md §8). Mutations in
 * `features/manage-reminders` invalidate via this factory — never
 * hand-written arrays.
 */
export const reminderKeys = {
  all: ['reminder'] as const,
  list: (type?: ReminderType) => [...reminderKeys.all, 'list', type ?? 'all'] as const,
  enums: () => [...reminderKeys.all, 'enums'] as const,
};

/** GET /reminders — the user's reminders, newest first. */
export async function fetchReminders(type?: ReminderType): Promise<Reminder[]> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>('/reminders', {
    params: type ? { type } : undefined,
  });
  return reminderListSchema.parse(data.data ?? []);
}

/**
 * Reads the user's reminders (§8 — server state stays in TanStack Query).
 * Disabled until a token exists so it doesn't fire on public screens.
 * Reminder content is personal (§11): display only, never logged.
 */
export function useReminders(type?: ReminderType) {
  return useQuery({
    queryKey: reminderKeys.list(type),
    queryFn: () => fetchReminders(type),
    enabled: isAuthenticated(),
    retry: false,
  });
}

/** GET /reminders/enums — form options, labels localized by the backend. */
export async function fetchReminderEnums(): Promise<ReminderEnums> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>('/reminders/enums');
  return reminderEnumsSchema.parse(data.data ?? {});
}

/** Reads the type/recurrence options that drive the create form. */
export function useReminderEnums() {
  return useQuery({
    queryKey: reminderKeys.enums(),
    queryFn: fetchReminderEnums,
    enabled: isAuthenticated(),
    staleTime: 30 * 60_000,
    retry: false,
  });
}

'use client';

import { z } from 'zod';

import { type ApiEnvelope, apiClient } from '@/shared/api';

import type { LogCustomItem, LogPreferences } from '../model/log-v2';
import { logPreferencesSchema } from './log-v2-schema';

/**
 * Writes of the log preferences (B-N3-02, used by the customisation screen B-N3-04):
 * `PUT|DELETE /logs/preferences[?mode=]` and `POST /logs/custom-items`, `PATCH|DELETE /logs/custom-items/{id}`.
 * Plain request functions — the optimistic cache handling lives with the draft in `features/customize-log`.
 */

/** The PUT body: each list sent replaces the saved one, `null` resets it to the default, lists left out are kept. */
export interface LogPreferencesChanges {
  order?: string[] | null;
  hidden?: string[] | null;
  pinned?: string[] | null;
}

/** The categories that take custom items (POST /logs/custom-items `category`). */
export type LogCustomHost = 'custom' | 'symptoms' | 'mood' | 'activity' | 'appetite_energy' | 'urogenital' | 'skin_hair';

const modeParams = (mode?: string) => (mode ? { mode } : undefined);

const customItemSchema = z
  .object({ id: z.number(), code: z.string(), category: z.string(), param: z.string(), label: z.string() })
  .transform((i): LogCustomItem => i);

/** PUT /logs/preferences[?mode=] — returns the preferences in force. */
export async function saveLogPreferences(changes: LogPreferencesChanges, mode?: string): Promise<LogPreferences> {
  const { data } = await apiClient.put<ApiEnvelope<unknown>>('/logs/preferences', changes, { params: modeParams(mode) });
  return logPreferencesSchema.parse(data.data);
}

/** DELETE /logs/preferences[?mode=] — back to the mode's defaults (custom items are kept). */
export async function resetLogPreferences(mode?: string): Promise<LogPreferences> {
  const { data } = await apiClient.delete<ApiEnvelope<unknown>>('/logs/preferences', { params: modeParams(mode) });
  return logPreferencesSchema.parse(data.data);
}

/** POST /logs/custom-items — 422 on an empty / too long / duplicate label or past the 20-item cap. */
export async function addLogCustomItem(category: string, label: string): Promise<LogCustomItem> {
  const { data } = await apiClient.post<ApiEnvelope<unknown>>('/logs/custom-items', { category, label });
  return customItemSchema.parse(data.data);
}

/** PATCH /logs/custom-items/{id} — same label rules as POST. */
export async function renameLogCustomItem(id: number, label: string): Promise<LogCustomItem> {
  const { data } = await apiClient.patch<ApiEnvelope<unknown>>(`/logs/custom-items/${id}`, { label });
  return customItemSchema.parse(data.data);
}

/** DELETE /logs/custom-items/{id} — soft delete: logged days keep the item. */
export async function deleteLogCustomItem(id: number): Promise<void> {
  await apiClient.delete<ApiEnvelope<unknown>>(`/logs/custom-items/${id}`);
}

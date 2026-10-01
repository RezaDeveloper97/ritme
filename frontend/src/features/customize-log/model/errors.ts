import { ApiError, getApiErrorStatus } from '@/shared/api';

import type { LabelProblem } from './draft';

/** What a failed custom-item write shows, as a `logCustomize.errors.*` key. */
export type CustomItemError = LabelProblem | 'limit' | 'category' | 'throttled' | 'generic';

function fieldErrors(error: unknown): Record<string, unknown> {
  if (!(error instanceof ApiError)) return {};
  const body = error.response?.data;
  if (!body || typeof body !== 'object') return {};
  const errors = (body as { errors?: unknown }).errors;
  return errors && typeof errors === 'object' ? (errors as Record<string, unknown>) : {};
}

/**
 * Maps a 422 from POST / PATCH /logs/custom-items onto our own copy (the server's text is the generic
 * framework line): `custom_items` → the 20-item cap, `category` → pick a category, `label` → the label is
 * taken (empty and too long are caught before sending). 429 → slow down; anything else → generic.
 */
export function customItemError(error: unknown): CustomItemError {
  const status = getApiErrorStatus(error);
  if (status === 429) return 'throttled';
  if (status !== 422) return 'generic';
  const errors = fieldErrors(error);
  if ('custom_items' in errors) return 'limit';
  if ('category' in errors) return 'category';
  if ('label' in errors) return 'duplicate';
  return 'generic';
}

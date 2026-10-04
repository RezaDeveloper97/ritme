'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { z } from 'zod';

import { ApiError, type ApiEnvelope, apiClient, getApiErrorStatus } from '@/shared/api';
import { isAuthenticated } from '@/shared/session';

import {
  IVF_E2_UNITS,
  IVF_FOLLICLE_BINS,
  type IvfGrowthPoint,
  type IvfOvaryCounts,
  type IvfScan,
  type IvfScanInput,
  type IvfScansView,
} from '../model/scan';
import { ivfKeys } from './keys';
import { ivfCycleSchema } from './schema';

/*
 * `/api/v1/ivf/scans*` (CB-IVF-01) for the scan log (CB-IVF-04). Health data
 * (CLAUDE.md §11): never log a payload or a response.
 */

/** Query key of `GET /ivf/scans` — under `ivfKeys.all`, so every IVF write refreshes it. */
export const ivfScansKey = () => [...ivfKeys.all, 'scans'] as const;

const count = z.number().int().min(0).catch(0);
const decimalText = z
  .union([z.string(), z.number()])
  .nullable()
  .catch(null)
  .transform((v) => (v === null || String(v).trim() === '' ? null : String(v)));
const nullableInt = z.number().int().nullable().catch(null);

const ovarySchema = z
  .object({ lt_10: count, '10_14': count, '15_17': count, '18_plus': count })
  .partial()
  .nullable()
  .catch(null)
  .transform(
    (o): IvfOvaryCounts =>
      Object.fromEntries(IVF_FOLLICLE_BINS.map((bin) => [bin, o?.[bin] ?? 0])) as IvfOvaryCounts,
  );

const scanSchema = z
  .object({
    date: z.string().regex(/^\d{4}-\d{2}-\d{2}$/),
    stim_day: nullableInt,
    right: ovarySchema,
    left: ovarySchema,
    endometrium_mm: decimalText,
    e2: decimalText,
    e2_unit: z.enum(IVF_E2_UNITS).nullable().catch(null),
    notes: z.string().nullable().catch(null),
  })
  .transform(
    (d): IvfScan => ({
      date: d.date,
      stimDay: d.stim_day,
      right: d.right,
      left: d.left,
      endometriumMm: d.endometrium_mm,
      e2: d.e2,
      e2Unit: d.e2_unit,
      notes: d.notes,
    }),
  );

const growthSchema = z
  .object({ date: z.string(), stim_day: nullableInt, follicles_10_14: count, follicles_15_plus: count })
  .transform(
    (d): IvfGrowthPoint => ({ date: d.date, stimDay: d.stim_day, mid: d.follicles_10_14, lead: d.follicles_15_plus }),
  );

/** A malformed item is dropped, never the list. */
function listOf<T>(schema: z.ZodType<T, z.ZodTypeDef, unknown>) {
  return z
    .array(z.unknown())
    .catch([])
    .transform((items) =>
      items.flatMap((raw) => {
        const item = schema.safeParse(raw);
        return item.success ? [item.data] : [];
      }),
    );
}

/** `IvfScansView` (GET and every write) → the client shape. */
export const ivfScansSchema = z
  .object({
    cycle: z.unknown().optional(),
    scans: listOf(scanSchema),
    growth: listOf(growthSchema),
  })
  .transform((d): IvfScansView => {
    const cycle = ivfCycleSchema.nullable().catch(null).parse(d.cycle ?? null);
    const stim = z
      .object({ stim_started_on: z.string().nullable().catch(null) })
      .catch({ stim_started_on: null })
      .parse(d.cycle ?? {});
    return { cycle, stimStartedOn: stim.stim_started_on, scans: d.scans, growth: d.growth };
  });

async function readScans(): Promise<IvfScansView> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>('/ivf/scans');
  return ivfScansSchema.parse(data.data);
}

/** GET /ivf/scans — the open cycle's scans (oldest first) + the growth series. */
export function useIvfScans() {
  return useQuery({
    queryKey: ivfScansKey(),
    queryFn: readScans,
    enabled: isAuthenticated(),
    staleTime: 30_000,
    retry: 1,
  });
}

/** Put the answer of a write into the cache; the IVF home (latest scan) refetches. */
function useApplyScans() {
  const queryClient = useQueryClient();
  return (view: IvfScansView) => {
    queryClient.setQueryData(ivfScansKey(), view);
    void queryClient.invalidateQueries({ queryKey: ivfKeys.home() });
  };
}

/** PUT /ivf/scans/{date} — saves (replaces) the scan of a day. */
export function useSaveIvfScan() {
  const apply = useApplyScans();
  return useMutation<IvfScansView, unknown, IvfScanInput>({
    mutationFn: async (input) => {
      const { data } = await apiClient.put<ApiEnvelope<unknown>>(`/ivf/scans/${input.date}`, {
        right: input.right,
        left: input.left,
        endometrium_mm: input.endometriumMm,
        e2: input.e2,
        e2_unit: input.e2 === null ? null : input.e2Unit,
        notes: input.notes,
      });
      return ivfScansSchema.parse(data.data);
    },
    onSuccess: apply,
  });
}

/** First server validation message of a failed scan write (already in the request language), if any. */
export function ivfScanErrorMessage(error: unknown): string | undefined {
  if (getApiErrorStatus(error) !== 422 || !(error instanceof ApiError)) return undefined;
  const body = error.response?.data as ApiEnvelope<unknown> | undefined;
  const lists = body && typeof body === 'object' && body.errors ? Object.values(body.errors) : [];
  for (const list of lists) {
    const first = Array.isArray(list) ? list.find((m) => typeof m === 'string' && m.trim()) : undefined;
    if (first) return first;
  }
  return undefined;
}

/** DELETE /ivf/scans/{date} — removes the scan of a day. */
export function useDeleteIvfScan() {
  const apply = useApplyScans();
  return useMutation<IvfScansView, unknown, string>({
    mutationFn: async (date) => {
      const { data } = await apiClient.delete<ApiEnvelope<unknown>>(`/ivf/scans/${date}`);
      return ivfScansSchema.parse(data.data);
    },
    onSuccess: apply,
  });
}

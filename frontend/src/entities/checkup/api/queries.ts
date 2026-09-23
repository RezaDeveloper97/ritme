'use client';

import { useInfiniteQuery, useQuery } from '@tanstack/react-query';

import { type ApiEnvelope, apiClient } from '@/shared/api';
import { isAuthenticated } from '@/shared/session';

import { checkupAttachments } from '../model/attachments';
import type {
  CheckupDetail,
  CheckupHome,
  CheckupList,
  CheckupListFilter,
  CheckupNextPreview,
  CheckupRecordPage,
} from '../model/types';
import { type CheckupRecordFilters, checkupKeys } from './keys';
import {
  checkupDetailSchema,
  checkupHomeSchema,
  checkupListSchema,
  checkupNextPreviewSchema,
  checkupRecordPageSchema,
} from './schema';

/*
 * Reads for `/api/v1/checkups/*` (CLAUDE.md §8 — server state in TanStack
 * Query). Every hook is disabled until a token exists so public screens never
 * fire it. Personal health data (§11): never log what these return.
 */

/** GET /checkups (`?filter=action|done`; `all` sends no filter). */
export async function fetchCheckups(filter: CheckupListFilter = 'all'): Promise<CheckupList> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>('/checkups', {
    params: filter === 'all' ? undefined : { filter },
  });
  return checkupListSchema.parse(data.data ?? {});
}

export function useCheckups(filter: CheckupListFilter = 'all') {
  return useQuery({
    queryKey: checkupKeys.list(filter),
    queryFn: () => fetchCheckups(filter),
    enabled: isAuthenticated(),
    retry: false,
  });
}

/** GET /checkups/home — null when nothing applies. */
export async function fetchCheckupHome(): Promise<CheckupHome | null> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>('/checkups/home');
  return checkupHomeSchema.parse(data.data ?? null);
}

export function useCheckupHome(options: { enabled?: boolean } = {}) {
  return useQuery({
    queryKey: checkupKeys.home(),
    queryFn: fetchCheckupHome,
    // The card is hidden in pregnancy mode: the caller passes `enabled: false` there.
    enabled: isAuthenticated() && (options.enabled ?? true),
    retry: false,
  });
}

/** GET /checkups/{id}. */
export async function fetchCheckup(id: number): Promise<CheckupDetail> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>(`/checkups/${id}`);
  return checkupDetailSchema.parse(data.data);
}

export function useCheckup(id: number | null) {
  return useQuery({
    queryKey: checkupKeys.detail(id ?? 0),
    queryFn: () => fetchCheckup(id as number),
    enabled: isAuthenticated() && id !== null,
    retry: false,
  });
}

/** GET /checkups/records (`?filter=&type=&page=`), newest first. */
export async function fetchCheckupRecords(
  filters: CheckupRecordFilters = {},
  page = 1,
): Promise<CheckupRecordPage> {
  const params: Record<string, string | number> = { page };
  if (filters.filter && filters.filter !== 'all') params.filter = filters.filter;
  if (filters.type != null) params.type = filters.type;
  const { data } = await apiClient.get<ApiEnvelope<unknown>>('/checkups/records', { params });
  return checkupRecordPageSchema.parse(data.data ?? []);
}

/** The History timeline: pages until `lastPage`. */
export function useCheckupRecords(filters: CheckupRecordFilters = {}) {
  return useInfiniteQuery({
    queryKey: checkupKeys.records(filters),
    queryFn: ({ pageParam }) => fetchCheckupRecords(filters, pageParam),
    initialPageParam: 1,
    getNextPageParam: (last) => (last.page < last.lastPage ? last.page + 1 : undefined),
    enabled: isAuthenticated(),
    retry: false,
  });
}

/** GET /checkups/preview-next?type=&done_on= — the MarkDone «موعد بعدی» banner. */
export async function fetchCheckupNextPreview(
  typeId: number,
  doneOn: string,
): Promise<CheckupNextPreview> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>('/checkups/preview-next', {
    params: { type: typeId, done_on: doneOn },
  });
  return checkupNextPreviewSchema.parse(data.data ?? {});
}

export function useCheckupNextPreview(typeId: number | null, doneOn: string | null) {
  return useQuery({
    queryKey: checkupKeys.previewNext(typeId ?? 0, doneOn ?? ''),
    queryFn: () => fetchCheckupNextPreview(typeId as number, doneOn as string),
    enabled: isAuthenticated() && typeId !== null && !!doneOn,
    staleTime: 5 * 60_000,
    retry: false,
  });
}

/**
 * The on-device report of one record (null when none / storage unavailable).
 * Read from IndexedDB, never the network.
 */
export function useCheckupAttachment(recordId: number | null) {
  return useQuery({
    queryKey: checkupKeys.attachment(recordId ?? 0),
    queryFn: () => checkupAttachments.get(recordId as number),
    enabled: recordId !== null,
    staleTime: Infinity,
    retry: false,
  });
}

/** Ids of every record with a report on this device (History «با پیوست», PDF summary). */
export function useCheckupAttachmentIds() {
  return useQuery({
    queryKey: checkupKeys.attachmentsAll(),
    queryFn: async (): Promise<Set<number>> =>
      new Set((await checkupAttachments.list()).map((m) => Number(m.key)).filter(Number.isInteger)),
    staleTime: Infinity,
    retry: false,
  });
}

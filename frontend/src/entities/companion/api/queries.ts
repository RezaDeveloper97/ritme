'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { type ApiEnvelope, apiClient, getApiErrorStatus } from '@/shared/api';
import { isAuthenticated } from '@/shared/session';

import type {
  CompanionAuditEntry,
  CompanionGrants,
  CreateCompanionInput,
  CreatedCompanion,
  OwnerCompanion,
} from '../model/types';
import { auditSchema, createdCompanionSchema, ownerCompanionListSchema, ownerCompanionSchema } from './schema';

/*
 * Owner routes of `/api/v1/companions` (B-N4-02, Go only). Invite codes come
 * back only from create / renew: callers keep them in component state and
 * never log, persist or put them in a URL.
 */

/** Query-key factory for companions (CLAUDE.md §8). */
export const companionKeys = {
  all: ['companions'] as const,
  list: () => [...companionKeys.all, 'list'] as const,
  detail: (id: number) => [...companionKeys.all, 'detail', id] as const,
  audit: () => [...companionKeys.all, 'audit'] as const,
};

/** GET /companions — invited and active companions of the signed-in owner. */
export async function fetchCompanions(): Promise<OwnerCompanion[]> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>('/companions');
  return ownerCompanionListSchema.parse(data.data ?? []);
}

/** GET /companions/{id}. */
export async function fetchCompanion(id: number): Promise<OwnerCompanion> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>(`/companions/${id}`);
  return ownerCompanionSchema.parse(data.data);
}

/** GET /companions/audit — who read / wrote what and when (no payload). */
export async function fetchCompanionAudit(): Promise<CompanionAuditEntry[]> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>('/companions/audit', { params: { limit: 50 } });
  return auditSchema.parse(data.data ?? []);
}

/** POST /companions — a pending companion and its one-time code (+ SMS when a phone is given). 201. */
export async function createCompanion(input: CreateCompanionInput): Promise<CreatedCompanion> {
  const body: Record<string, unknown> = { type: input.type, grants: input.grants };
  if (input.displayName?.trim()) body.display_name = input.displayName.trim();
  if (input.phone?.trim()) body.phone = input.phone.trim();
  if (input.childIds?.length) body.child_ids = input.childIds;
  const { data } = await apiClient.post<ApiEnvelope<unknown>>('/companions', body);
  return createdCompanionSchema.parse(data.data);
}

export function useCompanions(enabled = true) {
  return useQuery({
    queryKey: companionKeys.list(),
    queryFn: fetchCompanions,
    enabled: enabled && isAuthenticated(),
    staleTime: 30_000,
  });
}

export function useCompanion(id: number) {
  const queryClient = useQueryClient();
  return useQuery({
    queryKey: companionKeys.detail(id),
    queryFn: () => fetchCompanion(id),
    enabled: Number.isFinite(id) && id > 0 && isAuthenticated(),
    // The list row is the same shape — paint it while the detail loads.
    placeholderData: () => queryClient.getQueryData<OwnerCompanion[]>(companionKeys.list())?.find((c) => c.id === id),
    retry: (count, error) => !isNotFound(error) && count < 2,
  });
}

export function useCompanionAudit(enabled = true) {
  return useQuery({
    queryKey: companionKeys.audit(),
    queryFn: fetchCompanionAudit,
    enabled: enabled && isAuthenticated(),
    staleTime: 60_000,
  });
}

function isNotFound(error: unknown): boolean {
  return getApiErrorStatus(error) === 404;
}

/** Writes the fresh row into the detail cache and refreshes the list + audit. */
function useCompanionWrite<V>(write: (vars: V) => Promise<OwnerCompanion>) {
  const queryClient = useQueryClient();
  return useMutation<OwnerCompanion, unknown, V>({
    mutationFn: write,
    onSuccess: (companion) => {
      queryClient.setQueryData(companionKeys.detail(companion.id), companion);
      void queryClient.invalidateQueries({ queryKey: companionKeys.list() });
      void queryClient.invalidateQueries({ queryKey: companionKeys.audit() });
    },
  });
}

/** PUT /companions/{id}/grants — replaces the access; takes effect on the companion's next request. */
export function useUpdateCompanionGrants(id: number) {
  return useCompanionWrite(async (grants: CompanionGrants) => {
    const { data } = await apiClient.put<ApiEnvelope<unknown>>(`/companions/${id}/grants`, { grants });
    return ownerCompanionSchema.parse(data.data);
  });
}

/** POST /companions — see {@link createCompanion}. */
export function useCreateCompanion() {
  const queryClient = useQueryClient();
  return useMutation<CreatedCompanion, unknown, CreateCompanionInput>({
    mutationFn: createCompanion,
    onSuccess: ({ companion }) => {
      queryClient.setQueryData(companionKeys.detail(companion.id), companion);
      void queryClient.invalidateQueries({ queryKey: companionKeys.list() });
      void queryClient.invalidateQueries({ queryKey: companionKeys.audit() });
    },
  });
}

/** POST /companions/{id}/renew — a fresh code for a pending invite (the old code stops working). */
export function useRenewCompanionInvite(id: number) {
  const queryClient = useQueryClient();
  return useMutation<CreatedCompanion, unknown, void>({
    mutationFn: async () => {
      const { data } = await apiClient.post<ApiEnvelope<unknown>>(`/companions/${id}/renew`, {});
      return createdCompanionSchema.parse(data.data);
    },
    onSuccess: () => {
      // The renew payload carries no name lookup — refetch rather than overwrite the detail.
      void queryClient.invalidateQueries({ queryKey: companionKeys.all });
    },
  });
}

/** DELETE /companions/{id} — access ends at once (pending or active). */
export function useRevokeCompanion(id: number) {
  const queryClient = useQueryClient();
  return useMutation<void, unknown, void>({
    mutationFn: async () => {
      await apiClient.delete(`/companions/${id}`);
    },
    onSuccess: () => {
      queryClient.removeQueries({ queryKey: companionKeys.detail(id) });
      queryClient.setQueryData<OwnerCompanion[]>(companionKeys.list(), (rows) => rows?.filter((c) => c.id !== id));
      void queryClient.invalidateQueries({ queryKey: companionKeys.list() });
      void queryClient.invalidateQueries({ queryKey: companionKeys.audit() });
    },
  });
}

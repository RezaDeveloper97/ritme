'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { type ApiEnvelope, apiClient } from '@/shared/api';
import { isAuthenticated } from '@/shared/session';

import type {
  TeenGrants,
  TeenParentCard,
  TeenParentInvite,
  TeenParentInviteInput,
  TeenParentLink,
  TeenProfileState,
  TeenToday,
} from '../model/types';
import { teenKeys } from './keys';
import {
  teenGrantsBody,
  teenLinkedSchema,
  teenParentCreatedSchema,
  teenParentLinkSchema,
  teenProfileStateSchema,
} from './schema';

/*
 * Mother sharing (CB-TEEN-03 on the CB-TEEN-01 API). The parent link is a
 * bloom companion of `type: parent` — invite, grants and renew go through
 * `/companions` with the three teen sections only, always view-only. Invite
 * codes live only in the caller's component state (never logged, stored or
 * put in a URL). Revoke is bloom's `useRevokeCompanion` (entities/companion).
 */

type Created = { link: TeenParentLink; invite: TeenParentInvite };

/** Puts a fresh / changed link into the cached teen home (`parent_links`). */
function upsertLink(today: TeenToday | undefined, link: TeenParentLink): TeenToday | undefined {
  if (!today) return today;
  const exists = today.parentLinks.some((l) => l.id === link.id);
  const parentLinks = exists ? today.parentLinks.map((l) => (l.id === link.id ? link : l)) : [...today.parentLinks, link];
  return { ...today, parentLinks };
}

/** POST /companions {type: parent} — phone required; the code is SMSed to the parent. 201. */
export function useInviteParent() {
  const queryClient = useQueryClient();
  return useMutation<Created, unknown, TeenParentInviteInput>({
    mutationFn: async (input) => {
      const body: Record<string, unknown> = { type: 'parent', phone: input.phone, grants: teenGrantsBody(input.grants) };
      if (input.displayName?.trim()) body.display_name = input.displayName.trim();
      const { data } = await apiClient.post<ApiEnvelope<unknown>>('/companions', body);
      return teenParentCreatedSchema.parse(data.data);
    },
    onSuccess: ({ link }) => {
      queryClient.setQueryData<TeenToday>(teenKeys.today(), (today) => upsertLink(today, link));
      void queryClient.invalidateQueries({ queryKey: teenKeys.today() });
    },
  });
}

/** POST /companions/{id}/renew — a fresh code for a pending invite (the old one stops working). */
export function useRenewParentInvite() {
  return useMutation<Created, unknown, number>({
    mutationFn: async (id) => {
      const { data } = await apiClient.post<ApiEnvelope<unknown>>(`/companions/${id}/renew`, {});
      return teenParentCreatedSchema.parse(data.data);
    },
  });
}

/**
 * PUT /companions/{id}/grants — what the parent may see; takes effect on her
 * next read. Optimistic: the switch moves at once and rolls back on failure.
 */
export function useUpdateParentGrants() {
  const queryClient = useQueryClient();
  return useMutation<TeenParentLink, unknown, { id: number; grants: TeenGrants }, { previous: TeenToday | undefined }>({
    mutationFn: async ({ id, grants }) => {
      const { data } = await apiClient.put<ApiEnvelope<unknown>>(`/companions/${id}/grants`, {
        grants: teenGrantsBody(grants),
      });
      return teenParentLinkSchema.parse(data.data);
    },
    onMutate: async ({ id, grants }) => {
      const key = teenKeys.today();
      await queryClient.cancelQueries({ queryKey: key });
      const previous = queryClient.getQueryData<TeenToday>(key);
      const link = previous?.parentLinks.find((l) => l.id === id);
      if (link) queryClient.setQueryData<TeenToday>(key, upsertLink(previous, { ...link, grants }));
      return { previous };
    },
    onError: (_error, _vars, context) => {
      if (context?.previous) queryClient.setQueryData(teenKeys.today(), context.previous);
    },
    onSuccess: (link) => {
      queryClient.setQueryData<TeenToday>(teenKeys.today(), (today) => upsertLink(today, link));
    },
  });
}

/** PUT /teen/parent-note — the note a parent sees only while «علائم و یادداشت‌ها» is shared. Empty clears it. */
export function useSaveParentNote() {
  const queryClient = useQueryClient();
  return useMutation<TeenProfileState, unknown, string>({
    mutationFn: async (note) => {
      const trimmed = note.trim();
      const { data } = await apiClient.put<ApiEnvelope<unknown>>('/teen/parent-note', { note: trimmed || null });
      return teenProfileStateSchema.parse(data.data);
    },
    onSuccess: (saved) => {
      queryClient.setQueryData(teenKeys.profile(), saved);
      void queryClient.invalidateQueries({ queryKey: teenKeys.today() });
    },
  });
}

/** Drops a revoked link from the cached teen home (after bloom's DELETE /companions/{id}). */
export function useForgetParentLink() {
  const queryClient = useQueryClient();
  return (id: number) => {
    queryClient.setQueryData<TeenToday>(teenKeys.today(), (today) =>
      today ? { ...today, parentLinks: today.parentLinks.filter((l) => l.id !== id) } : today,
    );
    void queryClient.invalidateQueries({ queryKey: teenKeys.today() });
  };
}

export async function fetchTeenLinked(): Promise<TeenParentCard[]> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>('/teen/linked');
  return teenLinkedSchema.parse(data.data ?? []);
}

/** GET /teen/linked — the read-only cards of the teens the signed-in user is a parent of (`[]` for most). */
export function useTeenLinked() {
  return useQuery({
    queryKey: teenKeys.linked(),
    queryFn: fetchTeenLinked,
    enabled: isAuthenticated(),
    staleTime: 60_000,
    retry: 1,
  });
}

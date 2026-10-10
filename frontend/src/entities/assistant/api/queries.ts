'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { type ApiEnvelope, apiClient } from '@/shared/api';
import { isAuthenticated } from '@/shared/session';

import { assistantKeys } from './keys';
import {
  consentSchema,
  conversationListSchema,
  conversationPageSchema,
  departmentPageSchema,
  handoffSchema,
  hubSchema,
  profileSchema,
  summaryPageSchema,
} from './schema';

type Params = Record<string, string | number | boolean | undefined | null>;

async function get<T>(path: string, schema: { parse: (v: unknown) => T }, params?: Params): Promise<T> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>(path, { params });
  return schema.parse(data.data);
}

const enabled = () => isAuthenticated();

/** GET /assistant — departments, quota, recent conversations, profile chips. */
export const useHub = () => useQuery({ queryKey: assistantKeys.hub(), queryFn: () => get('/assistant', hubSchema), enabled: enabled() });

/** GET /assistant/departments/{code}. */
export const useDepartment = (code: string) =>
  useQuery({
    queryKey: assistantKeys.department(code),
    queryFn: () => get(`/assistant/departments/${encodeURIComponent(code)}`, departmentPageSchema),
    enabled: enabled() && code !== '',
    retry: 1,
  });

/** GET /assistant/profile. */
export const useAssistantProfile = () =>
  useQuery({
    queryKey: assistantKeys.profile(),
    queryFn: async () => (await get('/assistant/profile', { parse: (v) => profileSchema.parse((v as { profile: unknown }).profile) })),
    enabled: enabled(),
  });

/** PUT /assistant/profile {shared: [key…]}. */
export function useSaveProfile() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (shared: string[]) => {
      const { data } = await apiClient.put<ApiEnvelope<unknown>>('/assistant/profile', { shared });
      return profileSchema.parse((data.data as { profile: unknown }).profile);
    },
    onSuccess: (profile) => {
      client.setQueryData(assistantKeys.profile(), profile);
      void client.invalidateQueries({ queryKey: assistantKeys.hub() });
    },
  });
}

/** GET /assistant/conversations — the user's whole history. */
export const useConversations = () =>
  useQuery({ queryKey: assistantKeys.conversations(), queryFn: () => get('/assistant/conversations', conversationListSchema), enabled: enabled() });

/** GET /assistant/conversations/{id}. */
export const useConversation = (id: number) =>
  useQuery({
    queryKey: assistantKeys.conversation(id),
    queryFn: () => get(`/assistant/conversations/${id}`, conversationPageSchema),
    enabled: enabled() && id > 0,
    retry: 1,
  });

/** POST /assistant/conversations {department} → the new conversation id. */
export function useStartConversation() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (department: string) => {
      const { data } = await apiClient.post<ApiEnvelope<unknown>>('/assistant/conversations', { department });
      return conversationPageSchema.parse(data.data).conversation.id;
    },
    onSuccess: () => void client.invalidateQueries({ queryKey: assistantKeys.all }),
  });
}

/** DELETE /assistant/conversations/{id} (hard delete) and DELETE /assistant/conversations (all). */
export function useDeleteConversations() {
  const client = useQueryClient();
  const done = () => void client.invalidateQueries({ queryKey: assistantKeys.all });
  return {
    one: useMutation({ mutationFn: (id: number) => apiClient.delete(`/assistant/conversations/${id}`), onSuccess: done }),
    all: useMutation({ mutationFn: () => apiClient.delete('/assistant/conversations'), onSuccess: done }),
  };
}

/** POST /assistant/conversations/{id}/summary — writes (or returns) the summary document. */
export function usePrepareSummary(id: number) {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async () => {
      const { data } = await apiClient.post<ApiEnvelope<unknown>>(`/assistant/conversations/${id}/summary`, {}, { timeoutMs: 60_000 });
      return summaryPageSchema.parse(data.data);
    },
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: assistantKeys.conversation(id) });
    },
  });
}

/** POST /assistant/conversations/{id}/handoff {booking_id}. */
export function useHandoff(id: number) {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (bookingId: number) => {
      const { data } = await apiClient.post<ApiEnvelope<unknown>>(`/assistant/conversations/${id}/handoff`, { booking_id: bookingId });
      return handoffSchema.parse((data.data as { handoff: unknown }).handoff);
    },
    onSuccess: () => void client.invalidateQueries({ queryKey: assistantKeys.conversation(id) }),
  });
}

/** POST /assistant/photos (multipart "photo") → the photo id for the next message. */
export async function uploadPhoto(file: File): Promise<number> {
  const form = new FormData();
  form.append('photo', file);
  const { data } = await apiClient.post<ApiEnvelope<{ photo: { id: number } }>>('/assistant/photos', form, { timeoutMs: 60_000 });
  return Number(data.data?.photo.id);
}

/** GET /consents/{code} — the text in force and whether it is granted (`ai_assistant`, `assistant_profile`). */
export const useAssistantConsent = (code: string, on = true) =>
  useQuery({ queryKey: assistantKeys.consent(code), queryFn: () => get(`/consents/${code}`, consentSchema), enabled: enabled() && on, retry: 1 });

/** PUT /consents/{code} {granted, version}. */
export function useGrantConsent(code: string) {
  const client = useQueryClient();
  return useMutation({
    mutationFn: async (version: number) => {
      const { data } = await apiClient.put<ApiEnvelope<unknown>>(`/consents/${code}`, { granted: true, version });
      return consentSchema.parse(data.data);
    },
    onSuccess: (c) => {
      client.setQueryData(assistantKeys.consent(code), c);
      void client.invalidateQueries({ queryKey: assistantKeys.profile() });
      void client.invalidateQueries({ queryKey: assistantKeys.hub() });
    },
    onError: () => void client.invalidateQueries({ queryKey: assistantKeys.consent(code) }),
  });
}

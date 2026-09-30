'use client';

import { useMutation, useQuery } from '@tanstack/react-query';

import { type ApiEnvelope, apiClient } from '@/shared/api';

import { boxesSchema, type SupportBox } from '../model/support';

/** Query-key factory (CLAUDE.md §8); the copy comes localized, so keyed by locale. */
export const supportKeys = {
  all: ['support'] as const,
  boxes: (group: 'help' | 'support', locale: string) => [...supportKeys.all, group, locale] as const,
};

/** GET /info-pages/{help|support} — public admin content, cached for an hour. */
export function useSupportBoxes(group: 'help' | 'support', locale: string) {
  return useQuery({
    queryKey: supportKeys.boxes(group, locale),
    queryFn: async (): Promise<SupportBox[]> => {
      const { data } = await apiClient.get<ApiEnvelope<unknown>>(`/info-pages/${group}`);
      return boxesSchema.parse(data.data);
    },
    staleTime: 60 * 60_000,
    retry: 1,
  });
}

export interface ReportInput {
  message: string;
  /** `data:image/…;base64,…` of the chosen screenshot. */
  screenshot?: string;
}

/** POST /support/reports — the screenshot travels as a data URL inside the JSON body. */
export function useSendReport() {
  return useMutation<void, unknown, ReportInput>({
    mutationFn: async ({ message, screenshot }) => {
      await apiClient.post('/support/reports', {
        message: message.trim(),
        app_version: process.env.NEXT_PUBLIC_APP_VERSION ?? undefined,
        ...(screenshot ? { screenshot } : {}),
      });
    },
  });
}

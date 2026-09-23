'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { z } from 'zod';

import { api, createResource } from '@/shared/api';
import { contentLanguageKeys } from '@/shared/i18n';

/** `Language` — a row of the content-language registry (admin-api.md §11, super only). */
export const languageSchema = z.object({
  id: z.number(),
  code: z.string(),
  name: z.string(),
  english_name: z.string(),
  direction: z.string(),
  is_active: z.boolean(),
  is_default: z.boolean(),
  sort_order: z.number(),
});
export type Language = z.infer<typeof languageSchema>;

const optionsSchema = z.object({
  directions: z.array(z.string()),
  sources: z.array(z.object({ code: z.string(), name: z.string() })),
  default_code: z.string(),
  next_sort_order: z.number(),
});
export type LanguageOptions = z.infer<typeof optionsSchema>;

export const languagesApi = createResource({
  key: 'languages',
  path: '/languages',
  list: z.object({ items: z.array(languageSchema), default_code: z.string() }),
  detail: z.object({ language: languageSchema }),
  options: optionsSchema,
  // Every translatable form reads the public registry: adding or toggling a language changes them.
  alsoInvalidate: [contentLanguageKeys.all, ['language-translations']],
});

/** Store / regenerate answer: what was copied for the language. */
export const provisionSchema = z
  .object({
    source: z.string().optional(),
    provisioned: z
      .object({ messages: z.number(), lang_files: z.number(), smart_messages: z.number() })
      .partial()
      .optional(),
  })
  .passthrough();

// ── Translation editor: GET/PUT /languages/:id/translations ──

const translationsSchema = z.object({
  language: languageSchema,
  namespaces: z.array(z.string()),
  namespace: z.string(),
  rows: z.array(z.object({ key: z.string(), value: z.string().nullable(), reference: z.string().nullable() })),
  default_code: z.string(),
  default_name: z.string(),
  is_default_locale: z.boolean(),
});
export type TranslationsPage = z.infer<typeof translationsSchema>;

export const translationKeys = {
  all: ['language-translations'] as const,
  page: (id: number, namespace: string) => [...translationKeys.all, id, namespace] as const,
};

export function useTranslationsPage(id: number, namespace: string) {
  return useQuery({
    queryKey: translationKeys.page(id, namespace),
    queryFn: ({ signal }) =>
      api.get(`/languages/${id}/translations`, { query: { namespace }, schema: translationsSchema, signal }),
  });
}

export function useSaveTranslations(id: number) {
  const client = useQueryClient();
  return useMutation({
    mutationFn: (body: { namespace: string; rows: { key: string; value: string }[] }) =>
      api.put(`/languages/${id}/translations`, body, {
        schema: z.object({ namespace: z.string(), saved: z.number() }).passthrough(),
      }),
    onSuccess: (_data, body) => client.invalidateQueries({ queryKey: translationKeys.page(id, body.namespace) }),
  });
}

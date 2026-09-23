'use client';

import { useQuery } from '@tanstack/react-query';
import { z } from 'zod';

import { api } from '@/shared/api';

/**
 * The CONTENT languages — rows of the backend `languages` table, served by the
 * public GET /api/v1/languages. Every translatable admin field renders one input
 * per entry; only `is_default` is required (frontend/CLAUDE.md §6.3).
 */
export const contentLanguageSchema = z.object({
  code: z.string(),
  name: z.string(),
  english_name: z.string().optional().default(''),
  direction: z.enum(['rtl', 'ltr']).catch('ltr'),
  is_default: z.boolean(),
});
export type ContentLanguage = z.infer<typeof contentLanguageSchema>;

const languagesSchema = z.object({
  default: z.string(),
  languages: z.array(contentLanguageSchema),
});

export const contentLanguageKeys = {
  all: ['content-languages'] as const,
};

export function useContentLanguages() {
  return useQuery({
    queryKey: contentLanguageKeys.all,
    queryFn: ({ signal }) => api.get('/languages', { api: 'public', schema: languagesSchema, signal }),
    staleTime: 5 * 60_000,
  });
}

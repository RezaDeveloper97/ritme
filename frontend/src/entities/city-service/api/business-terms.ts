import { useQuery } from '@tanstack/react-query';
import { z } from 'zod';

import { type ApiEnvelope, apiClient } from '@/shared/api';
import { env } from '@/shared/config';

import { cityServiceKeys } from './keys';

/**
 * The cooperation terms for businesses: the admin-editable `business_terms` info-sections group, read from the public
 * GET /info-pages/business_terms (no login). Empty (or unreachable) means "show the neutral placeholder".
 */
export interface BusinessTerm {
  id: number;
  heading: string;
  body: string;
}

const termsSchema = z.object({
  sections: z.array(z.object({ id: z.number(), heading: z.string(), body: z.string() })),
});

function toTerms(data: unknown): BusinessTerm[] {
  const parsed = termsSchema.safeParse(data);
  if (!parsed.success) return [];
  return parsed.data.sections
    .map((s) => ({ id: s.id, heading: s.heading.trim(), body: s.body.trim() }))
    .filter((s) => s.heading !== '' || s.body !== '');
}

/** From a server component (landing SSR): `[]` when the API is unreachable or the group has no published box. */
export async function fetchBusinessTerms(locale: string): Promise<BusinessTerm[]> {
  try {
    const url = `${env.apiBaseUrl.replace(/\/+$/, '')}/info-pages/business_terms`;
    const res = await fetch(url, {
      headers: { Accept: 'application/json', 'Accept-Language': locale },
      next: { revalidate: 300 },
    });
    if (!res.ok) return [];
    const body = (await res.json()) as { data?: unknown };
    return toTerms(body.data);
  } catch {
    return [];
  }
}

/** Client read for the join flow (keyed by locale; the copy comes localized). */
export function useBusinessTerms(locale: string) {
  return useQuery({
    queryKey: cityServiceKeys.businessTerms(locale),
    queryFn: async (): Promise<BusinessTerm[]> => {
      const { data } = await apiClient.get<ApiEnvelope<unknown>>('/info-pages/business_terms');
      return toTerms(data.data);
    },
    staleTime: 10 * 60_000,
    retry: 1,
  });
}

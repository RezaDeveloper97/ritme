'use client';

import { useQuery } from '@tanstack/react-query';
import { useEffect, useState } from 'react';
import { z } from 'zod';

import { ApiError, buildUrl, createResource, pagedSchema } from '@/shared/api';
import { ADMIN_API_BASE_URL } from '@/shared/config';

export type SupportReportStatus = 'open' | 'resolved';

const reportUserSchema = z.object({ id: z.number(), name: z.string().nullable(), mobile: z.string().nullable() });

/** GET /support-reports row — a preview only, never the full message (admin-api.md §14). */
export const supportReportRowSchema = z.object({
  id: z.number(),
  user: reportUserSchema,
  preview: z.string(),
  has_screenshot: z.boolean(),
  app_version: z.string().nullable(),
  status: z.string(),
  created_at: z.string().nullable(),
});
export type SupportReportRow = z.infer<typeof supportReportRowSchema>;

export const supportReportSchema = z.object({
  id: z.number(),
  user: reportUserSchema,
  message: z.string(),
  has_screenshot: z.boolean(),
  app_version: z.string().nullable(),
  user_agent: z.string().nullable(),
  status: z.string(),
  created_at: z.string().nullable(),
  updated_at: z.string().nullable(),
});
export type SupportReport = z.infer<typeof supportReportSchema>;

export const supportReportsApi = createResource({
  key: 'support-reports',
  path: '/support-reports',
  list: pagedSchema(supportReportRowSchema, {
    filters: z.object({ status: z.string() }).partial().optional(),
    counts: z.object({ open: z.number(), resolved: z.number() }).optional(),
  }),
  detail: z.object({ support_report: supportReportSchema }),
});

/**
 * The report screenshot as an object URL. It lives on private storage and is only streamed to a signed-in
 * admin (session cookie), so it is fetched with credentials instead of being put in an <img src> directly.
 */
export function useScreenshotUrl(id: number, enabled: boolean) {
  const query = useQuery({
    queryKey: [...supportReportsApi.keys.detail(id), 'screenshot'],
    enabled,
    staleTime: Infinity,
    gcTime: 60_000,
    queryFn: async ({ signal }) => {
      const res = await fetch(buildUrl(ADMIN_API_BASE_URL, `/support-reports/${id}/screenshot`), {
        credentials: 'include',
        signal,
      });
      if (!res.ok) throw new ApiError({ status: res.status, code: res.status === 404 ? 'not_found' : 'http_error' });
      return res.blob();
    },
  });
  const [url, setUrl] = useState<string | null>(null);
  useEffect(() => {
    if (!query.data) return undefined;
    const next = URL.createObjectURL(query.data);
    setUrl(next);
    return () => {
      URL.revokeObjectURL(next);
      setUrl(null);
    };
  }, [query.data]);
  return { url, isPending: enabled && query.isPending, error: query.error, refetch: query.refetch };
}

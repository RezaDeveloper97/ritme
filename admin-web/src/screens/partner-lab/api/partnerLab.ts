'use client';

import { useMutation, useQueryClient } from '@tanstack/react-query';
import { z } from 'zod';

import { api, createResource, pagedSchema, translationsSchema } from '@/shared/api';

/** Home lab / screening catalog and request queue (admin-api.md §26, D-94; B-N10-04). */

export const labServiceSchema = z.object({
  id: z.number(),
  code: z.string(),
  kind: z.string(),
  title: translationsSchema,
  description: translationsSchema,
  partner_name: z.string(),
  partner_code: z.string(),
  is_active: z.boolean(),
  sort_order: z.number(),
});
export type LabServiceRow = z.infer<typeof labServiceSchema>;

export const labServicesApi = createResource({
  key: 'partner-lab-services',
  path: '/partners/lab-services',
  list: z.object({
    items: z.array(labServiceSchema),
    kinds: z.array(z.string()).catch([]),
    partners: z.object({ codes: z.array(z.string()) }).catch({ codes: ['fake'] }),
  }),
  detail: labServiceSchema,
});

export const labRequestSchema = z.object({
  id: z.number(),
  reference: z.string(),
  user_id: z.number(),
  user_mobile: z.string().nullable(),
  status: z.string(),
  service: z.string(),
  service_code: z.string(),
  partner_name: z.string(),
  preferred_date: z.string(),
  preferred_window: z.string(),
  consent_import: z.boolean(),
  imported: z.boolean(),
  purged: z.boolean(),
  manual_status: z.boolean(),
  created_at: z.string().nullable(),
});
export type LabRequestRow = z.infer<typeof labRequestSchema>;

export const labRequestsApi = createResource({
  key: 'partner-lab-requests',
  path: '/partners/lab-requests',
  list: pagedSchema(labRequestSchema, { statuses: z.array(z.string()).optional() }),
  detail: z.unknown(),
});

/** POST /partners/lab-requests/:id/status — the console plays the fake partner. */
export function useSetLabStatus() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: ({ id, status }: { id: number; status: string }) => api.post(`/partners/lab-requests/${id}/status`, { status }),
    onSuccess: () => client.invalidateQueries({ queryKey: labRequestsApi.keys.all }),
  });
}

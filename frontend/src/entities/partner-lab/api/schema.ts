import { z } from 'zod';

/*
 * Parsers of /api/v1/partners/* (bloom B-N10-04, backend-go/api/openapi.yaml). Snake_case = wire contract. The
 * catalog is admin content; the request carries the user's own details only until the 30-day purge.
 */

export const LAB_WINDOWS = ['morning', 'noon', 'evening'] as const;
export type LabWindow = (typeof LAB_WINDOWS)[number];

export const LAB_STATUSES = ['requested', 'accepted', 'sampled', 'result_ready', 'rejected', 'cancelled'] as const;
export type LabStatus = (typeof LAB_STATUSES)[number];

export const labServiceSchema = z.object({
  id: z.number(),
  code: z.string(),
  kind: z.string(),
  title: z.string(),
  description: z.string().catch(''),
  partner_name: z.string(),
});
export type LabService = z.infer<typeof labServiceSchema>;

export const labServicesSchema = z.object({
  services: z.array(labServiceSchema),
  max_open: z.number().catch(3),
  max_days_ahead: z.number().catch(60),
  purge_days: z.number().catch(30),
});

export const labRequestSchema = z.object({
  id: z.number(),
  reference: z.string(),
  status: z.enum(LAB_STATUSES).catch('requested'),
  service: z.object({ id: z.number(), kind: z.string(), title: z.string(), partner_name: z.string() }),
  preferred_date: z.string(),
  preferred_window: z.enum(LAB_WINDOWS).catch('morning'),
  consent_import: z.boolean().catch(false),
  imported: z.boolean().catch(false),
  lab_id: z.number().nullable().catch(null),
  can_cancel: z.boolean().catch(false),
  can_import: z.boolean().catch(false),
  purged: z.boolean().catch(false),
  created_at: z.string().nullable().catch(null),
  details: z
    .object({
      full_name: z.string().catch(''),
      phone: z.string().catch(''),
      city: z.string().catch(''),
      address: z.string().catch(''),
      note: z.string().catch(''),
    })
    .nullable()
    .catch(null)
    .optional(),
});
export type LabRequest = z.infer<typeof labRequestSchema>;

export const labRequestsSchema = z.object({ requests: z.array(labRequestSchema) });

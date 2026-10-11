import { z } from 'zod';

import { createResource } from '@/shared/api';

/** Insurance catalog: insurers and contracted centres (capability content), admin-api.md §27, CB-INS-02. */

export const insurerSchema = z.object({
  id: z.number(),
  code: z.string(),
  name: z.string(),
  website: z.string().nullable(),
  phone: z.string().nullable(),
  is_active: z.boolean(),
  sort_order: z.number(),
});
export type Insurer = z.infer<typeof insurerSchema>;

export const centreSchema = z.object({
  id: z.number(),
  insurer_id: z.number().nullable(),
  insurer_name: z.string().nullable(),
  name: z.string(),
  type: z.string(),
  services: z.string().nullable(),
  city: z.string().nullable(),
  address: z.string().nullable(),
  phone: z.string().nullable(),
  lat: z.string().nullable(),
  lng: z.string().nullable(),
  is_direct: z.boolean(),
  is_active: z.boolean(),
  sort_order: z.number(),
});
export type Centre = z.infer<typeof centreSchema>;

/** Fallback until the list loads; the API returns the authoritative `types`. */
export const CENTRE_TYPES = ['gyn', 'lab', 'imaging', 'hospital', 'pharmacy', 'pediatric', 'other'] as const;

export const insurersApi = createResource({
  key: 'insurance-insurers',
  path: '/insurance/insurers',
  list: z.object({ items: z.array(insurerSchema) }),
  detail: z.unknown(),
  alsoInvalidate: [['insurance-centres']],
});

export const centresApi = createResource({
  key: 'insurance-centres',
  path: '/insurance/centres',
  list: z.object({ items: z.array(centreSchema), types: z.array(z.string()).optional() }),
  detail: z.unknown(),
});

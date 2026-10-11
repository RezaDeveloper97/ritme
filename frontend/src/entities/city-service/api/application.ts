'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { z } from 'zod';

import { type ApiEnvelope, apiClient } from '@/shared/api';
import { isAuthenticated } from '@/shared/session';

/*
 * CB-DIR-10: the business join flow's API — /api/v1/city-services/applications (CB-DIR-04, docs/canvas-build/directory.md §4).
 * Snake_case = wire contract. Documents are PRIVATE files: their `url` is a 5-minute signed link, shown only through
 * the «view» action and never stored. Contact details and addresses go to the API only (no storage, no analytics).
 */

export const APPLICATION_STATUSES = ['draft', 'submitted', 'in_review', 'needs_info', 'approved', 'rejected'] as const;
export type ApplicationStatus = (typeof APPLICATION_STATUSES)[number];

export const DOCUMENT_SLOTS = ['licence', 'instructor_cert', 'health_cert'] as const;
export type DocumentSlot = (typeof DOCUMENT_SLOTS)[number];

export const JOIN_BOOKING_MODES = ['online', 'request', 'phone'] as const;
export type JoinBookingMode = (typeof JOIN_BOOKING_MODES)[number];

/** Catalog codes (seeded, directory.md §3) — labels live in the `cityJoin` messages. */
export const JOIN_AGE_RANGES = ['pregnancy', 'm0_6', 'm6_12', 'y1_2', 'y2_4', 'y4_6'] as const;
export const JOIN_AMENITIES = [
  'nursing_room',
  'changing_table',
  'female_instructor',
  'stroller_parking',
  'parking',
  'elevator',
  'wheelchair_access',
] as const;
export const JOIN_CATEGORY_ORDER = [
  'class',
  'pool',
  'playhouse',
  'music_art',
  'postpartum_fitness',
  'baby_massage',
  'parenting_workshop',
  'other',
] as const;

const text = z.string().nullable().catch(null);

const fileSchema = z.object({
  id: z.number(),
  mime: z.string().catch(''),
  size_bytes: z.number().catch(0),
  url: z.string(),
  url_expires_at: text,
});
export type JoinFile = z.infer<typeof fileSchema>;

const hoursSchema = z.object({ day: z.number(), open: z.string(), close: z.string() });
export type JoinHours = z.infer<typeof hoursSchema>;

const int = z.number().nullable().catch(null);
export const offerSchema = z.object({
  kind: z.enum(['session', 'package']).catch('session'),
  name: z.string().catch(''),
  description: text.optional(),
  format: z.enum(['group', 'private']).nullable().catch(null).optional(),
  duration_minutes: int.optional(),
  age_min_months: int.optional(),
  age_max_months: int.optional(),
  capacity: int.optional(),
  price_toman: int.optional(),
  package_sessions: int.optional(),
  package_validity_days: int.optional(),
});
export type JoinOffer = z.infer<typeof offerSchema>;

const docs = z.object({
  licence: fileSchema.nullable().catch(null),
  instructor_cert: fileSchema.nullable().catch(null),
  health_cert: fileSchema.nullable().catch(null),
});

export const applicationSchema = z.object({
  id: z.number(),
  status: z.enum(APPLICATION_STATUSES).catch('draft'),
  step: z.number().catch(1),
  tracking_code: text,
  editable: z.boolean().catch(false),
  can_add_photos: z.boolean().catch(false),
  business_name: text,
  category_slugs: z.array(z.string()).catch([]),
  city_id: z.number().nullable().catch(null),
  district: text,
  contact_name: text,
  contact_phone: text,
  owner_confirmed: z.boolean().catch(false),
  lat: z.number().nullable().catch(null),
  lng: z.number().nullable().catch(null),
  address: text,
  about: text,
  age_ranges: z.array(z.string()).catch([]),
  amenities: z.array(z.string()).catch([]),
  weekly_hours: z.array(hoursSchema).catch([]),
  photos: z.array(fileSchema).catch([]),
  services: z.array(offerSchema).catch([]),
  booking_mode: z.enum(JOIN_BOOKING_MODES).nullable().catch(null),
  terms_accepted_at: text,
  documents: docs.catch({ licence: null, instructor_cert: null, health_cert: null }),
  requirements: z
    .object({ min_photos: z.number(), max_photos: z.number(), health_cert_category: z.string() })
    .catch({ min_photos: 4, max_photos: 20, health_cert_category: 'pool' }),
  submitted_at: text,
  reviewed_at: text,
  review_note: text,
});
export type JoinApplication = z.infer<typeof applicationSchema>;

const summarySchema = z.object({
  id: z.number(),
  status: z.enum(APPLICATION_STATUSES).catch('draft'),
  step: z.number().catch(1),
  tracking_code: text,
  business_name: text,
  editable: z.boolean().catch(false),
  submitted_at: text,
});
export type JoinSummary = z.infer<typeof summarySchema>;

export const TIMELINE_KEYS = ['submitted', 'documents_review', 'expert_contact', 'published'] as const;
export type TimelineKey = (typeof TIMELINE_KEYS)[number];
const statusSchema = z.object({
  id: z.number(),
  status: z.enum(APPLICATION_STATUSES).catch('draft'),
  tracking_code: text,
  business_name: text,
  submitted_at: text,
  reviewed_at: text,
  review_note: text,
  timeline: z
    .array(z.object({ key: z.enum(TIMELINE_KEYS), state: z.enum(['done', 'current', 'pending', 'failed']).catch('pending'), at: text }))
    .catch([]),
});
export type JoinStatus = z.infer<typeof statusSchema>;

export const joinKeys = {
  all: ['city-service', 'applications'] as const,
  list: () => [...joinKeys.all, 'list'] as const,
  one: (id: number) => [...joinKeys.all, 'one', id] as const,
  status: (id: number) => [...joinKeys.all, 'status', id] as const,
};

const BASE = '/city-services/applications';
const UPLOAD_TIMEOUT_MS = 120_000;
const oneApp = z.object({ application: applicationSchema });

/** GET /applications — my applications, newest first (≤ 10). */
export const useJoinApplications = () =>
  useQuery({
    queryKey: joinKeys.list(),
    queryFn: async () =>
      z.object({ applications: z.array(summarySchema) }).parse((await apiClient.get<ApiEnvelope<unknown>>(BASE)).data.data).applications,
    enabled: isAuthenticated(),
    staleTime: 0,
  });

/** GET /applications/:id — the full application (photos public, documents with signed links). */
export const useJoinApplication = (id: number | null) =>
  useQuery({
    queryKey: joinKeys.one(id ?? 0),
    queryFn: async () => oneApp.parse((await apiClient.get<ApiEnvelope<unknown>>(`${BASE}/${id}`)).data.data).application,
    enabled: isAuthenticated() && id !== null,
    staleTime: 0,
    retry: false,
  });

/** GET /applications/:id/status — status + the four-step timeline of the done screen. */
export const useJoinStatus = (id: number | null) =>
  useQuery({
    queryKey: joinKeys.status(id ?? 0),
    queryFn: async () => statusSchema.parse((await apiClient.get<ApiEnvelope<unknown>>(`${BASE}/${id}/status`)).data.data),
    enabled: isAuthenticated() && id !== null,
    staleTime: 0,
  });

/** POST /applications — opens a draft (201) or returns the open one (200). */
export function useOpenJoinDraft() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async () => oneApp.parse((await apiClient.post<ApiEnvelope<unknown>>(BASE)).data.data).application,
    onSuccess: (app) => {
      qc.setQueryData(joinKeys.one(app.id), app);
      void qc.invalidateQueries({ queryKey: joinKeys.list() });
    },
  });
}

/** PUT /applications/:id/steps/:n — auto-saves the whole step form; the answer is the fresh application. */
export function useSaveJoinStep(id: number) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async ({ step, body }: { step: 1 | 2 | 3 | 4; body: Record<string, unknown> }) =>
      oneApp.parse((await apiClient.put<ApiEnvelope<unknown>>(`${BASE}/${id}/steps/${step}`, body)).data.data).application,
    onSuccess: (app) => qc.setQueryData(joinKeys.one(id), app),
  });
}

/** POST /applications/:id/submit — 422 `application_incomplete` carries per-field `errors`. */
export function useSubmitJoin(id: number) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async () => oneApp.parse((await apiClient.post<ApiEnvelope<unknown>>(`${BASE}/${id}/submit`)).data.data).application,
    onSuccess: (app) => {
      qc.setQueryData(joinKeys.one(id), app);
      void qc.invalidateQueries({ queryKey: joinKeys.all });
    },
  });
}

/** DELETE /applications/:id — a draft and its files. */
export function useDeleteJoinDraft() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (id: number) => void (await apiClient.delete(`${BASE}/${id}`)),
    onSuccess: () => void qc.invalidateQueries({ queryKey: joinKeys.all }),
  });
}

/** POST /applications/:id/photos (multipart `file`) → the stored photo; refetches the application. */
export function useAddJoinPhoto(id: number) {
  const qc = useQueryClient();
  return useMutation<JoinFile, unknown, { file: File; onProgress?: (fraction: number) => void }>({
    mutationFn: async ({ file, onProgress }) => {
      const form = new FormData();
      form.append('file', file);
      const { data } = await apiClient.post<ApiEnvelope<unknown>>(`${BASE}/${id}/photos`, form, {
        timeoutMs: UPLOAD_TIMEOUT_MS,
        onUploadProgress: onProgress,
      });
      return z.object({ photo: fileSchema }).parse(data.data).photo;
    },
    onSettled: () => void qc.invalidateQueries({ queryKey: joinKeys.one(id) }),
  });
}

/** DELETE /applications/:id/photos/:fileId. */
export function useRemoveJoinPhoto(id: number) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (fileId: number) => void (await apiClient.delete(`${BASE}/${id}/photos/${fileId}`)),
    onSettled: () => void qc.invalidateQueries({ queryKey: joinKeys.one(id) }),
  });
}

/** PUT /applications/:id/documents/:slot (multipart `file`) — private; replaces the slot's file. */
export function useUploadJoinDocument(id: number) {
  const qc = useQueryClient();
  return useMutation<JoinFile, unknown, { slot: DocumentSlot; file: File; onProgress?: (fraction: number) => void }>({
    mutationFn: async ({ slot, file, onProgress }) => {
      const form = new FormData();
      form.append('file', file);
      const { data } = await apiClient.put<ApiEnvelope<unknown>>(`${BASE}/${id}/documents/${slot}`, form, {
        timeoutMs: UPLOAD_TIMEOUT_MS,
        onUploadProgress: onProgress,
      });
      return z.object({ document: fileSchema }).parse(data.data).document;
    },
    onSettled: () => void qc.invalidateQueries({ queryKey: joinKeys.one(id) }),
  });
}

/** DELETE /applications/:id/documents/:slot. */
export function useRemoveJoinDocument(id: number) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: async (slot: DocumentSlot) => void (await apiClient.delete(`${BASE}/${id}/documents/${slot}`)),
    onSettled: () => void qc.invalidateQueries({ queryKey: joinKeys.one(id) }),
  });
}

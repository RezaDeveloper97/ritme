'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { z } from 'zod';

import { api, createResource, pagedSchema } from '@/shared/api';

import { listingSchema, listingsApi } from './city-services';

/**
 * CB-DIR-05 — the mother & child directory in bloom's city-services admin: application review queue (capability
 * content; contact details only with «support»), a place's directory fields, services and slots, review moderation
 * and wrong-information reports.
 */

const hoursSchema = z.object({ day: z.number(), open: z.string(), close: z.string() });

const offerSchema = z.object({
  kind: z.string(),
  name: z.string(),
  description: z.string().nullable().optional(),
  format: z.string().nullable().optional(),
  duration_minutes: z.number().nullable().optional(),
  age_min_months: z.number().nullable().optional(),
  age_max_months: z.number().nullable().optional(),
  capacity: z.number().nullable().optional(),
  price_toman: z.number().nullable().optional(),
});

export const applicationSchema = z.object({
  id: z.number(),
  tracking_code: z.string().nullable(),
  status: z.string(),
  business_name: z.string().nullable(),
  category_slugs: z.array(z.string()),
  city_id: z.number().nullable(),
  district: z.string().nullable(),
  contact_name: z.string().nullable(),
  contact_phone: z.string().nullable(),
  contact_hidden: z.boolean(),
  photos_count: z.number(),
  booking_mode: z.string().nullable(),
  submitted_at: z.string().nullable(),
  reviewed_at: z.string().nullable(),
  review_note: z.string().nullable(),
  listing_id: z.number().nullable(),
  allowed_moves: z.array(z.string()),
  can_approve: z.boolean(),
  // detail only
  owner_confirmed: z.boolean().optional(),
  lat: z.string().nullable().optional(),
  lng: z.string().nullable().optional(),
  address: z.string().nullable().optional(),
  about: z.string().nullable().optional(),
  age_ranges: z.array(z.string()).optional(),
  amenities: z.array(z.string()).optional(),
  weekly_hours: z.array(hoursSchema).optional(),
  services: z.array(offerSchema).optional(),
  /** Pending photos are private until approval: `url` is a 5-minute signed link (`expires_at`); approved = public URL. */
  photos: z
    .array(z.object({ id: z.number(), url: z.string(), expires_at: z.string().nullable().optional(), pending: z.boolean().optional() }))
    .optional(),
  documents: z.array(z.object({ slot: z.string(), present: z.boolean(), mime: z.string().nullable(), size: z.number().nullable() })).optional(),
});
export type Application = z.infer<typeof applicationSchema>;

export const applicationsApi = createResource({
  key: 'city-service-applications',
  path: '/city-service-applications',
  list: pagedSchema(applicationSchema, {}),
  detail: z.object({ application: applicationSchema }),
  alsoInvalidate: [['city-service-listings']],
});

/** GET …/documents/:slot — a fresh 5-minute signed link (each view is audited); opened in a new tab, never stored. */
export function useDocumentLink(id: number) {
  return useMutation({
    mutationFn: (slot: string) => api.get(`/city-service-applications/${id}/documents/${slot}`, { schema: z.object({ url: z.string() }) }),
  });
}

export const reviewSchema = z.object({
  id: z.number(),
  listing: z.object({ id: z.number(), name: z.string() }),
  rating: z.number(),
  score_cleanliness: z.number().nullable(),
  score_instructor: z.number().nullable(),
  score_amenities: z.number().nullable(),
  body: z.string().nullable(),
  status: z.string(),
  author: z.string().nullable(),
  created_at: z.string().nullable(),
});
export type Review = z.infer<typeof reviewSchema>;

export const reviewsApi = createResource({
  key: 'city-service-reviews',
  path: '/city-services/reviews',
  list: pagedSchema(reviewSchema, {}),
  detail: z.unknown(),
  alsoInvalidate: [['city-service-listings']],
});

export const reportSchema = z.object({
  id: z.number(),
  listing: z.object({ id: z.number(), name: z.string() }),
  reason: z.string(),
  note: z.string().nullable(),
  status: z.string(),
  resolved_at: z.string().nullable(),
  created_at: z.string().nullable(),
});
export type Report = z.infer<typeof reportSchema>;

export const reportsApi = createResource({
  key: 'city-service-reports',
  path: '/city-services/reports',
  list: pagedSchema(reportSchema, {}),
  detail: z.unknown(),
});

const listingOne = z.object({ listing: listingSchema });

/** PUT /city-services/listings/:id/directory. */
export function useSaveDirectory(id: number) {
  const client = useQueryClient();
  return useMutation({
    mutationFn: (body: unknown) => api.put(`/city-services/listings/${id}/directory`, body, { schema: listingOne }),
    onSuccess: () => void client.invalidateQueries({ queryKey: listingsApi.keys.all }),
  });
}

/** DELETE /city-services/listings/:id/file-photos/:fileId. */
export function useRemoveFilePhoto(id: number) {
  const client = useQueryClient();
  return useMutation({
    mutationFn: (fileId: number) => api.delete(`/city-services/listings/${id}/file-photos/${fileId}`, { schema: listingOne }),
    onSuccess: () => void client.invalidateQueries({ queryKey: listingsApi.keys.all }),
  });
}

export const serviceSchema = z.object({
  id: z.number(),
  kind: z.string(),
  name: z.string(),
  description: z.string().nullable(),
  format: z.string().nullable(),
  duration_minutes: z.number().nullable(),
  age_min_months: z.number().nullable(),
  age_max_months: z.number().nullable(),
  capacity: z.number().nullable(),
  price_toman: z.number().nullable(),
  sort_order: z.number(),
  is_active: z.boolean(),
  slots_count: z.number(),
  bookings_count: z.number(),
});
export type PlaceService = z.infer<typeof serviceSchema>;

export const slotSchema = z.object({
  id: z.number(),
  service_id: z.number().nullable(),
  starts_at: z.string(),
  ends_at: z.string().nullable(),
  capacity: z.number(),
  booked_count: z.number(),
  status: z.string(),
});
export type PlaceSlot = z.infer<typeof slotSchema>;

const servicesKey = (id: number) => ['city-service-services', id] as const;
const slotsKey = (id: number) => ['city-service-slots', id] as const;

export const useServices = (listingId: number) =>
  useQuery({
    queryKey: servicesKey(listingId),
    queryFn: ({ signal }) => api.get(`/city-services/listings/${listingId}/services`, { schema: z.object({ items: z.array(serviceSchema) }), signal }),
  });

/** Save (POST / PUT) or delete a service of a place. */
export function useServiceMutations(listingId: number) {
  const client = useQueryClient();
  const done = () => void client.invalidateQueries({ queryKey: servicesKey(listingId) });
  const save = useMutation({
    mutationFn: ({ id, body }: { id: number | null; body: unknown }) =>
      id === null
        ? api.post(`/city-services/listings/${listingId}/services`, body)
        : api.put(`/city-services/listings/${listingId}/services/${id}`, body),
    onSuccess: done,
  });
  const remove = useMutation({
    mutationFn: (id: number) => api.delete(`/city-services/listings/${listingId}/services/${id}`),
    onSuccess: done,
  });
  return { save, remove };
}

export const useSlots = (listingId: number, from: string) =>
  useQuery({
    queryKey: [...slotsKey(listingId), from],
    queryFn: ({ signal }) =>
      api.get(`/city-services/listings/${listingId}/slots`, { query: from ? { from } : {}, schema: z.object({ items: z.array(slotSchema) }), signal }),
  });

/** Create one slot, run the weekly generator, change capacity / status (never below booked seats) or delete. */
export function useSlotMutations(listingId: number) {
  const client = useQueryClient();
  const done = () => {
    void client.invalidateQueries({ queryKey: slotsKey(listingId) });
    void client.invalidateQueries({ queryKey: servicesKey(listingId) });
  };
  const base = `/city-services/listings/${listingId}/slots`;
  const create = useMutation({ mutationFn: (body: unknown) => api.post(base, body), onSuccess: done });
  const generate = useMutation({
    mutationFn: (body: unknown) => api.post(`${base}/generate`, body, { schema: z.object({ created: z.number(), skipped: z.number() }) }),
    onSuccess: done,
  });
  const update = useMutation({ mutationFn: ({ id, body }: { id: number; body: unknown }) => api.put(`${base}/${id}`, body), onSuccess: done });
  const remove = useMutation({ mutationFn: (id: number) => api.delete(`${base}/${id}`), onSuccess: done });
  return { create, generate, update, remove };
}

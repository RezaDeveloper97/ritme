'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { z } from 'zod';

import { api, createResource, pagedSchema, translationsSchema } from '@/shared/api';

/** Doctors directory, bookings, refunds and the doctor inbox (admin-api.md §19, §19b, §19c; B-N7-08). */

const list = <T extends z.ZodType>(item: T) => z.preprocess((v) => (Array.isArray(v) ? v : []), z.array(item));
const nullableNumber = z.number().nullable().catch(null);

export const visitTypeSchema = z.object({
  id: z.number(),
  mode: z.string(),
  duration_minutes: z.number(),
  price_rials: z.number(),
  note: translationsSchema,
  address: translationsSchema,
  is_active: z.boolean(),
});
export type VisitType = z.infer<typeof visitTypeSchema>;

export const ruleSchema = z.object({
  id: z.number(),
  weekday: z.number(),
  start_time: z.string(),
  end_time: z.string(),
  slot_minutes: z.number(),
  modes: z.array(z.string()).nullable().catch(null),
});
export type Rule = z.infer<typeof ruleSchema>;

export const timeOffSchema = z.object({
  id: z.number(),
  starts_at: z.string().nullable(),
  ends_at: z.string().nullable(),
  note: z.string().nullable(),
});
export type TimeOff = z.infer<typeof timeOffSchema>;

export const doctorSchema = z.object({
  id: z.number(),
  kind: z.string(),
  name: translationsSchema,
  headline: translationsSchema,
  bio: translationsSchema,
  specialty: z.string(),
  city: z.string().nullable(),
  licence_no: z.string(),
  experience_years: nullableNumber,
  response_minutes: nullableNumber,
  photo_url: z.string().nullable().catch(null),
  visits_count: z.number().catch(0),
  rating: z.union([z.number(), z.string()]).nullable().catch(null),
  reviews_count: z.number().catch(0),
  is_active: z.boolean(),
  sort_order: z.number(),
  admin_id: nullableNumber,
  insurers: list(z.string()),
  visit_types: list(visitTypeSchema),
  availability: list(ruleSchema),
  time_off: list(timeOffSchema),
  modes: list(z.string()),
});
export type Doctor = z.infer<typeof doctorSchema>;

export const doctorOptionsSchema = z.object({
  kinds: z.array(z.string()),
  modes: z.array(z.string()),
  limits: z.record(z.string(), z.number()),
  next_sort_order: z.number(),
});
export type DoctorOptions = z.infer<typeof doctorOptionsSchema>;

export const doctorsApi = createResource({
  key: 'telemed-doctors',
  path: '/telemed/doctors',
  list: pagedSchema(doctorSchema, { filters: z.object({ q: z.string(), status: z.string() }).partial().optional() }),
  detail: z.object({ doctor: doctorSchema }),
  options: doctorOptionsSchema,
});

/** `PUT|POST|DELETE /telemed/doctors/:id/<path>` writes of the schedule panels and the photo. */
export function useDoctorWrite(id: number) {
  const client = useQueryClient();
  return useMutation({
    mutationFn: ({ method, path, body }: { method: 'put' | 'post' | 'delete'; path: string; body?: unknown }) =>
      method === 'delete'
        ? api.delete(`/telemed/doctors/${id}/${path}`)
        : api[method](`/telemed/doctors/${id}/${path}`, body),
    onSuccess: () => client.invalidateQueries({ queryKey: doctorsApi.keys.all }),
  });
}

const catalogSchema = z.object({
  items: z.array(z.object({ code: z.string(), title: translationsSchema, is_active: z.boolean() })),
});
export type CatalogOption = z.infer<typeof catalogSchema>['items'][number];

/** Active codes of one telemed catalog group (`telemed_specialties`, `telemed_cities`, `telemed_insurers`). */
export function useCatalogOptions(group: string) {
  return useQuery({
    queryKey: ['telemed-catalog', group],
    queryFn: ({ signal }) => api.get(`/catalog/${group}`, { query: { per_page: 100 }, schema: catalogSchema, signal }),
    staleTime: 5 * 60_000,
  });
}

export const reviewSchema = z.object({
  id: z.number(),
  doctor_id: z.number(),
  user_id: z.number(),
  rating: z.number(),
  body: z.string().nullable(),
  is_visible: z.boolean(),
  created_at: z.string().nullable(),
});
export type Review = z.infer<typeof reviewSchema>;

export const reviewsApi = createResource({
  key: 'telemed-reviews',
  path: '/telemed/reviews',
  list: pagedSchema(reviewSchema, { filters: z.object({ visibility: z.string() }).partial().optional() }),
  detail: z.object({ review: reviewSchema }),
});

export function useReviewVisibility() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: ({ id, visible }: { id: number; visible: boolean }) => api.put(`/telemed/reviews/${id}`, { is_visible: visible }),
    onSuccess: () => client.invalidateQueries({ queryKey: reviewsApi.keys.all }),
  });
}

export const bookingSchema = z.object({
  id: z.number(),
  reference: z.string(),
  user_id: z.number(),
  user_mobile: z.string().nullable(),
  doctor: z.object({ id: z.number(), name: translationsSchema }),
  mode: z.string(),
  status: z.string(),
  payment_status: z.string(),
  starts_at: z.string().nullable(),
  total_rials: z.number(),
  refunded_rials: z.number(),
  refund_ref: z.string().nullable(),
  refunded_at: z.string().nullable(),
  paid_at: z.string().nullable(),
});
export type AdminBooking = z.infer<typeof bookingSchema>;

export const bookingsApi = createResource({
  key: 'telemed-bookings',
  path: '/telemed/bookings',
  list: pagedSchema(bookingSchema, {
    counts: z.object({ refund_pending: z.number() }).optional(),
    statuses: z.array(z.string()).optional(),
    payment_statuses: z.array(z.string()).optional(),
  }),
  detail: z.unknown(),
});

export function useMarkRefunded() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: ({ id, refundRef }: { id: number; refundRef: string }) =>
      api.post(`/telemed/bookings/${id}/refunded`, refundRef ? { refund_ref: refundRef } : {}),
    onSuccess: () => client.invalidateQueries({ queryKey: bookingsApi.keys.all }),
  });
}

/* ---- Doctor inbox (§19b) ---- */

export const inboxBookingSchema = z.object({
  id: z.number(),
  reference: z.string(),
  mode: z.string(),
  status: z.string(),
  starts_at: z.string().nullable(),
  ends_at: z.string().nullable(),
  for_whom: z.string(),
  chat_open: z.boolean(),
});
export type InboxBooking = z.infer<typeof inboxBookingSchema>;

export const inboxKeys = {
  list: ['doctor-inbox', 'list'] as const,
  thread: (id: number) => ['doctor-inbox', 'thread', id] as const,
  rx: (id: number) => ['doctor-inbox', 'rx', id] as const,
  video: (id: number) => ['doctor-inbox', 'video', id] as const,
};

export function useInboxBookings() {
  return useQuery({
    queryKey: inboxKeys.list,
    queryFn: ({ signal }) => api.get('/telemed/doctor/bookings', { schema: z.object({ items: z.array(inboxBookingSchema) }), signal }),
    retry: false,
  });
}

export const messageSchema = z.object({
  id: z.number(),
  sender: z.string(),
  body: z.string().nullable(),
  created_at: z.string().nullable(),
  read_at: z.string().nullable().optional(),
  mine: z.boolean().catch(false),
  attachment: z.object({ mime: z.string().nullable().optional(), url: z.string().nullable().optional() }).nullable().optional(),
});
export type Message = z.infer<typeof messageSchema>;

export function useThread(id: number | null) {
  return useQuery({
    queryKey: inboxKeys.thread(id ?? 0),
    queryFn: ({ signal }) =>
      api.get(`/telemed/doctor/bookings/${id}/messages`, {
        schema: z.object({ items: z.array(messageSchema), chat: z.object({ open: z.boolean(), closes_at: z.string().nullable().optional() }) }),
        signal,
      }),
    enabled: id !== null,
    refetchInterval: 15_000,
  });
}

export function useSendMessage(id: number) {
  const client = useQueryClient();
  return useMutation({
    mutationFn: (body: string) => api.post(`/telemed/doctor/bookings/${id}/messages`, { body }),
    onSuccess: () => client.invalidateQueries({ queryKey: inboxKeys.thread(id) }),
  });
}

export const rxItemSchema = z.object({
  id: z.number(),
  drug: z.string(),
  dose: z.string().nullable().optional(),
  form: z.string().nullable().optional(),
  times: z.array(z.string()).catch([]),
  duration_days: z.number().nullable().optional(),
  instructions: z.string().nullable().optional(),
});
export type RxItem = z.infer<typeof rxItemSchema>;

export function usePrescription(id: number | null) {
  return useQuery({
    queryKey: inboxKeys.rx(id ?? 0),
    queryFn: ({ signal }) => api.get(`/telemed/doctor/bookings/${id}/prescription`, { schema: z.object({ items: z.array(rxItemSchema) }), signal }),
    enabled: id !== null,
  });
}

export function useRxWrite(id: number) {
  const client = useQueryClient();
  return useMutation({
    mutationFn: (v: { item?: number; body?: unknown }) =>
      v.item ? api.delete(`/telemed/doctor/bookings/${id}/prescription/${v.item}`) : api.post(`/telemed/doctor/bookings/${id}/prescription`, v.body),
    onSuccess: () => client.invalidateQueries({ queryKey: inboxKeys.rx(id) }),
  });
}

export function useVideo(id: number | null) {
  return useQuery({
    queryKey: inboxKeys.video(id ?? 0),
    queryFn: ({ signal }) =>
      api.get(`/telemed/doctor/bookings/${id}/video`, {
        schema: z.object({
          video: z.object({ active: z.boolean(), opens_at: z.string().nullable().optional(), closes_at: z.string().nullable().optional(), url: z.string().nullable().optional() }),
        }),
        signal,
      }),
    enabled: id !== null,
    retry: false,
  });
}

'use client';

import { keepPreviousData, useQuery, useQueryClient } from '@tanstack/react-query';

import { type ApiEnvelope, apiClient } from '@/shared/api';
import { isAuthenticated } from '@/shared/session';

import { bookingKeys, doctorKeys } from './keys';
import {
  bookingListSchema,
  bookingOneSchema,
  doctorFiltersSchema,
  doctorListSchema,
  doctorProfileSchema,
  prescriptionSchema,
  quoteSchema,
  reviewsPageSchema,
  slotsSchema,
  threadSchema,
  videoSchema,
  type Thread,
  type VisitMode,
} from './schema';

export interface DoctorQuery {
  [key: string]: string | number | boolean | undefined;
  mode?: VisitMode;
  specialty?: string;
  kind?: string;
  city?: string;
  insurance?: string;
  today?: boolean;
  q?: string;
  page?: number;
}

type Params = Record<string, string | number | boolean | undefined | null>;

async function get<T>(path: string, schema: { parse: (v: unknown) => T }, params?: Params): Promise<T> {
  const { data } = await apiClient.get<ApiEnvelope<unknown>>(path, { params });
  return schema.parse(data.data);
}

const enabled = () => isAuthenticated();

/** GET /telemed/doctors/filters — the chips (specialties / cities / insurers, catalog labels). */
export const useDoctorFilters = () =>
  useQuery({ queryKey: doctorKeys.filters(), queryFn: () => get('/telemed/doctors/filters', doctorFiltersSchema), enabled: enabled(), staleTime: 300_000 });

/** GET /telemed/doctors — the directory page (20 per page). */
export function useDoctors(query: DoctorQuery) {
  const params = { ...query, today: query.today ? 1 : undefined };
  return useQuery({
    queryKey: doctorKeys.list(params),
    queryFn: () => get('/telemed/doctors', doctorListSchema, params),
    enabled: enabled(),
    placeholderData: keepPreviousData,
    staleTime: 30_000,
  });
}

export const useDoctor = (id: number) =>
  useQuery({
    queryKey: doctorKeys.profile(id),
    queryFn: async () => (await get(`/telemed/doctors/${id}`, { parse: (v) => doctorProfileSchema.parse((v as { doctor: unknown }).doctor) })),
    enabled: enabled() && id > 0,
    retry: 1,
  });

export const useDoctorSlots = (id: number, mode: VisitMode | null, from: string) =>
  useQuery({
    queryKey: doctorKeys.slots(id, mode ?? '', from),
    queryFn: () => get(`/telemed/doctors/${id}/slots`, slotsSchema, { mode: mode ?? undefined, from, days: 7 }),
    enabled: enabled() && id > 0 && mode !== null,
    placeholderData: keepPreviousData,
  });

export const useDoctorReviews = (id: number, enabledFlag: boolean) =>
  useQuery({ queryKey: doctorKeys.reviews(id), queryFn: () => get(`/telemed/doctors/${id}/reviews`, reviewsPageSchema), enabled: enabled() && enabledFlag });

export const useBookings = (scope: 'upcoming' | 'past' | 'all', enabledFlag = true) =>
  useQuery({ queryKey: bookingKeys.list(scope), queryFn: () => get('/telemed/bookings', bookingListSchema, { scope }), enabled: enabled() && enabledFlag });

export const useBooking = (id: number) =>
  useQuery({
    queryKey: bookingKeys.one(id),
    queryFn: async () => (await get(`/telemed/bookings/${id}`, bookingOneSchema)).booking,
    enabled: enabled() && id > 0,
    retry: 1,
  });

export const useBookingQuote = (doctorId: number, mode: VisitMode | null) =>
  useQuery({
    queryKey: bookingKeys.quote(doctorId, mode ?? ''),
    queryFn: () => get('/telemed/bookings/quote', quoteSchema, { doctor_id: doctorId, mode }),
    enabled: enabled() && doctorId > 0 && mode !== null,
    retry: 1,
  });

export const usePrescription = (id: number) =>
  useQuery({ queryKey: bookingKeys.prescription(id), queryFn: () => get(`/telemed/bookings/${id}/prescription`, prescriptionSchema), enabled: enabled() && id > 0 });

export const useVideoLink = (id: number, on: boolean) =>
  useQuery({
    queryKey: bookingKeys.video(id),
    queryFn: async () => (await get(`/telemed/bookings/${id}/video`, videoSchema)).video,
    enabled: enabled() && id > 0 && on,
    refetchInterval: 30_000,
    retry: false,
  });

export async function fetchThread(id: number, afterId: number) {
  return get(`/telemed/bookings/${id}/messages`, threadSchema, afterId > 0 ? { after_id: afterId } : undefined);
}

/**
 * Chat thread (B-N7-04): the first fetch loads everything, every later poll asks `after_id` = the last id seen and
 * appends. The cache holds the whole list; `chat.open` comes from the latest poll.
 */
export function useThread(id: number, intervalMs = 6000) {
  const client = useQueryClient();
  return useQuery({
    queryKey: bookingKeys.messages(id),
    queryFn: async () => {
      const prev = client.getQueryData<Thread>(bookingKeys.messages(id));
      const lastId = prev?.items.at(-1)?.id ?? 0;
      const next = await fetchThread(id, lastId);
      if (!prev) return next;
      const seen = new Set(prev.items.map((m) => m.id));
      return { chat: next.chat, items: [...prev.items, ...next.items.filter((m) => !seen.has(m.id))] };
    },
    enabled: enabled() && id > 0,
    refetchInterval: (q) => (q.state.status === 'error' ? false : intervalMs),
    refetchIntervalInBackground: false,
    staleTime: 0,
    retry: 1,
  });
}

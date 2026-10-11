'use client';

import { keepPreviousData, useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { type ApiEnvelope, apiClient } from '@/shared/api';
import { isAuthenticated } from '@/shared/session';

import { cityServiceKeys, type DirectoryParams } from './keys';
import { type AgeRange, bookingOneSchema, bookingsSchema, listingOneSchema, listingsSchema, optionsSchema, slotsSchema } from './schema';

/** GET /city-services/options — cities with a visible listing and the active categories. */
export const useCityServiceOptions = () =>
  useQuery({
    queryKey: cityServiceKeys.options(),
    queryFn: async () => optionsSchema.parse((await apiClient.get<ApiEnvelope<unknown>>('/city-services/options')).data.data),
    enabled: isAuthenticated(),
    staleTime: 60_000,
  });

/** GET /city-services/listings?city=&category=&page= — one page (20). */
export const useCityServiceListings = (city: number | null, category: string | null, page: number) =>
  useQuery({
    queryKey: cityServiceKeys.listings(city, category, page),
    queryFn: async () =>
      listingsSchema.parse(
        (
          await apiClient.get<ApiEnvelope<unknown>>('/city-services/listings', {
            params: { ...(city ? { city } : {}), ...(category ? { category } : {}), page },
          })
        ).data.data,
      ),
    enabled: isAuthenticated(),
    placeholderData: keepPreviousData,
    staleTime: 60_000,
  });

/**
 * CB-DIR-06: GET /city-services/listings with the CB-DIR-02 search params (q, age_months, sort, per_page). Public read
 * (works without a session). The child's age is sent as whole months only — never a name or a birth date.
 */
export const useDirectoryListings = (params: DirectoryParams, enabled = true) =>
  useQuery({
    queryKey: cityServiceKeys.directory(params),
    queryFn: async () =>
      listingsSchema.parse(
        (
          await apiClient.get<ApiEnvelope<unknown>>('/city-services/listings', {
            params: {
              ...(params.city ? { city: params.city } : {}),
              ...(params.category ? { category: params.category } : {}),
              ...(params.q?.trim() ? { q: params.q.trim() } : {}),
              ...(params.ageMonths != null ? { age_months: params.ageMonths } : {}),
              ...(params.sort ? { sort: params.sort } : {}),
              ...(params.openNow ? { open_now: 1 } : {}),
              ...(params.amenities?.length ? { amenities: params.amenities.join(',') } : {}),
              ...(params.lat != null && params.lng != null ? { lat: params.lat, lng: params.lng } : {}),
              ...(params.radiusKm ? { radius_km: params.radiusKm } : {}),
              ...(params.bbox ? { bbox: params.bbox } : {}),
              ...(params.perPage ? { per_page: params.perPage } : {}),
              page: params.page ?? 1,
            },
          })
        ).data.data,
      ),
    enabled,
    placeholderData: keepPreviousData,
    staleTime: 60_000,
  });

/** GET /city-services/listings/:id — 404 for an unknown or inactive listing. */
export const useCityServiceListing = (id: number) =>
  useQuery({
    queryKey: cityServiceKeys.listing(id),
    queryFn: async () => listingOneSchema.parse((await apiClient.get<ApiEnvelope<unknown>>(`/city-services/listings/${id}`)).data.data).listing,
    enabled: isAuthenticated() && Number.isFinite(id),
    staleTime: 60_000,
    retry: false,
  });

/** CB-DIR-08: GET /city-services/listings/:id/slots?service_id= — open slots of one service (31 days). Public read. */
export const useCityServiceSlots = (id: number, serviceId: number | null) =>
  useQuery({
    queryKey: cityServiceKeys.slots(id, serviceId),
    queryFn: async () =>
      slotsSchema.parse(
        (await apiClient.get<ApiEnvelope<unknown>>(`/city-services/listings/${id}/slots`, { params: { service_id: serviceId } })).data.data,
      ).slots,
    enabled: serviceId !== null && Number.isFinite(id),
    staleTime: 30_000,
  });

/** GET /city-services/bookings — my own requests, newest first («رزروها»). */
export const useCityBookings = () =>
  useQuery({
    queryKey: cityServiceKeys.bookings(),
    queryFn: async () => bookingsSchema.parse((await apiClient.get<ApiEnvelope<unknown>>('/city-services/bookings')).data.data).bookings,
    enabled: isAuthenticated(),
  });

export interface CityBookingInput {
  listingId: number;
  /** Omitted on the slot flow: the API defaults them to the account's name and mobile (CB-DIR-09). */
  contactName?: string;
  contactPhone?: string;
  preferredDate?: string;
  childAgeRange?: AgeRange | null;
  note: string;
  /** CB-DIR-09: the slot flow — a service, one of her own children and a seat. */
  serviceId?: number | null;
  slotId?: number | null;
  childId?: number | null;
}

/** POST /city-services/bookings → 201. Contact details go to the API only (never to storage or analytics). */
export function useCreateCityBooking() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (input: CityBookingInput) =>
      bookingOneSchema.parse(
        (
          await apiClient.post<ApiEnvelope<unknown>>('/city-services/bookings', {
            listing_id: input.listingId,
            ...(input.contactName !== undefined ? { contact_name: input.contactName } : {}),
            ...(input.contactPhone !== undefined ? { contact_phone: input.contactPhone } : {}),
            ...(input.preferredDate !== undefined ? { preferred_date: input.preferredDate } : {}),
            ...(input.childAgeRange !== undefined ? { child_age_range: input.childAgeRange } : {}),
            ...(input.serviceId ? { service_id: input.serviceId } : {}),
            ...(input.slotId ? { slot_id: input.slotId } : {}),
            ...(input.childId ? { child_id: input.childId } : {}),
            note: input.note.trim() === '' ? null : input.note.trim(),
          })
        ).data.data,
      ).booking,
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: cityServiceKeys.bookings() }),
  });
}

/** POST /city-services/bookings/:id/cancel — owner only. */
export function useCancelCityBooking() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (id: number) =>
      bookingOneSchema.parse((await apiClient.post<ApiEnvelope<unknown>>(`/city-services/bookings/${id}/cancel`)).data.data).booking,
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: cityServiceKeys.bookings() }),
  });
}

/** CB-DIR-09: GET /city-services/bookings?scope= — «رزروهای من» tabs with the counts of both. */
export const useCityBookingScope = (scope: 'upcoming' | 'past') =>
  useQuery({
    queryKey: cityServiceKeys.bookingList(scope),
    queryFn: async () => bookingsSchema.parse((await apiClient.get<ApiEnvelope<unknown>>('/city-services/bookings', { params: { scope } })).data.data),
    enabled: isAuthenticated(),
  });

/** CB-DIR-09: GET /city-services/bookings/:id — the «booked» screen (owner only; 404 otherwise). */
export const useCityBooking = (id: number) =>
  useQuery({
    queryKey: cityServiceKeys.booking(id),
    queryFn: async () => bookingOneSchema.parse((await apiClient.get<ApiEnvelope<unknown>>(`/city-services/bookings/${id}`)).data.data).booking,
    enabled: isAuthenticated() && Number.isFinite(id) && id > 0,
    retry: false,
  });

/** CB-DIR-09: POST /city-services/bookings/:id/reschedule — another open slot; the team confirms again. */
export function useRescheduleCityBooking() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (v: { id: number; slotId: number }) =>
      bookingOneSchema.parse((await apiClient.post<ApiEnvelope<unknown>>(`/city-services/bookings/${v.id}/reschedule`, { slot_id: v.slotId })).data.data).booking,
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: cityServiceKeys.bookings() }),
  });
}

export interface CityReviewInput {
  id: number;
  rating: number;
  body: string;
}

/** CB-DIR-09: POST /city-services/bookings/:id/review — stored pending until the team moderates it. */
export function useReviewCityBooking() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (v: CityReviewInput) => {
      await apiClient.post(`/city-services/bookings/${v.id}/review`, { rating: v.rating, body: v.body.trim() === '' ? null : v.body.trim() });
    },
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: cityServiceKeys.bookings() }),
  });
}

export type ReportReason = 'wrong_info' | 'closed' | 'wrong_location' | 'other';
export const REPORT_REASONS: readonly ReportReason[] = ['wrong_info', 'closed', 'wrong_location', 'other'];

/** CB-DIR-09: POST /city-services/listings/:id/report — «گزارش اطلاعات نادرست»; 422 when she already has one open. */
export const useReportCityListing = (id: number) =>
  useMutation({
    mutationFn: async (v: { reason: ReportReason; note: string }) => {
      await apiClient.post(`/city-services/listings/${id}/report`, { reason: v.reason, note: v.note.trim() === '' ? null : v.note.trim() });
    },
  });

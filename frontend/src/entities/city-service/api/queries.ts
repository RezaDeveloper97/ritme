'use client';

import { keepPreviousData, useMutation, useQuery, useQueryClient } from '@tanstack/react-query';

import { type ApiEnvelope, apiClient } from '@/shared/api';
import { isAuthenticated } from '@/shared/session';

import { cityServiceKeys } from './keys';
import { type AgeRange, bookingOneSchema, bookingsSchema, listingOneSchema, listingsSchema, optionsSchema } from './schema';

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

/** GET /city-services/listings/:id — 404 for an unknown or inactive listing. */
export const useCityServiceListing = (id: number) =>
  useQuery({
    queryKey: cityServiceKeys.listing(id),
    queryFn: async () => listingOneSchema.parse((await apiClient.get<ApiEnvelope<unknown>>(`/city-services/listings/${id}`)).data.data).listing,
    enabled: isAuthenticated() && Number.isFinite(id),
    staleTime: 60_000,
    retry: false,
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
  contactName: string;
  contactPhone: string;
  preferredDate: string;
  childAgeRange: AgeRange | null;
  note: string;
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
            contact_name: input.contactName,
            contact_phone: input.contactPhone,
            preferred_date: input.preferredDate,
            child_age_range: input.childAgeRange,
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

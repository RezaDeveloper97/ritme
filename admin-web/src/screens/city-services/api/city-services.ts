'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { z } from 'zod';

import { api, createResource, pagedSchema, translationsSchema } from '@/shared/api';

/** City services: cities, categories, listings (capability content) and the request queue (capability support), admin-api.md §27, D-92. */

export const taxonSchema = z.object({
  id: z.number(),
  slug: z.string(),
  name: translationsSchema,
  icon: z.string().nullable().optional(),
  sort_order: z.number(),
  is_active: z.boolean(),
  listings_count: z.number(),
});
export type Taxon = z.infer<typeof taxonSchema>;

const optionsSchema = z.object({
  cities: z.array(taxonSchema),
  categories: z.array(taxonSchema),
  booking_modes: z.array(z.string()),
  statuses: z.array(z.string()),
  max_photos: z.number(),
  image: z.object({ max_kb: z.number(), min_width: z.number(), min_height: z.number(), types: z.array(z.string()) }),
});
export type CityServiceOptions = z.infer<typeof optionsSchema>;

const nameRef = z.object({ slug: z.string(), name: translationsSchema });

export const listingSchema = z.object({
  id: z.number(),
  category_id: z.number(),
  city_id: z.number(),
  category: nameRef,
  city: nameRef,
  name: z.string(),
  district: z.string().nullable(),
  address: z.string().nullable(),
  phone: z.string().nullable(),
  description: z.string().nullable(),
  price_text: z.string().nullable(),
  hours_text: z.string().nullable(),
  photos: z.array(z.string()),
  photo_urls: z.array(z.string()),
  booking_mode: z.string(),
  booking_url: z.string().nullable(),
  sort_order: z.number(),
  is_active: z.boolean(),
});
export type Listing = z.infer<typeof listingSchema>;

export const bookingSchema = z.object({
  id: z.number(),
  status: z.string(),
  status_note: z.string().nullable(),
  contact_name: z.string(),
  contact_phone: z.string(),
  preferred_date: z.string(),
  child_age_range: z.string().nullable(),
  note: z.string().nullable(),
  allowed_statuses: z.array(z.string()).nullable().transform((v) => v ?? []),
  listing: z.object({ id: z.number(), name: z.string(), phone: z.string().nullable(), category: nameRef, city: nameRef }),
  created_at: z.string().nullable(),
});
export type CityBooking = z.infer<typeof bookingSchema>;

export const citiesApi = createResource({
  key: 'city-service-cities',
  path: '/city-services/cities',
  list: z.object({ items: z.array(taxonSchema) }),
  detail: z.unknown(),
  alsoInvalidate: [['city-service-listings'], ['city-service-options']],
});

export const categoriesApi = createResource({
  key: 'city-service-categories',
  path: '/city-services/categories',
  list: z.object({ items: z.array(taxonSchema) }),
  detail: z.unknown(),
  alsoInvalidate: [['city-service-listings'], ['city-service-options']],
});

export const listingsApi = createResource({
  key: 'city-service-listings',
  path: '/city-services/listings',
  list: pagedSchema(listingSchema, {}),
  detail: z.object({ listing: listingSchema }),
  alsoInvalidate: [['city-service-cities'], ['city-service-categories'], ['city-service-options']],
});

export const bookingsApi = createResource({
  key: 'city-service-bookings',
  path: '/city-service-bookings',
  list: pagedSchema(bookingSchema, {}),
  detail: z.object({ booking: bookingSchema }),
});

/** GET /city-services/options — every city and category (any state) and the enums / limits. */
export const useCityServiceOptions = () =>
  useQuery({
    queryKey: ['city-service-options'],
    queryFn: ({ signal }) => api.get('/city-services/options', { schema: optionsSchema, signal }),
    staleTime: 60_000,
  });

const listingOne = z.object({ listing: listingSchema });

/** POST /city-services/listings/:id/photos (multipart `photo`). */
export function useAddPhoto(id: number) {
  const client = useQueryClient();
  return useMutation({
    mutationFn: (file: File) => {
      const fd = new FormData();
      fd.append('photo', file);
      return api.post(`/city-services/listings/${id}/photos`, fd, { schema: listingOne });
    },
    onSuccess: () => void client.invalidateQueries({ queryKey: listingsApi.keys.all }),
  });
}

/** DELETE /city-services/listings/:id/photos/:index. */
export function useRemovePhoto(id: number) {
  const client = useQueryClient();
  return useMutation({
    mutationFn: (index: number) => api.delete(`/city-services/listings/${id}/photos/${index}`, { schema: listingOne }),
    onSuccess: () => void client.invalidateQueries({ queryKey: listingsApi.keys.all }),
  });
}

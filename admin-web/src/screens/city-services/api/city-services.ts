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
  // CB-DIR-05: directory enums and catalog chips.
  admin_booking_modes: z.array(z.string()).default([]),
  listing_statuses: z.array(z.string()).default([]),
  review_statuses: z.array(z.string()).default([]),
  report_statuses: z.array(z.string()).default([]),
  report_reasons: z.array(z.string()).default([]),
  slot_statuses: z.array(z.string()).default([]),
  service_kinds: z.array(z.string()).default([]),
  service_formats: z.array(z.string()).default([]),
  application_statuses: z.array(z.string()).default([]),
  document_slots: z.array(z.string()).default([]),
  age_ranges: z.array(z.object({ code: z.string(), name: translationsSchema })).default([]),
  amenities: z.array(z.object({ code: z.string(), name: translationsSchema })).default([]),
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
  // CB-DIR-05: directory columns.
  status: z.string().default('published'),
  verified: z.boolean().default(false),
  lat: z.number().nullable().default(null),
  lng: z.number().nullable().default(null),
  about: z.string().nullable().default(null),
  age_ranges: z.array(z.string()).default([]),
  amenities: z.array(z.string()).default([]),
  weekly_hours: z.array(z.object({ day: z.number(), open: z.string(), close: z.string() })).default([]),
  photo_file_ids: z.array(z.number()).default([]),
  file_photo_urls: z.array(z.string()).default([]),
  website_url: z.string().nullable().default(null),
  cancel_free_hours: z.number().nullable().default(null),
  cancel_policy: z.string().nullable().default(null),
  rules: z.string().nullable().default(null),
  rating_avg: z.number().nullable().default(null),
  rating_count: z.number().default(0),
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
  // CB-DIR-05: directory booking fields (child = first name + age in months only).
  code: z.string().nullable().default(null),
  service: z.object({ id: z.number(), name: z.string().nullable() }).nullable().default(null),
  slot: z.object({ id: z.number(), starts_at: z.string().nullable(), ends_at: z.string().nullable() }).nullable().default(null),
  child: z.object({ name: z.string(), age_months: z.number() }).nullable().default(null),
  price_toman: z.number().nullable().default(null),
  confirmed_at: z.string().nullable().default(null),
  completed_at: z.string().nullable().default(null),
  cancelled_by: z.string().nullable().default(null),
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

import { z } from 'zod';

import { isIconName, type IconName } from '@/shared/ui';

/*
 * Parsers of /api/v1/city-services/* (bloom B-N10-05, backend-go/api/openapi.yaml). Snake_case = wire contract.
 * Content is admin-edited: optional text is null, unknown icons fall back. Nothing here is health data, and a
 * booking request carries only contact details, a day, an age range and a note.
 */

const text = z.string().nullable().catch(null);

export const BOOKING_MODES = ['link', 'request', 'phone'] as const;
export type BookingMode = (typeof BOOKING_MODES)[number];

export const BOOKING_STATUSES = ['requested', 'confirmed', 'rejected', 'cancelled', 'completed'] as const;
export type BookingStatus = (typeof BOOKING_STATUSES)[number];

/** The only child detail a request may carry (typed by the user). */
export const AGE_RANGES = ['0-1', '1-3', '3-6', '6-12', '12+'] as const;
export type AgeRange = (typeof AGE_RANGES)[number];

const icon = z
  .string()
  .nullable()
  .catch(null)
  .transform((v): IconName => (v && isIconName(v) ? v : 'home'));

export const taxonSchema = z.object({ id: z.number(), slug: z.string(), name: z.string(), icon: icon.optional() });
export type CityServiceTaxon = z.infer<typeof taxonSchema>;

export const optionsSchema = z.object({
  cities: z.array(taxonSchema),
  categories: z.array(taxonSchema),
  age_ranges: z.array(z.string()).catch([...AGE_RANGES]),
  max_advance_days: z.number().catch(365),
});

const categoryRef = z.object({ slug: z.string(), name: z.string(), icon });
const cityRef = z.object({ id: z.number(), slug: z.string(), name: z.string() });
const mode = z.enum(BOOKING_MODES).catch('phone');

export const cardSchema = z.object({
  id: z.number(),
  name: z.string(),
  category: categoryRef,
  city: cityRef,
  district: text,
  price_text: text,
  booking_mode: mode,
  photo_url: text,
});
export type CityServiceCard = z.infer<typeof cardSchema>;

export const detailSchema = cardSchema.extend({
  address: text,
  phone: text,
  description: text,
  hours_text: text,
  photos: z.array(z.string()).catch([]),
  booking_url: text,
});
export type CityServiceDetail = z.infer<typeof detailSchema>;

export const listingsSchema = z.object({
  listings: z.array(cardSchema),
  meta: z.object({ current_page: z.number(), last_page: z.number(), total: z.number() }).catch({ current_page: 1, last_page: 1, total: 0 }),
});

export const bookingSchema = z.object({
  id: z.number(),
  status: z.enum(BOOKING_STATUSES).catch('requested'),
  status_note: text,
  preferred_date: z.string(),
  child_age_range: text,
  note: text,
  contact_name: z.string(),
  contact_phone: z.string(),
  can_cancel: z.boolean().catch(false),
  created_at: text,
  listing: z.object({
    id: z.number(),
    name: z.string(),
    category: categoryRef,
    city: cityRef,
    district: text,
    address: text,
    phone: text,
    photo_url: text,
  }),
});
export type CityServiceBooking = z.infer<typeof bookingSchema>;

export const bookingsSchema = z.object({ bookings: z.array(bookingSchema) });
export const bookingOneSchema = z.object({ booking: bookingSchema });
export const listingOneSchema = z.object({ listing: detailSchema });

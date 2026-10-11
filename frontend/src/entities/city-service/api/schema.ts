import { z } from 'zod';

import { isIconName, type IconName } from '@/shared/ui';

/*
 * Parsers of /api/v1/city-services/* (bloom B-N10-05, backend-go/api/openapi.yaml). Snake_case = wire contract.
 * Content is admin-edited: optional text is null, unknown icons fall back. Nothing here is health data, and a
 * booking request carries only contact details, a day, an age range and a note.
 */

const text = z.string().nullable().catch(null);

// CB-DIR-02 added `online` (pick a slot; still a request the team confirms).
export const BOOKING_MODES = ['link', 'request', 'phone', 'online'] as const;
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

/** Catalog option (CB-DIR-02): `meta` carries {min_months,max_months} or {pregnancy}. */
export const catalogOptionSchema = z.object({ code: z.string(), name: text, meta: z.unknown().optional() });
export type CityServiceCatalogOption = z.infer<typeof catalogOptionSchema>;

export const optionsSchema = z.object({
  cities: z.array(taxonSchema),
  categories: z.array(taxonSchema),
  age_ranges: z.array(z.string()).catch([...AGE_RANGES]),
  max_advance_days: z.number().catch(365),
  // CB-DIR-02: the directory's place age ranges (catalog options) — used to label «مناسب …» on the cards.
  place_age_ranges: z.array(catalogOptionSchema).catch([]),
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
  // CB-DIR-02 directory fields — all optional on the wire for older payloads.
  lat: z.number().nullable().catch(null),
  lng: z.number().nullable().catch(null),
  distance_km: z.number().nullable().catch(null),
  verified: z.boolean().catch(false),
  rating_avg: z.number().nullable().catch(null),
  rating_count: z.number().catch(0),
  age_ranges: z.array(z.string()).catch([]),
  amenities: z.array(z.string()).catch([]),
  price_from_toman: z.number().nullable().catch(null),
  open_now: z.boolean().nullable().catch(null),
  next_slot_at: text,
  age_fit: z.boolean().nullable().catch(null),
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

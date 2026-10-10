import { z } from 'zod';

/*
 * Parsers of the /api/v1/telemed responses (B-N7-02 directory, B-N7-03 bookings, B-N7-04 chat; schemas in
 * backend-go/api/openapi.yaml). Kept snake_case — the field names are the wire contract. Prices are integer RIALS
 * (QUESTIONS #120): show toman through `toman()`.
 */

export const MODES = ['video', 'phone', 'in_person'] as const;
export type VisitMode = (typeof MODES)[number];
export const mode = z.enum(MODES);

const text = z.string().nullable().catch(null);
const code = z.object({ code: z.string(), title: text });
const nextSlot = z
  .object({ starts_at: z.string(), date: z.string(), time: z.string(), mode })
  .nullable()
  .catch(null);

export const doctorCardSchema = z.object({
  id: z.number(),
  kind: z.enum(['doctor', 'midwife']).catch('doctor'),
  name: z.string(),
  initial: text,
  headline: text,
  specialty: code,
  city: code.nullable().catch(null),
  experience_years: z.number().nullable().catch(null),
  photo_url: text,
  rating: z.number().nullable().catch(null),
  reviews_count: z.number().catch(0),
  modes: z.array(mode).catch([]),
  price_from_rials: z.number().catch(0),
  next_slot: nextSlot,
});
export type DoctorCard = z.infer<typeof doctorCardSchema>;

export const doctorListSchema = z.object({
  items: z.array(doctorCardSchema),
  meta: z.object({ current_page: z.number().catch(1), last_page: z.number().catch(1), total: z.number().catch(0) }).passthrough(),
});
export type DoctorList = z.infer<typeof doctorListSchema>;

const option = z.object({ code: z.string(), title: text, count: z.number().catch(0) });
export const doctorFiltersSchema = z.object({
  total: z.number().catch(0),
  specialties: z.array(option).catch([]),
  cities: z.array(option).catch([]),
  insurers: z.array(option).catch([]),
});
export type DoctorFilters = z.infer<typeof doctorFiltersSchema>;

export const visitTypeSchema = z.object({
  mode,
  duration_minutes: z.number(),
  price_rials: z.number(),
  note: text,
  address: text,
});
export type VisitType = z.infer<typeof visitTypeSchema>;

export const reviewSchema = z.object({
  id: z.number(),
  rating: z.number(),
  body: text,
  author_initial: text,
  created_at: text,
  mine: z.boolean().catch(false),
});
export type Review = z.infer<typeof reviewSchema>;

export const doctorProfileSchema = doctorCardSchema
  .omit({ modes: true, price_from_rials: true })
  .extend({
    licence_no: text,
    bio: text,
    stats: z.object({
      experience_years: z.number().nullable().catch(null),
      visits_count: z.number().catch(0),
      response_minutes: z.number().nullable().catch(null),
      satisfaction_percent: z.number().nullable().catch(null),
    }),
    visit_types: z.array(visitTypeSchema),
    insurers: z.array(code).catch([]),
    reviews: z.object({ count: z.number().catch(0), items: z.array(reviewSchema).catch([]) }),
    can_review: z.boolean().catch(false),
  });
export type DoctorProfile = z.infer<typeof doctorProfileSchema>;

export const slotsSchema = z.object({
  visit_type: visitTypeSchema,
  from: z.string(),
  days: z.array(
    z.object({
      date: z.string(),
      weekday: z.number(),
      count: z.number(),
      slots: z.array(z.object({ starts_at: z.string(), ends_at: z.string(), time: z.string() })),
    }),
  ),
  next_slot: nextSlot,
});
export type SlotsResult = z.infer<typeof slotsSchema>;

export const reviewsPageSchema = z.object({ items: z.array(reviewSchema) });

export const CONSENT_SCOPES = ['cycle_summary', 'bbt_lh', 'assistant_summaries'] as const;
export type ConsentScope = (typeof CONSENT_SCOPES)[number];
export const FOR_WHOM = ['self', 'child', 'other'] as const;
export type ForWhom = (typeof FOR_WHOM)[number];

const price = z.object({
  price_rials: z.number(),
  discount_rials: z.number().catch(0),
  discount_source: text,
  total_rials: z.number(),
});

export const bookingSchema = z.object({
  id: z.number(),
  reference: z.string(),
  status: z.enum(['held', 'confirmed', 'completed', 'cancelled', 'expired', 'failed']).catch('failed'),
  mode,
  starts_at: z.string(),
  ends_at: z.string(),
  date: z.string(),
  time: z.string(),
  duration_minutes: z.number(),
  hold_expires_at: text,
  doctor: z.object({ id: z.number(), kind: z.enum(['doctor', 'midwife']).catch('doctor'), name: z.string(), initial: text, specialty: code, photo_url: text }),
  for_whom: z.enum(FOR_WHOM).catch('self'),
  child: z.object({ id: z.number(), name: z.string() }).nullable().catch(null),
  patient_name: text,
  reason: code.nullable().catch(null),
  note: text,
  price,
  payment_status: z.string(),
  cancel_deadline: text,
  can_cancel: z.boolean().catch(false),
  can_reschedule: z.boolean().catch(false),
  consent: z.record(z.boolean()).catch({}),
});
export type Booking = z.infer<typeof bookingSchema>;

export const bookingListSchema = z.object({ items: z.array(bookingSchema) });
export const bookingOneSchema = z.object({ booking: bookingSchema });
export const bookResultSchema = z.object({
  booking: bookingSchema,
  payment: z.object({ authority: z.string(), redirect_url: z.string() }).nullable().catch(null),
});
export type BookResult = z.infer<typeof bookResultSchema>;
export const verifySchema = z.object({ booking: bookingSchema, already_paid: z.boolean().catch(false) });

export const quoteSchema = z.object({
  doctor: bookingSchema.shape.doctor,
  visit_type: visitTypeSchema,
  price,
  plus: z.boolean().catch(false),
  plus_discount_percent: z.number().nullable().catch(null),
  payments_available: z.boolean().catch(true),
  cancel_window_hours: z.number().catch(2),
  hold_minutes: z.number().catch(10),
  children: z.array(z.object({ id: z.number(), name: z.string(), birth_date: z.string().catch('') })).catch([]),
  reasons: z.array(z.object({ code: z.string(), title: z.string().catch('') })).catch([]),
});
export type Quote = z.infer<typeof quoteSchema>;

export const messageSchema = z.object({
  id: z.number(),
  sender: z.enum(['user', 'doctor']).catch('doctor'),
  mine: z.boolean(),
  body: text,
  attachment: z.object({ id: z.number(), mime: z.string(), size_bytes: z.number(), url: z.string() }).nullable().catch(null),
  created_at: z.string(),
});
export type ChatMessage = z.infer<typeof messageSchema>;
export const threadSchema = z.object({
  items: z.array(messageSchema),
  chat: z.object({ open: z.boolean(), closes_at: text }),
});
export type Thread = z.infer<typeof threadSchema>;

export const prescriptionItemSchema = z.object({
  id: z.number(),
  drug: z.string(),
  dose: text,
  form: z.string().catch(''),
  times: z.array(z.string()).catch([]),
  duration_days: z.number().nullable().catch(null),
  instructions: text,
  reminder_id: z.number().nullable().catch(null),
});
export type PrescriptionItem = z.infer<typeof prescriptionItemSchema>;
export const prescriptionSchema = z.object({ items: z.array(prescriptionItemSchema) });

export const videoSchema = z.object({
  video: z.object({ active: z.boolean(), opens_at: z.string(), closes_at: z.string(), url: text }),
});
export type VideoLink = z.infer<typeof videoSchema>['video'];

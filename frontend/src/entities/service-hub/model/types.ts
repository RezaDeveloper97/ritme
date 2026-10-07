import type { IconName, Tone } from '@/shared/ui';

/** Section codes of the «خدمات» hub (backend-go internal/services SectionCodes), in board order. */
export const SERVICE_SECTIONS = [
  'search',
  'booking',
  'care',
  'checkups',
  'programs',
  'mother_child',
  'learning',
  'shop',
  'emergency',
] as const;
export type ServiceSectionCode = (typeof SERVICE_SECTIONS)[number];

/** `live` = has an in-app destination; `soon` = the screen does not exist yet (never linked). */
export type ServiceStatus = 'live' | 'soon';

export interface ShopCategory {
  code: string;
  title: string | null;
  icon: IconName;
}

/** One hub section. Copy is admin content and may be null — the screen then uses its bundled strings. */
export interface ServiceSection {
  code: ServiceSectionCode;
  title: string | null;
  subtitle: string | null;
  caption: string | null;
  href: string | null;
  status: ServiceStatus;
  /** Emergency section only. */
  phone: string | null;
  /** Shop section only. */
  categories: ShopCategory[];
}

/** A care tile or a care-program card. */
export interface ServiceTile {
  code: string;
  title: string | null;
  subtitle: string | null;
  icon: IconName;
  tone: Tone;
  href: string | null;
  status: ServiceStatus;
  /** Record tile: the record's document count; otherwise null. */
  count: number | null;
}

export type BookingKind = 'video' | 'phone' | 'in_person';

/** The next booked visit (telemedicine, B-N7-03). */
export interface UpcomingBooking {
  id: number;
  kind: BookingKind;
  providerName: string;
  /** ISO date-time with the Tehran offset, e.g. `2026-10-15T18:00:00+03:30`. */
  startsAt: string;
  durationMinutes: number;
  href: string | null;
}

export interface ServicesHub {
  sections: ServiceSection[];
  upcomingBooking: UpcomingBooking | null;
  care: ServiceTile[];
  programs: ServiceTile[];
}

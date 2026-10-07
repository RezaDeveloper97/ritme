import { diffInDays, fromApiDate, today } from '@/shared/lib/date';

import type { UpcomingBooking } from './types';

export interface BookingWhen {
  /** Start-of-day date of the visit (format it with shared/lib/date). */
  date: Date;
  /** Wall-clock time as sent by the server (Tehran), `HH:mm`. */
  time: string;
  /** Whole days from today (0 = today). */
  daysAway: number;
}

/**
 * Splits the booking's ISO start into what the card shows. The server sends Tehran wall-clock time with its offset,
 * so the date and the time are read from the string itself — never through the device's time zone.
 */
export function bookingWhen(booking: Pick<UpcomingBooking, 'startsAt'>, now: Date = today()): BookingWhen {
  const date = fromApiDate(booking.startsAt.slice(0, 10));
  return { date, time: booking.startsAt.slice(11, 16), daysAway: Math.max(0, diffInDays(date, now)) };
}

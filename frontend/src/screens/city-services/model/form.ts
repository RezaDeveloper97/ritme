import type { AgeRange } from '@/entities/city-service';

export const NAME_MAX = 120;
export const NOTE_MAX = 500;
export const DAY_CHOICES = 14;

export interface BookingFormState {
  name: string;
  phone: string;
  day: string;
  age: AgeRange | null;
  note: string;
}

export const emptyBookingForm = (day: string): BookingFormState => ({ name: '', phone: '', day, age: null, note: '' });

export type BookingFormError = 'required' | 'phoneInvalid';
export type BookingFormErrors = Partial<Record<'name' | 'phone', BookingFormError>>;

/** Persian / Arabic digits → ASCII, spaces, dashes and brackets dropped (the server does the same before it checks). */
export function asciiDigits(value: string): string {
  return value
    .trim()
    .replace(/[۰-۹]/g, (d) => String(d.charCodeAt(0) - 0x06f0))
    .replace(/[٠-٩]/g, (d) => String(d.charCodeAt(0) - 0x0660))
    .replace(/[\s\-()]/g, '');
}

/** Client-side checks; the server validates again. An empty object = ready to send. */
export function validateBookingForm(s: BookingFormState): BookingFormErrors {
  const errors: BookingFormErrors = {};
  if (s.name.trim() === '') errors.name = 'required';
  if (s.phone.trim() === '') errors.phone = 'required';
  else if (!/^\+?[0-9]{7,15}$/.test(asciiDigits(s.phone))) errors.phone = 'phoneInvalid';
  return errors;
}

import type { LabWindow } from '@/entities/partner-lab';

export const NAME_MAX = 80;
export const CITY_MAX = 60;
export const ADDRESS_MAX = 300;
export const NOTE_MAX = 300;
export const DAY_CHOICES = 14;

export interface LabFormState {
  fullName: string;
  phone: string;
  city: string;
  address: string;
  note: string;
  day: string;
  window: LabWindow;
  consentShare: boolean;
  consentImport: boolean;
}

export const emptyLabForm = (day: string): LabFormState => ({
  fullName: '',
  phone: '',
  city: '',
  address: '',
  note: '',
  day,
  window: 'morning',
  consentShare: false,
  consentImport: false,
});

export type LabFormError = 'required' | 'phoneInvalid' | 'consentRequired';
export type LabFormErrors = Partial<Record<'fullName' | 'phone' | 'city' | 'address' | 'consentShare', LabFormError>>;

/** Persian / Arabic digits → ASCII, spaces and dashes dropped (the server does the same before it checks). */
export function asciiDigits(value: string): string {
  return value
    .trim()
    .replace(/[۰-۹]/g, (d) => String(d.charCodeAt(0) - 0x06f0))
    .replace(/[٠-٩]/g, (d) => String(d.charCodeAt(0) - 0x0660))
    .replace(/[\s-]/g, '');
}

/** Client-side checks; the server validates again. An empty object = ready to send. */
export function validateLabForm(s: LabFormState): LabFormErrors {
  const errors: LabFormErrors = {};
  if (s.fullName.trim() === '') errors.fullName = 'required';
  if (s.city.trim() === '') errors.city = 'required';
  if (s.address.trim() === '') errors.address = 'required';
  if (s.phone.trim() === '') errors.phone = 'required';
  else if (!/^\+?[0-9]{7,15}$/.test(asciiDigits(s.phone))) errors.phone = 'phoneInvalid';
  if (!s.consentShare) errors.consentShare = 'consentRequired';
  return errors;
}

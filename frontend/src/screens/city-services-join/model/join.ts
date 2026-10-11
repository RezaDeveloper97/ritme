import type { JoinApplication, JoinHours, JoinOffer, JoinSummary } from '@/entities/city-service';

/* Pure helpers of the business join flow (CB-DIR-10): step bodies, error → step mapping, number parsing, checks. */

export const STEP_COUNT = 4;
export type JoinStep = 1 | 2 | 3 | 4;

/** Days as the API numbers them: 0 = Saturday … 6 = Friday. */
export const DAYS = [0, 1, 2, 3, 4, 5, 6] as const;
/** Message keys of the days (numeric keys would read as a list in the translation bundle). */
export const DAY_KEYS = ['sat', 'sun', 'mon', 'tue', 'wed', 'thu', 'fri'] as const;
export const DEFAULT_OPEN = '09:00';
export const DEFAULT_CLOSE = '20:00';
export const PHOTO_MAX_BYTES = 5 * 1024 * 1024;
export const DOC_MAX_BYTES = 10 * 1024 * 1024;
export const PHOTO_TYPES: readonly string[] = ['image/jpeg', 'image/png', 'image/webp'];
export const DOC_TYPES: readonly string[] = ['image/jpeg', 'image/png', 'image/webp', 'application/pdf'];

/** Persian / Arabic digits → ASCII, separators dropped. */
export function foldDigits(value: string): string {
  return value
    .replace(/[۰-۹]/g, (d) => String(d.charCodeAt(0) - 0x06f0))
    .replace(/[٠-٩]/g, (d) => String(d.charCodeAt(0) - 0x0660))
    .replace(/[,٬،\s]/g, '');
}

/** A whole number from a typed field; null when empty or not a number. */
export function toInt(value: string): number | null {
  const folded = foldDigits(value);
  if (!/^\d{1,9}$/.test(folded)) return null;
  return Number(folded);
}

/** The week used when nothing is saved yet: Saturday–Thursday open, Friday closed (nbl_Dir_JoinForm). */
export function defaultHours(): JoinHours[] {
  return [0, 1, 2, 3, 4, 5].map((day) => ({ day, open: DEFAULT_OPEN, close: DEFAULT_CLOSE }));
}

export function hoursFor(saved: readonly JoinHours[]): JoinHours[] {
  return saved.length > 0 ? [...saved].sort((a, b) => a.day - b.day) : defaultHours();
}

/** True when `close` is after `open` ("HH:MM" compares as text, like the API). */
export function validHours(h: JoinHours): boolean {
  return /^\d{2}:\d{2}$/.test(h.open) && /^\d{2}:\d{2}$/.test(h.close) && h.close > h.open;
}

export function toggleItem<T>(list: readonly T[], item: T): T[] {
  return list.includes(item) ? list.filter((x) => x !== item) : [...list, item];
}

/** Step bodies — always the whole form of the step (an absent field would be stored empty). */
export interface Step1State {
  businessName: string;
  categories: string[];
  cityId: number | null;
  district: string;
  contactName: string;
  contactPhone: string;
  ownerConfirmed: boolean;
}
export interface Step2State {
  lat: number | null;
  lng: number | null;
  address: string;
  about: string;
  ageRanges: string[];
  amenities: string[];
  hours: JoinHours[];
}
export interface Step4State {
  bookingMode: string | null;
  terms: boolean;
}

export const step1From = (a: JoinApplication): Step1State => ({
  businessName: a.business_name ?? '',
  categories: a.category_slugs,
  cityId: a.city_id,
  district: a.district ?? '',
  contactName: a.contact_name ?? '',
  contactPhone: a.contact_phone ?? '',
  ownerConfirmed: a.owner_confirmed,
});
export const step2From = (a: JoinApplication): Step2State => ({
  lat: a.lat,
  lng: a.lng,
  address: a.address ?? '',
  about: a.about ?? '',
  ageRanges: a.age_ranges,
  amenities: a.amenities,
  hours: hoursFor(a.weekly_hours),
});
export const step4From = (a: JoinApplication): Step4State => ({ bookingMode: a.booking_mode, terms: a.terms_accepted_at !== null });

export const step1Body = (s: Step1State) => ({
  business_name: s.businessName.trim(),
  category_slugs: s.categories,
  city_id: s.cityId,
  district: s.district.trim(),
  contact_name: s.contactName.trim(),
  contact_phone: foldDigits(s.contactPhone),
  owner_confirmed: s.ownerConfirmed,
});
export const step2Body = (s: Step2State) => ({
  lat: s.lat,
  lng: s.lng,
  address: s.address.trim(),
  about: s.about.trim(),
  age_ranges: s.ageRanges,
  amenities: s.amenities,
  weekly_hours: s.hours.filter(validHours),
});
export const step3Body = (services: readonly JoinOffer[]) => ({ services });
export const step4Body = (s: Step4State) => ({ booking_mode: s.bookingMode, terms_accepted: s.terms });

/** Which step a submit error field belongs to (`application_incomplete` keys). Unknown → 4. */
export function stepOfField(field: string): JoinStep {
  const root = field.split('.')[0] ?? field;
  if (['business_name', 'category_slugs', 'city_id', 'district', 'contact_name', 'contact_phone', 'owner_confirmed'].includes(root)) return 1;
  if (['location', 'lat', 'lng', 'address', 'about', 'age_ranges', 'amenities', 'weekly_hours', 'photos'].includes(root)) return 2;
  if (root === 'services') return 3;
  return 4;
}

/** Per-field errors grouped by step, in step order. */
export function groupErrors(errors: Record<string, string[]>): { step: JoinStep; field: string; message: string }[] {
  return Object.entries(errors)
    .map(([field, messages]) => ({ step: stepOfField(field), field, message: messages[0] ?? '' }))
    .sort((a, b) => a.step - b.step);
}

/** The step to resume a saved draft at: past the last saved step, but never ahead of an empty first step. */
export function resumeStep(a: Pick<JoinApplication, 'step' | 'business_name'>): JoinStep {
  if (!a.business_name) return 1;
  return Math.min(STEP_COUNT, Math.max(1, a.step + 1)) as JoinStep;
}

/** What the page shows for the account's applications: the open draft, else the latest sent one, else a new draft. */
export function pickApplication(list: readonly JoinSummary[]): { kind: 'open' | 'sent' | 'none'; id: number | null } {
  const open = list.find((a) => a.editable);
  if (open) return { kind: 'open', id: open.id };
  const sent = list[0];
  return sent ? { kind: 'sent', id: sent.id } : { kind: 'none', id: null };
}

/** Client-side check of a picked file; returns an error key or null. */
export function fileProblem(file: Pick<File, 'type' | 'size' | 'name'>, kind: 'photo' | 'doc'): 'fileType' | 'fileSize' | null {
  const types = kind === 'photo' ? PHOTO_TYPES : DOC_TYPES;
  const pdfByName = kind === 'doc' && file.type === '' && file.name.toLowerCase().endsWith('.pdf');
  if (!types.includes(file.type) && !pdfByName) return 'fileType';
  if (file.size > (kind === 'photo' ? PHOTO_MAX_BYTES : DOC_MAX_BYTES)) return 'fileSize';
  return null;
}

/** A service is saveable with a name and the numbers it carries inside the API ranges. */
export interface OfferForm {
  kind: 'session' | 'package';
  name: string;
  description: string;
  format: 'group' | 'private' | null;
  duration: string;
  ageMin: string;
  ageMax: string;
  capacity: string;
  price: string;
  sessions: string;
  validity: string;
}

export const emptyOffer = (): OfferForm => ({
  kind: 'session',
  name: '',
  description: '',
  format: null,
  duration: '',
  ageMin: '',
  ageMax: '',
  capacity: '',
  price: '',
  sessions: '',
  validity: '',
});

const str = (n: number | null | undefined): string => (n === null || n === undefined ? '' : String(n));
export const offerToForm = (o: JoinOffer): OfferForm => ({
  kind: o.kind,
  name: o.name,
  description: o.description ?? '',
  format: o.format ?? null,
  duration: str(o.duration_minutes),
  ageMin: str(o.age_min_months),
  ageMax: str(o.age_max_months),
  capacity: str(o.capacity),
  price: str(o.price_toman),
  sessions: str(o.package_sessions),
  validity: str(o.package_validity_days),
});

export type OfferProblem = 'name' | 'duration' | 'capacity' | 'age' | 'sessions' | 'validity';

const inRange = (raw: string, min: number, max: number): boolean => {
  if (raw.trim() === '') return true;
  const n = toInt(raw);
  return n !== null && n >= min && n <= max;
};

export function offerProblems(f: OfferForm): OfferProblem[] {
  const out: OfferProblem[] = [];
  if (f.name.trim() === '') out.push('name');
  if (!inRange(f.duration, 5, 600)) out.push('duration');
  if (!inRange(f.capacity, 1, 500)) out.push('capacity');
  const lo = toInt(f.ageMin);
  const hi = toInt(f.ageMax);
  if (!inRange(f.ageMin, 0, 240) || !inRange(f.ageMax, 0, 240) || (lo !== null && hi !== null && hi < lo)) out.push('age');
  if (f.kind === 'package') {
    if (!inRange(f.sessions, 1, 200) || toInt(f.sessions) === null) out.push('sessions');
    if (!inRange(f.validity, 1, 730)) out.push('validity');
  }
  return out;
}

export function offerFromForm(f: OfferForm): JoinOffer {
  const pkg = f.kind === 'package';
  return {
    kind: f.kind,
    name: f.name.trim(),
    description: f.description.trim() === '' ? null : f.description.trim(),
    format: f.format,
    duration_minutes: toInt(f.duration),
    age_min_months: toInt(f.ageMin),
    age_max_months: toInt(f.ageMax),
    capacity: toInt(f.capacity),
    price_toman: toInt(f.price),
    package_sessions: pkg ? toInt(f.sessions) : null,
    package_validity_days: pkg ? toInt(f.validity) : null,
  };
}

import { z } from 'zod';

import { type IconName, isIconName, type Tone } from '@/shared/ui';

import {
  SERVICE_SECTIONS,
  type ServiceSection,
  type ServicesHub,
  type ServiceTile,
  type UpcomingBooking,
} from '../model/types';

/*
 * Parser of GET /api/v1/services (B-N7-01; schema ServicesHub in backend-go/api/openapi.yaml). Snake case → camel
 * case. Content is admin-edited: an unknown section code is dropped, an unknown icon / tone falls back, and a
 * malformed tile is skipped rather than blanking the hub. Hrefs are re-checked to be in-app paths.
 */

const TONES = ['brand', 'data', 'warm', 'period', 'bloom', 'success', 'danger', 'neutral'] as const;
const HREF = /^\/[a-z0-9][a-z0-9/_-]*$/;

const text = z.string().nullable().catch(null);
const href = z
  .string()
  .nullable()
  .catch(null)
  .transform((v) => (v && HREF.test(v) ? v : null));
const icon = (fallback: IconName) =>
  z
    .string()
    .catch(fallback)
    .transform((v): IconName => (isIconName(v) ? v : fallback));

const sectionRaw = z.object({
  code: z.string(),
  title: text,
  subtitle: text,
  caption: text,
  href,
  phone: text,
  categories: z
    .array(z.object({ code: z.string(), title: text, icon: icon('box') }))
    .catch([]),
});

const tileRaw = z.object({
  code: z.string(),
  title: text,
  subtitle: text,
  icon: icon('grid'),
  tone: z.enum(TONES).catch('neutral' satisfies Tone),
  href,
  count: z.number().int().nonnegative().nullable().optional().catch(null),
});

const bookingRaw = z.object({
  id: z.number(),
  kind: z.enum(['video', 'phone', 'in_person']),
  provider_name: z.string(),
  starts_at: z.string().regex(/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}/),
  duration_minutes: z.number(),
  href,
});

const hubRaw = z.object({
  sections: z.array(z.unknown()),
  upcoming_booking: z.unknown().nullable(),
  care: z.array(z.unknown()),
  programs: z.array(z.unknown()),
});

const isSection = (code: string): code is ServiceSection['code'] => (SERVICE_SECTIONS as readonly string[]).includes(code);

function tiles(list: unknown[]): ServiceTile[] {
  const out: ServiceTile[] = [];
  for (const raw of list) {
    const r = tileRaw.safeParse(raw);
    if (!r.success) continue;
    const t = r.data;
    out.push({
      code: t.code,
      title: t.title,
      subtitle: t.subtitle,
      icon: t.icon,
      tone: t.tone,
      href: t.href,
      status: t.href ? 'live' : 'soon',
      count: t.count ?? null,
    });
  }
  return out;
}

function booking(raw: unknown): UpcomingBooking | null {
  const r = bookingRaw.safeParse(raw);
  if (!r.success) return null;
  const b = r.data;
  return {
    id: b.id,
    kind: b.kind,
    providerName: b.provider_name,
    startsAt: b.starts_at,
    durationMinutes: b.duration_minutes,
    href: b.href,
  };
}

export const servicesHubSchema = hubRaw.transform((raw): ServicesHub => {
  const sections: ServiceSection[] = [];
  const seen = new Set<string>();
  for (const item of raw.sections) {
    const r = sectionRaw.safeParse(item);
    if (!r.success || !isSection(r.data.code) || seen.has(r.data.code)) continue;
    const s = r.data;
    seen.add(s.code);
    sections.push({
      code: s.code as ServiceSection['code'],
      title: s.title,
      subtitle: s.subtitle,
      caption: s.caption,
      href: s.href,
      status: s.href ? 'live' : 'soon',
      phone: s.phone && /^\d{3,6}$/.test(s.phone) ? s.phone : null,
      categories: s.categories,
    });
  }
  return { sections, upcomingBooking: booking(raw.upcoming_booking), care: tiles(raw.care), programs: tiles(raw.programs) };
});

import { existsSync, readFileSync } from 'node:fs';
import { resolve } from 'node:path';

import { describe, expect, it } from 'vitest';

import { servicesHubSchema } from './schema';

/* Boundary contract of GET /api/v1/services against the Go contract golden (B-N7-01). */
const GOLDEN = resolve(process.cwd(), '../backend-go/contract/golden/services/hub.en.json');

describe('servicesHubSchema', () => {
  it.skipIf(!existsSync(GOLDEN))('parses the Go golden (seeded catalog)', () => {
    const golden = JSON.parse(readFileSync(GOLDEN, 'utf8')) as { steps: { body: { data: unknown } }[] };
    const hub = servicesHubSchema.parse(golden.steps[0].body.data);
    expect(hub.sections.map((s) => s.code)).toEqual([
      'search',
      'booking',
      'care',
      'checkups',
      'programs',
      'mother_child',
      'learning',
      'shop',
      'emergency',
    ]);
    expect(hub.sections.at(-1)?.phone).toBe('115');
    expect(hub.upcomingBooking).toBeNull();
    expect(hub.care.map((t) => [t.code, t.status])).toEqual([
      ['assistant', 'soon'],
      ['doctors', 'soon'],
      ['record', 'live'],
      ['labs', 'live'],
      ['vitals', 'live'],
      ['insurance', 'soon'],
    ]);
    expect(hub.care[2]).toMatchObject({ title: 'Health record', href: '/record', count: 0, icon: 'fileDoc' });
    expect(hub.programs.find((p) => p.code === 'contraception')?.href).toBe('/contraception');
    expect(hub.sections.find((s) => s.code === 'shop')?.categories.map((c) => c.icon)).toEqual(['bottle', 'dropLine']);
  });

  it('drops unknown sections, rejects external hrefs and falls back on unknown icons / tones', () => {
    const hub = servicesHubSchema.parse({
      sections: [
        { code: 'nope', title: null },
        { code: 'shop', title: 'Shop', subtitle: null, caption: null, href: 'https://x.example', phone: null, categories: [{ code: 'a', title: 'A', icon: 'zzz' }] },
      ],
      upcoming_booking: { id: 3, kind: 'video', provider_name: 'Dr', starts_at: '2026-10-15T18:00:00+03:30', duration_minutes: 20, href: '//evil' },
      care: [
        { code: 'record', title: 'Record', subtitle: 'Docs', icon: 'fileDoc', tone: 'brand', href: '/record', status: 'live', count: 41 },
        { code: 'odd', title: 'Odd', subtitle: null, icon: '<svg>', tone: 'neon', href: null, status: 'soon' },
        { broken: true },
      ],
      programs: [],
    });
    expect(hub.sections).toHaveLength(1);
    expect(hub.sections[0]).toMatchObject({ code: 'shop', href: null, status: 'soon', categories: [{ code: 'a', icon: 'box' }] });
    expect(hub.upcomingBooking).toMatchObject({ providerName: 'Dr', href: null });
    expect(hub.care.map((t) => [t.code, t.icon, t.tone, t.status, t.count])).toEqual([
      ['record', 'fileDoc', 'brand', 'live', 41],
      ['odd', 'grid', 'neutral', 'soon', null],
    ]);
  });
});

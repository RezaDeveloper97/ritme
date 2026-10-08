import { describe, expect, it } from 'vitest';

import { conditionCatalogSchema, conditionNudgesSchema, conditionsOverviewSchema } from './schema';

describe('conditionsOverviewSchema', () => {
  it('parses the programs and drops unknown codes', () => {
    const parsed = conditionsOverviewSchema.parse({
      programs: [
        { code: 'endo', enrolled: true, enrolled_on: '2026-09-01' },
        { code: 'pmdd', enrolled: false, enrolled_on: null },
        { code: 'unknown', enrolled: true, enrolled_on: null },
      ],
    });
    expect(parsed.programs).toEqual([
      { code: 'endo', enrolled: true, enrolledOn: '2026-09-01' },
      { code: 'pmdd', enrolled: false, enrolledOn: null },
    ]);
  });
});

describe('conditionCatalogSchema', () => {
  it('reads meta.logs as the «ثبت می‌کنی» line', () => {
    const items = conditionCatalogSchema.parse({
      items: [
        { code: 'endo', title: 'Endometriosis', body: 'A pain diary', meta: { logs: 'pain location' } },
        { code: 'pcos', title: '', body: null, meta: null },
        { title: 'no code' },
      ],
    });
    expect(items).toEqual([
      { code: 'endo', title: 'Endometriosis', body: 'A pain diary', logs: 'pain location' },
      { code: 'pcos', title: null, body: null, logs: null },
    ]);
  });
});

describe('conditionNudgesSchema', () => {
  const nudge = {
    key: 'heavy_pain',
    program: 'endo',
    title: 'Strong pain on several days',
    body: 'You logged pain of 7 or more on 2 days this cycle.',
    action: 'Open the pain diary',
    link: '/programs/pain',
    doctor: null,
    days: 2,
    dates: ['2026-09-15', '2026-09-16'],
    needs_review: true,
  };

  it('parses the golden shape', () => {
    expect(conditionNudgesSchema.parse({ from: '2026-09-14', to: '2026-09-23', nudges: [nudge] })).toEqual([
      {
        key: 'heavy_pain',
        program: 'endo',
        title: 'Strong pain on several days',
        body: 'You logged pain of 7 or more on 2 days this cycle.',
        action: 'Open the pain diary',
        link: '/programs/pain',
        doctor: null,
        days: 2,
      },
    ]);
  });

  it('never keeps an external link', () => {
    const [n] = conditionNudgesSchema.parse({
      nudges: [{ ...nudge, link: 'https://evil.example', doctor: { action: 'Doctor', link: '//evil' } }],
    });
    expect(n.link).toBeNull();
    expect(n.doctor).toBeNull();
  });

  it('degrades to an empty list', () => {
    expect(conditionNudgesSchema.parse(null)).toEqual([]);
    expect(conditionNudgesSchema.parse({ nudges: [{ key: 'other' }] })).toEqual([]);
  });
});

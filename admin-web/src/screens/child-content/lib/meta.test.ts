import { describe, expect, it } from 'vitest';

import { approxDays, bandsOf, buildMeta, draftOf, groupByVisit, isChildKind, isIncomplete } from './meta';

describe('child catalog meta', () => {
  it('round-trips a seeded vaccine and keeps keys it does not edit', () => {
    const stored = { visit: 'm2', age_months: 2, source: 'epi' };
    const draft = draftOf(stored);
    expect(draft).toMatchObject({ visit: 'm2', ageMonths: '2' });
    const res = buildMeta('vaccines', { ...draft, adminNote: ' check booster ' }, stored);
    expect(res).toEqual({ ok: true, meta: { visit: 'm2', age_months: 2, source: 'epi', admin_note: 'check booster' } });
  });

  it('requires a visit code and an age for vaccines', () => {
    const res = buildMeta('vaccines', { ...draftOf(null), visit: 'Two Months', ageMonths: '2.5' });
    expect(res).toEqual({ ok: false, errors: { visit: 'visit', ageMonths: 'months' } });
    expect(buildMeta('vaccines', draftOf(null))).toEqual({ ok: false, errors: { ageMonths: 'required', visit: 'required' } });
  });

  it('requires a domain for milestones only', () => {
    expect(buildMeta('milestones', { ...draftOf(null), ageMonths: '3' })).toEqual({ ok: false, errors: { domain: 'required' } });
    expect(buildMeta('milestones', { ...draftOf(null), ageMonths: '3', domain: 'motor' })).toEqual({
      ok: true,
      meta: { age_months: 3, domain: 'motor' },
    });
    expect(buildMeta('activities', { ...draftOf(null), ageMonths: '3' })).toEqual({ ok: true, meta: { age_months: 3 } });
  });

  it('validates learn ranges, minutes and the article slug', () => {
    const base = { ...draftOf(null), topic: 'sleep', fromMonths: '6', toMonths: '3', minutes: 'x', articleSlug: 'bad slug' };
    expect(buildMeta('learn', base)).toEqual({ ok: false, errors: { toMonths: 'range', minutes: 'minutes', articleSlug: 'slug' } });
    const ok = buildMeta('learn', { ...base, toMonths: '12', minutes: '4', articleSlug: '', featured: true }, { article_slug: 'old' });
    expect(ok).toEqual({ ok: true, meta: { topic: 'sleep', from_months: 6, to_months: 12, minutes: 4, featured: true } });
  });

  it('flags rows the app would skip', () => {
    expect(isIncomplete('vaccines', { age_months: 0 })).toBe(true);
    expect(isIncomplete('vaccines', { age_months: 0, visit: 'birth' })).toBe(false);
    expect(isIncomplete('learn', { topic: 'play', from_months: 0, to_months: 6, minutes: 3 })).toBe(false);
  });

  it('groups doses into visits in catalog order and lists month bands', () => {
    const rows = [
      { id: 1, meta: { visit: 'birth', age_months: 0 } },
      { id: 2, meta: { visit: 'm2', age_months: 2 } },
      { id: 3, meta: { visit: 'birth', age_months: 0 } },
      { id: 4, meta: null },
    ];
    expect(groupByVisit(rows).map((g) => [g.visit, g.ageMonths, g.rows.map((r) => r.id)])).toEqual([
      ['birth', 0, [1, 3]],
      ['m2', 2, [2]],
      ['', null, [4]],
    ]);
    expect(bandsOf([{ meta: { age_months: 6 } }, { meta: { age_months: 2 } }, { meta: { age_months: 6 } }, { meta: null }])).toEqual([2, 6]);
  });

  it('knows its kinds and approximates days', () => {
    expect(isChildKind('age-notes')).toBe(true);
    expect(isChildKind('child_vaccines')).toBe(false);
    expect(approxDays(2)).toBe(61);
    expect(approxDays(24)).toBe(731);
  });
});

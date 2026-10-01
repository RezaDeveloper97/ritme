import { describe, expect, it } from 'vitest';

import { searchResultsSchema } from './schema';

/* Fixtures follow backend-go/contract/golden/search/{mine_log_insight,education}.json + the OpenAPI example. */

describe('searchResultsSchema', () => {
  it('parses groups, hits and the meta the screen reads', () => {
    const parsed = searchResultsSchema.parse({
      query: 'درد',
      scope: 'all',
      groups: [
        {
          key: 'mine',
          total: 2,
          items: [
            {
              type: 'log_insight',
              id: 'pain',
              title: 'درد',
              subtitle: null,
              route: '/analysis/symptoms?category=pain',
              meta: {
                category: 'pain',
                param: null,
                item: null,
                days: 3,
                window: { kind: 'days', from: '2026-08-25', to: '2026-09-23' },
                peak: { score: 6, date: '2026-09-15', cycle_day: 2 },
              },
            },
            { type: 'log_analysis', id: 'pain', title: 'درد', subtitle: null, route: '/analysis/symptoms', meta: { category: 'pain' } },
          ],
        },
        { key: 'programs', total: 0, items: [] },
        {
          key: 'education',
          total: 12,
          items: [
            {
              type: 'article',
              id: 'period-pain',
              title: 'درد پریود',
              subtitle: null,
              route: '/articles/period-pain',
              meta: { kind: 'article', category: 'body', read_time_minutes: 4 },
            },
          ],
        },
        { key: 'shop', total: 1, items: [] },
      ],
    });
    expect(parsed.groups.map((g) => g.key)).toEqual(['mine', 'programs', 'education']);
    const insight = parsed.groups[0].items[0];
    expect(insight.meta).toEqual({
      category: 'pain',
      days: 3,
      windowKind: 'days',
      windowDays: 30,
      peak: { score: 6, cycleDay: 2 },
      kind: null,
      readTimeMinutes: null,
    });
    expect(parsed.groups[2].items[0].meta.readTimeMinutes).toBe(4);
  });

  it('drops hit types the app does not know yet', () => {
    const parsed = searchResultsSchema.parse({
      query: 'xx',
      scope: 'mine',
      groups: [{ key: 'mine', total: 1, items: [{ type: 'course', id: '1', title: 'x', subtitle: null, route: '/x', meta: {} }] }],
    });
    expect(parsed.groups[0].items).toEqual([]);
  });
});

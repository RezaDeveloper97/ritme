import { beforeEach, describe, expect, it, vi } from 'vitest';

const get = vi.fn();

vi.mock('@/shared/api', async (importOriginal) => {
  const real = await importOriginal<typeof import('@/shared/api')>();
  return { ...real, apiClient: { get } };
});

const { ApiError } = await import('@/shared/api');
const { fetchHubReport, hubReportSchema } = await import('./report');

const fail = (status: number, body: unknown = {}) =>
  new ApiError(`HTTP ${status}`, { response: { data: body, status, headers: {} }, sentToken: 'tok' });

const card = (data: unknown, extra: Record<string, unknown> = {}) => ({ plus: false, locked: false, ready: true, data, days_needed: 0, ...extra });

const report = {
  birth_date: '2026-08-12',
  days_since_birth: 42,
  week: 7,
  cards: {
    epds: card({
      kind: 'full',
      trend: { ready: true, direction: 'down' },
      points: [
        { kind: 'full', taken_on: '2026-09-07', week: 4, total: 11, max: 30, band: 'possible', urgent: false },
        { kind: 'full', taken_on: '2026-09-21', week: 6, total: 8, max: 30, band: 'low', urgent: false },
      ],
      thresholds: { short_elevated: 6, full_possible: 10, full_likely: 13 },
    }),
    bleeding: card({
      cells: [
        { date: '2026-09-22', day: 41, level: 2, code: 'light' },
        { date: '2026-09-23', day: 42, level: null, code: null },
      ],
      logged_days: 6,
      weeks: 3,
      trend: 'decreasing',
      large_clots: false,
      alert_today: true,
    }),
    breastfeeding: card({
      source: 'baby_log',
      child_id: 7,
      logged_days: 7,
      feeds_per_day: 8.5,
      left_percent: 60,
      right_percent: 40,
      days: [{ date: '2026-09-23', count: 9 }],
    }),
    sleep: { plus: true, locked: true, ready: false, data: null, days_needed: null },
    growth: card({
      available: true,
      child: { id: 7, name: 'Sara' },
      latest: { weight: { value: 4.6, percentile: 41.2, z: -0.2, in_band: true }, length: null, head: null },
      verdict: { status: 'discuss', label: 'x' },
      crossings: [{ indicator: 'weight', hint: { lines: 2, label: 'Crossed 2 lines' } }, { broken: true }],
    }),
    weight: card({ points: [{ date: '2026-09-20', kg: 67.1 }], delta_since_first: null, delta_vs_pre_pregnancy: 6.1, from_week: 2 }, { ready: false, days_needed: 1 }),
  },
};

describe('hubReportSchema', () => {
  it('maps every card, EPDS newest first and locked cards without data', () => {
    const r = hubReportSchema.parse(report);
    if (r.kind !== 'report') throw new Error('kind');
    expect(r.week).toBe(7);
    expect(r.cards.epds.data?.trendReady).toBe(true);
    expect(r.cards.epds.data?.history.checks.map((c) => c.total)).toEqual([8, 11]);
    expect(r.cards.epds.data?.history.thresholds.fullLikely).toBe(13);
    expect(r.cards.bleeding.data).toMatchObject({ logged: 6, trend: 'decreasing', alertToday: true, weeks: 3 });
    expect(r.cards.bleeding.data?.cells[1]).toEqual({ date: '2026-09-23', level: null, code: null });
    expect(r.cards.breastfeeding.data).toMatchObject({ source: 'baby_log', childId: 7, feedsPerDay: 8.5, leftPercent: 60 });
    expect(r.cards.sleep).toEqual({ plus: true, locked: true, ready: false, data: null, daysNeeded: null });
    expect(r.cards.growth.data?.latest?.weight).toEqual({ percentile: 41.2, inBand: true });
    expect(r.cards.growth.data?.crossings).toEqual(['Crossed 2 lines']);
    expect(r.cards.weight).toMatchObject({ ready: false, daysNeeded: 1 });
    expect(r.cards.weight.data?.deltaVsPre).toBe(6.1);
  });
});

describe('fetchHubReport', () => {
  beforeEach(() => get.mockReset());

  it('reads GET /analysis/postpartum', async () => {
    get.mockResolvedValueOnce({ data: { success: true, data: report } });
    const r = await fetchHubReport();
    expect(get).toHaveBeenCalledWith('/analysis/postpartum');
    expect(r.kind).toBe('report');
  });

  it('falls back to the EPDS-only read on a 404 (Laravel)', async () => {
    get.mockRejectedValueOnce(fail(404));
    get.mockResolvedValueOnce({
      data: {
        success: true,
        data: { checks: [{ kind: 'full', taken_on: '2026-09-21', week: 6, total: 8, max: 30, urgent: false }], thresholds: { short_elevated: 6, full_possible: 10, full_likely: 13 } },
      },
    });
    const r = await fetchHubReport();
    expect(get).toHaveBeenLastCalledWith('/postpartum/epds', { params: { limit: 26 } });
    expect(r.kind).toBe('legacy');
    if (r.kind === 'legacy') expect(r.epds.checks).toHaveLength(1);
  });

  it('reads a 409 not_postpartum as the inactive state', async () => {
    get.mockRejectedValueOnce(fail(409, { success: false, error_code: 'not_postpartum' }));
    expect(await fetchHubReport()).toEqual({ kind: 'inactive' });
    expect(get).toHaveBeenCalledTimes(1);
  });

  it('rethrows anything else (module off, server error)', async () => {
    get.mockRejectedValueOnce(fail(403, { success: false, error_code: 'module_disabled' }));
    await expect(fetchHubReport()).rejects.toBeInstanceOf(ApiError);
  });
});

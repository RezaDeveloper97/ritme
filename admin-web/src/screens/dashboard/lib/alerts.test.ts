import { describe, expect, it } from 'vitest';

import { alertRows, fromDisplay, sourceHref, toDisplay } from './alerts';

const a = (key: string, over: Partial<Parameters<typeof alertRows>[0][number]> = {}) => ({
  key,
  subject: '',
  state: 'open',
  severity: 'warning',
  text: '',
  day: '2026-11-01',
  value: 0.15,
  threshold: 0.12,
  opened_at: '2026-11-01T10:00:00+03:30',
  ...over,
});

describe('alert rows', () => {
  it('shows OCR and CTR-drop fractions as percent and leaves day counts alone', () => {
    const { open } = alertRows([a('ocr_correction_rate'), a('rui_users', { value: 8, threshold: 7 })]);
    const ocr = open.find((r) => r.key === 'ocr_correction_rate');
    const rui = open.find((r) => r.key === 'rui_users');
    expect([ocr?.value, ocr?.threshold, ocr?.unit]).toEqual([15, 12, 'pct']);
    expect([rui?.value, rui?.threshold, rui?.unit]).toEqual([8, 7, 'days']);
  });

  it('puts critical first, then newest, and keeps resolved separate and newest first', () => {
    const rows = alertRows(
      [
        a('engine_mae', { opened_at: '2026-11-02T09:00:00+03:30' }),
        a('safety_selftest_failed', { severity: 'critical', opened_at: '2026-10-30T09:00:00+03:30' }),
        a('rui_users', { opened_at: '2026-11-01T09:00:00+03:30' }),
      ],
      [a('ocr_correction_rate', { state: 'resolved', opened_at: '2026-10-01T00:00:00+03:30' }), a('engine_mae', { state: 'resolved', opened_at: '2026-10-20T00:00:00+03:30' })],
    );
    expect(rows.open.map((r) => r.key)).toEqual(['safety_selftest_failed', 'engine_mae', 'rui_users']);
    expect(rows.resolved.map((r) => r.key)).toEqual(['engine_mae', 'ocr_correction_rate']);
    expect(rows.resolved.every((r) => r.resolved)).toBe(true);
  });

  it('links each alert to its source screen, a CTR alert to the rule behind its subject', () => {
    expect(sourceHref('message_ctr', 'r12v3')).toBe('/messages/rules/12');
    expect(sourceHref('message_ctr', '')).toBe('/messages/rules');
    expect(sourceHref('safety_selftest_failed', '')).toBe('/safety');
    expect(sourceHref('engine_mae', '')).toBe('/kpi');
  });

  it('round-trips the rule editor value', () => {
    expect(toDisplay('ocr_correction_rate', 0.12)).toBe(12);
    expect(fromDisplay('ocr_correction_rate', 12.5)).toBe(0.125);
    expect(fromDisplay('engine_mae', 2.5)).toBe(2.5);
  });
});

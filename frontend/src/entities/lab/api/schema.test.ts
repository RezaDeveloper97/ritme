import { existsSync, readFileSync } from 'node:fs';
import { resolve } from 'node:path';

import { describe, expect, it } from 'vitest';

import { toMarkerBody } from './queries';
import { catalogSchema, consentSchema, labListSchema, labSchema, markerDetailSchema, statusSchema, trendsSchema } from './schema';

/* Boundary contract of /api/v1/labs/* against the Go contract goldens (B-N6-06). */
const GOLDEN_DIR = resolve(process.cwd(), '../backend-go/contract/golden/labs');
interface Golden {
  steps: { request: { method: string; url: string }; status: number; body: { data?: unknown } }[];
}
const golden = (name: string): Golden =>
  JSON.parse(readFileSync(resolve(GOLDEN_DIR, `${name}.json`), 'utf8')) as Golden;
const stepData = (name: string, method: string, url: string, nth = 0): unknown =>
  golden(name).steps.filter((s) => s.request.method === method && s.request.url === url)[nth]?.body.data;

describe.skipIf(!existsSync(GOLDEN_DIR))('lab parsers', () => {
  it('parses the uploaded lab in review with a low-confidence marker', () => {
    const lab = labSchema.parse(stepData('upload_verify_result.fa', 'POST', '/api/v1/labs'));
    expect(lab).toMatchObject({ id: 1, status: 'needs_review', stage: 'review', editable: true, lowConfidenceCount: 1 });
    expect(lab.counts).toMatchObject({ total: 5, attention: 3, low: 2, borderlineLow: 1 });
    expect(lab.markers).toHaveLength(5);
    const fbs = lab.markers.find((m) => m.code === 'glucose_fasting')!;
    expect(fbs).toMatchObject({ lowConfidence: true, printedName: 'FBS', state: 'normal' });
    expect(fbs.reference).toEqual({ low: 70, high: 100, text: '70–100', source: 'sheet', label: null, note: null });
    expect(fbs).toMatchObject({ needsConfirmation: false, nearEdge: false, band: null, confirmedAt: null });
    expect(lab.interpretation).toBeNull();
  });

  it('parses the verified lab with its interpretation', () => {
    const lab = labSchema.parse(stepData('upload_verify_result.fa', 'POST', '/api/v1/labs/1/verify'));
    expect(lab.status).toBe('ready');
    expect(lab.interpretation?.source).toBe('ai');
    expect(lab.interpretation?.doctorQuestions.length).toBeGreaterThan(0);
    expect(lab.interpretation?.disclaimer).toBeTruthy();
    expect(lab.interpretation?.redFlags).toEqual([]);
  });

  it('parses status, history, marker detail and trends', () => {
    const st = statusSchema.parse(stepData('upload_verify_result.fa', 'GET', '/api/v1/labs/1/status'));
    expect(st).toMatchObject({ status: 'needs_review', stage: 'review', progress: 100, markerCount: 5 });
    const list = labListSchema.parse(stepData('upload_verify_result.fa', 'GET', '/api/v1/labs'));
    expect(list.labs[0]).toMatchObject({ id: 1, status: 'ready', markerCount: 5, attentionCount: 3, allNormal: false });
    expect(list.limits).toEqual({ maxFiles: 5, maxImageKb: 5120, maxPdfKb: 10240, categories: ['blood', 'hormone', 'thyroid', 'urine', 'other'] });
    const empty = labListSchema.parse(stepData('labs_empty.en', 'GET', '/api/v1/labs'));
    expect(empty.labs).toEqual([]);
    const d = markerDetailSchema.parse(stepData('upload_verify_result.fa', 'GET', '/api/v1/labs/1/markers/2'));
    expect(d.marker).toMatchObject({ code: 'ferritin', state: 'low', value: 9 });
    expect(d.about?.body).toBeTruthy();
    expect(d.factors).toHaveLength(3);
    expect(d.contextNotes).toHaveLength(1);
    expect(d.trend.points).toHaveLength(1);
    const t = trendsSchema.parse(stepData('upload_verify_result.fa', 'GET', '/api/v1/labs/trends'));
    expect(t.labsCount).toBe(1);
    expect(t.markers[0]).toMatchObject({ key: 'hemoglobin', latest: { labId: 1, markerId: 1, state: 'borderline_low' } });
  });

  it('parses a failed upload with its message', () => {
    const lab = labSchema.parse(stepData('upload_failed.fa', 'POST', '/api/v1/labs'));
    expect(lab).toMatchObject({ status: 'failed', stage: 'failed', errorCode: 'ai_failed', editable: false });
    expect(lab.errorMessage).toBeTruthy();
  });

  it('parses the catalog and the consent', () => {
    const cat = catalogSchema.parse(stepData('catalog.fa', 'GET', '/api/v1/labs/markers'));
    expect(cat.length).toBeGreaterThan(10);
    expect(cat[0]!.code).toBeTruthy();
    const c = consentSchema.parse(stepData('upload_failed.fa', 'PUT', '/api/v1/consents/ai_lab_analysis'));
    expect(c).toMatchObject({ code: 'ai_lab_analysis', version: 1, granted: true, needsConsent: false });
    expect(c.points.length).toBeGreaterThan(0);
  });

  it('parses the v6 fields: confirmation, band, general range and the JSON summary sections (D-171)', () => {
    const base = stepData('upload_verify_result.fa', 'POST', '/api/v1/labs/1/verify') as Record<string, unknown>;
    const markers = (base.markers as Record<string, unknown>[]).map((m, i) =>
      i === 0
        ? {
            ...m,
            needs_confirmation: true,
            confirmed_at: null,
            near_edge: true,
            band: { code: 'vitd_insufficient', label: 'کمتر از حد کافی', note: 'یادداشت' },
            reference: { low: 20, high: 100, text: '20–100', source: 'typical', label: 'محدوده عمومی', note: null },
          }
        : m,
    );
    const lab = labSchema.parse({
      ...base,
      markers,
      unconfirmed_count: 1,
      interpretation: {
        ...(base.interpretation as Record<string, unknown>),
        out_of_range: [
          { marker_id: 2, code: 'ferritin', name: 'فریتین', state: 'low', state_label: 'پایین', explanation: 'ممکن است مرتبط باشد.' },
          { marker_id: 1, code: null, name: 'Hb', state: 'bogus', state_label: 'x', explanation: '  ' },
          null,
        ],
      },
    });
    expect(lab.unconfirmedCount).toBe(1);
    expect(lab.markers[0]).toMatchObject({
      needsConfirmation: true,
      nearEdge: true,
      band: { code: 'vitd_insufficient', label: 'کمتر از حد کافی', note: 'یادداشت' },
      reference: { source: 'typical', label: 'محدوده عمومی', note: null },
    });
    expect(lab.interpretation?.outOfRange).toEqual([
      { markerId: 2, code: 'ferritin', name: 'فریتین', state: 'low', stateLabel: 'پایین', explanation: 'ممکن است مرتبط باشد.' },
      { markerId: 1, code: null, name: 'Hb', state: 'unknown', stateLabel: 'x', explanation: null },
    ]);
  });

  it('sends a full marker body (PUT replaces the row)', () => {
    expect(
      toMarkerBody({ name: ' FBS ', value: 95, valueText: 'x', unit: 'mg/dL', refLow: 70, refHigh: 100, refText: null }),
    ).toEqual({ name: 'FBS', value: 95, value_text: null, unit: 'mg/dL', ref_low: 70, ref_high: 100, ref_text: null });
  });
});

import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';

import { describe, expect, it } from 'vitest';

import {
  claimBadge,
  reviewDraftOf,
  reviewFieldsBody,
  reviewItemsBody,
  reviewItemsOf,
  safeFileUrl,
  stayDays,
} from '../model/documents-lib';
import {
  datingOfferSchema,
  docConsentSchema,
  recordCategoriesSchema,
  recordDocumentSchema,
  recordExtrasSchema,
  reviewResultSchema,
  timelinePageSchema,
} from './documents-schema';

/* Boundary contract of the record documents API (CB-REC-01/02) against the Go goldens. */
const GOLDEN = resolve(process.cwd(), '../backend-go/contract/golden');
interface Step {
  request: { method: string; url: string };
  status: number;
  body: { data?: unknown };
}
const steps = (group: string, name: string): Step[] =>
  (JSON.parse(readFileSync(resolve(GOLDEN, group, `${name}.json`), 'utf8')) as { steps: Step[] }).steps;
const step = (group: string, name: string, method: string, url: string, status = 200): Step => {
  const s = steps(group, name).find((x) => x.request.method === method && x.request.url === url && x.status === status);
  if (!s) throw new Error(`no ${method} ${url} ${status} in ${name}`);
  return s;
};

describe('record documents schemas', () => {
  it('parses the categories, timeline and extras', () => {
    const c = recordCategoriesSchema.parse(step('healthrecord-documents', 'documents_flow', 'GET', '/api/v1/health-record/categories').body.data);
    expect(c.counts.imaging).toBe(1);
    expect(c.allCount).toBeGreaterThanOrEqual(1);
    const tl = timelinePageSchema.parse(step('healthrecord-documents', 'documents_flow', 'GET', '/api/v1/health-record/timeline').body.data);
    expect(tl.months[0]).toMatchObject({ key: '1405-06', jalaliYear: 1405, jalaliMonth: 6 });
    expect(tl.months[0]?.items[0]).toMatchObject({ type: 'document', kind: 'imaging', dateKnown: true, links: [] });
    expect(tl.nextBefore).toBeNull();
    const ex = recordExtrasSchema.parse(steps('healthrecord-documents', 'extras_flow')[0]?.body.data);
    expect(ex.allergiesOnEmergencyCard).toBe(true);
  });

  it('parses a document with its extraction, review answer and dating offer', () => {
    const doc = recordDocumentSchema.parse(step('healthrecord-extract', 'dating_apply.fa', 'GET', '/api/v1/health-record/documents/1').body.data);
    expect(doc.kind).toBe('imaging');
    expect(doc.reviewState).toBe('confirmed');
    expect(doc.whereUsed).toEqual([expect.objectContaining({ type: 'pregnancy', state: 'applied' })]);
    expect(doc.extracted?.fields.ga_weeks?.value).toBe(12);
    const reviewed = reviewResultSchema.parse(step('healthrecord-extract', 'dating_apply.fa', 'POST', '/api/v1/health-record/documents/1/review').body.data);
    expect(reviewed.dating.state).toBe('offered');
    expect(reviewed.dating.proposed).toMatchObject({ gaWeeks: 12, gaDays: 3, dueDate: '2027-03-30' });
    const applied = datingOfferSchema.parse(step('healthrecord-extract', 'dating_apply.fa', 'POST', '/api/v1/health-record/documents/1/dating').body.data);
    expect(applied.state).toBe('applied');
  });

  it('parses the prescription rows and the consent', () => {
    const doc = recordDocumentSchema.parse(step('healthrecord-extract', 'extract_prescription', 'POST', '/api/v1/health-record/documents/1/extract', 202).body.data);
    expect(doc.extracted?.items).toHaveLength(2);
    expect(reviewItemsOf(doc.extracted)[0]?.medicine).toBe('Ferrous sulfate 50mg');
    const consent = docConsentSchema.parse(step('healthrecord-extract', 'extract_prescription', 'PUT', '/api/v1/consents/ai_documents').body.data);
    expect(consent).toMatchObject({ code: 'ai_documents', granted: true, needsConsent: false });
  });
});

describe('record documents helpers', () => {
  it('drafts the review form and builds its body', () => {
    const doc = recordDocumentSchema.parse(step('healthrecord-extract', 'extract_prescription', 'POST', '/api/v1/health-record/documents/1/extract', 202).body.data);
    const draft = reviewDraftOf('prescription', doc.extracted);
    expect(draft).toMatchObject({ date: '2026-09-15', doctor: 'دکتر نمونه' });
    expect(reviewFieldsBody('prescription', { ...draft, centre: '  ' })).toMatchObject({ centre: null, date: '2026-09-15' });
    expect(reviewFieldsBody('imaging', { ga_weeks: '۱۲', ga_days: '' })).toMatchObject({ ga_weeks: 12, ga_days: null });
    expect(reviewItemsBody([{ medicine: ' A ', dose: '', frequency: '', duration: '' }, { medicine: '', dose: '', frequency: '', duration: '' }])).toEqual([
      { medicine: 'A', dose: null, frequency: null, duration: null },
    ]);
  });

  it('keeps only signed links of this API over https', () => {
    const prod = 'https://api.ritme.app/api/v1';
    const ok = 'https://api.ritme.app/api/v1/files/7/download?expires=1&signature=abc';
    expect(safeFileUrl(ok, prod)).toBe(ok);
    expect(safeFileUrl('http://api.ritme.app/api/v1/files/7/download', prod)).toBeNull();
    expect(safeFileUrl('https://evil.example/api/v1/files/7/download', prod)).toBeNull();
    expect(safeFileUrl('https://api.ritme.app/api/v1/labs/7', prod)).toBeNull();
    expect(safeFileUrl('javascript:alert(1)', prod)).toBeNull();
    expect(safeFileUrl('not a url', prod)).toBeNull();
    expect(safeFileUrl(null, prod)).toBeNull();
    const dev = 'http://127.0.0.1:8251/api/v1';
    expect(safeFileUrl('http://127.0.0.1:8251/api/v1/files/1/download?x=1', dev)).toBe('http://127.0.0.1:8251/api/v1/files/1/download?x=1');
    expect(safeFileUrl('http://localhost/api/v1/files/1/download', dev)).toBeNull();
  });

  it('derives badges and stays', () => {
    expect(claimBadge([{ type: 'pregnancy', state: 'applied' }])).toBeNull();
    expect(claimBadge([{ type: 'claim', state: 'attached' }, { type: 'claim', state: 'waiting' }])).toBe('waiting');
    expect(stayDays('2025-03-04', '2025-03-06')).toBe(3);
    expect(stayDays('2025-03-04', null)).toBeNull();
  });
});

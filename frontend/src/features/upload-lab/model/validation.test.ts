import { describe, expect, it } from 'vitest';

import { ApiError } from '@/shared/api';

import { uploadErrorOf } from './errors';
import {
  addPages,
  checkFile,
  DEFAULT_UPLOAD_LIMITS,
  type DraftPage,
  hasPdfHeader,
  kindOf,
  limitsFrom,
  movePage,
  removePage,
  toFormData,
  validateDraft,
} from './validation';

const MB = 1024 * 1024;
const file = (name: string, type: string, size: number): File => {
  const f = new File(['x'], name, { type });
  Object.defineProperty(f, 'size', { value: size });
  return f;
};
let n = 0;
const id = () => `p${++n}`;
const L = DEFAULT_UPLOAD_LIMITS;

describe('lab upload validation (mirrors internal/labs/upload.go)', () => {
  it('accepts JPEG / PNG / WebP photos and PDFs only', () => {
    expect(kindOf({ name: 'a.jpg', type: 'image/jpeg' })).toBe('image');
    expect(kindOf({ name: 'a', type: 'image/webp' })).toBe('image');
    expect(kindOf({ name: 'a.pdf', type: 'application/pdf' })).toBe('pdf');
    expect(kindOf({ name: 'scan.PDF', type: '' })).toBe('pdf');
    expect(kindOf({ name: 'a.heic', type: 'image/heic' })).toBeNull();
    expect(kindOf({ name: 'a.gif', type: 'image/gif' })).toBeNull();
    expect(kindOf({ name: 'a.txt', type: '' })).toBeNull();
  });

  it('enforces the per-type size limits', () => {
    expect(checkFile(file('a.jpg', 'image/jpeg', 5 * MB), L)).toBeNull();
    expect(checkFile(file('a.jpg', 'image/jpeg', 5 * MB + 1), L)).toBe('image_too_large');
    expect(checkFile(file('a.pdf', 'application/pdf', 10 * MB), L)).toBeNull();
    expect(checkFile(file('a.pdf', 'application/pdf', 10 * MB + 1), L)).toBe('pdf_too_large');
    expect(checkFile(file('a.jpg', 'image/jpeg', 0), L)).toBe('empty_file');
    expect(checkFile(file('a.doc', 'application/msword', 10), L)).toBe('file_type');
  });

  it('reads the limits from GET /labs', () => {
    expect(limitsFrom({ maxFiles: 3, maxImageKb: 100, maxPdfKb: 200, categories: ['blood'] })).toMatchObject({
      maxFiles: 3, maxImageBytes: 100 * 1024, maxPdfBytes: 200 * 1024, categories: ['blood'],
    });
    expect(limitsFrom(undefined)).toBe(L);
  });

  it('adds at most five pages and reports the first refusal', () => {
    const six = Array.from({ length: 6 }, (_, i) => file(`p${i}.jpg`, 'image/jpeg', MB));
    const r = addPages([], six, L, id);
    expect(r.pages).toHaveLength(5);
    expect(r.error).toBe('too_many_files');
    const mixed = addPages([], [file('a.gif', 'image/gif', 1), file('b.png', 'image/png', 1)], L, id);
    expect(mixed.pages.map((p) => p.file.name)).toEqual(['b.png']);
    expect(mixed.error).toBe('file_type');
  });

  it('reorders and removes pages', () => {
    const pages: DraftPage[] = ['a', 'b', 'c'].map((x) => ({ id: x, file: file(`${x}.jpg`, 'image/jpeg', 1), kind: 'image' }));
    expect(movePage(pages, 'b', -1).map((p) => p.id)).toEqual(['b', 'a', 'c']);
    expect(movePage(pages, 'c', 1).map((p) => p.id)).toEqual(['a', 'b', 'c']);
    expect(removePage(pages, 'a').map((p) => p.id)).toEqual(['b', 'c']);
  });

  it('validates the whole draft before upload', () => {
    const page: DraftPage = { id: 'x', file: file('a.jpg', 'image/jpeg', MB), kind: 'image' };
    expect(validateDraft({ pages: [], category: 'blood', takenOn: null, fasting: null }, L, '2026-10-06')).toEqual({ files: 'files_required' });
    expect(validateDraft({ pages: [page], category: null, takenOn: null, fasting: null }, L, '2026-10-06')).toEqual({ category: 'category_invalid' });
    expect(validateDraft({ pages: [page], category: 'blood', takenOn: '2026-10-07', fasting: true }, L, '2026-10-06')).toEqual({ taken_on: 'date_future' });
    expect(validateDraft({ pages: [page], category: 'blood', takenOn: '2026-10-06', fasting: true }, L, '2026-10-06')).toEqual({});
    const big = Array.from({ length: 2 }, (_, i): DraftPage => ({ id: `${i}`, file: file('a.pdf', 'application/pdf', 10 * MB), kind: 'pdf' }));
    expect(validateDraft({ pages: big, category: 'blood', takenOn: null, fasting: null }, L, '2026-10-06').files).toBe('upload_too_large');
  });

  it('builds the multipart body without the device file names', () => {
    const page: DraftPage = { id: 'x', file: file('Maryam-Ahmadi.jpg', 'image/jpeg', 10), kind: 'image' };
    const form = toFormData({ pages: [page], category: 'thyroid', takenOn: '2026-10-01', fasting: false });
    expect((form.getAll('files[]')[0] as File).name).toBe('page-1.jpg');
    expect(form.get('category')).toBe('thyroid');
    expect(form.get('taken_on')).toBe('2026-10-01');
    expect(form.get('fasting')).toBe('0');
    const bare = toFormData({ pages: [page], category: 'blood', takenOn: null, fasting: null });
    expect(bare.has('taken_on')).toBe(false);
    expect(bare.has('fasting')).toBe(false);
  });

  it('checks the PDF header like the server', () => {
    const enc = (s: string) => new TextEncoder().encode(s);
    expect(hasPdfHeader(enc('%PDF-1.7\n'))).toBe(true);
    expect(hasPdfHeader(enc('\n\n%PDF-1.4'))).toBe(true);
    expect(hasPdfHeader(enc('0123456789%PDF-1.4'))).toBe(false);
    expect(hasPdfHeader(enc('<html>'))).toBe(false);
  });
});

describe('upload error mapping', () => {
  const err = (status: number, data: unknown) =>
    new ApiError('x', { response: { status, data, headers: {} }, sentToken: null });

  it('maps the AI gate refusals', () => {
    expect(uploadErrorOf(err(402, { error_code: 'plus_required', reason: 'locked', feature: 'plus.lab_ai' })).kind).toBe('plus');
    expect(uploadErrorOf(err(429, { error_code: 'plus_quota_exceeded', feature: 'plus.lab_ai', limit: 10 })).denial?.kind).toBe('exhausted');
    expect(uploadErrorOf(err(403, { error_code: 'consent_required', consent: 'ai_lab_analysis', version: 1 })).kind).toBe('consent');
    expect(uploadErrorOf(err(429, { error_code: 'ai_busy', message: 'busy' }))).toMatchObject({ kind: 'busy', message: 'busy' });
    expect(uploadErrorOf(err(503, { error_code: 'lab_busy' })).kind).toBe('unavailable');
    expect(uploadErrorOf(new ApiError('t', { code: 'timeout', sentToken: null })).kind).toBe('network');
    expect(uploadErrorOf(new Error('x')).kind).toBe('unknown');
  });

  it('keeps the first message per field of a 422', () => {
    const e = uploadErrorOf(err(422, { message: 'invalid', errors: { files: ['bad type', 'x'], category: ['bad'] } }));
    expect(e).toMatchObject({ kind: 'validation', fields: { files: 'bad type', category: 'bad' } });
  });
});

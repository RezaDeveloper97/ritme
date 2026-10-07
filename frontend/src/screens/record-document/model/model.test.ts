import { describe, expect, it } from 'vitest';

import type { Extraction } from '@/entities/health-record';
import { ApiError } from '@/shared/api';

import { extractErrorOf } from './errors';
import { confirmedRows, sizeOf } from './format';

const failed = (status: number, data: unknown) =>
  new ApiError(`Request failed with status code ${status}`, { response: { status, data, headers: {} }, sentToken: null });

describe('extractErrorOf', () => {
  it('maps the AI gate answers', () => {
    expect(extractErrorOf(failed(402, { error_code: 'plus_required' }))).toBe('plus');
    expect(extractErrorOf(failed(429, { error_code: 'plus_quota_exceeded' }))).toBe('plus');
    expect(extractErrorOf(failed(403, { error_code: 'consent_required' }))).toBe('consent');
    expect(extractErrorOf(failed(422, { errors: { file_ids: ['x'] } }))).toBe('noFiles');
    expect(extractErrorOf(failed(409, { error_code: 'extraction_running' }))).toBe('conflict');
    expect(extractErrorOf(failed(429, { error_code: 'ai_busy' }))).toBe('busy');
    expect(extractErrorOf(failed(503, { error_code: 'ai_unavailable' }))).toBe('busy');
    expect(extractErrorOf(new Error('x'))).toBe('unknown');
  });
});

describe('confirmedRows', () => {
  const e = (reviewed: Extraction['reviewed']): Extraction => ({
    schema: 'imaging',
    status: 'done',
    errorCode: null,
    fields: {},
    items: [],
    reviewed,
    reviewedItems: null,
    dating: null,
  });

  it('merges the gestational age and drops empty values', () => {
    expect(confirmedRows('imaging', e({ centre: 'C', doctor: null, ga_weeks: 8, ga_days: 6, edd: '2027-05-09' }))).toEqual([
      { key: 'centre', value: 'C' },
      { key: 'ga', weeks: 8, days: 6 },
      { key: 'edd', value: '2027-05-09' },
    ]);
    expect(confirmedRows('imaging', null)).toEqual([]);
  });

  it('sizes files', () => {
    expect(sizeOf(2048)).toEqual({ unit: 'kb', value: 2 });
    expect(sizeOf(1.5 * 1024 * 1024)).toEqual({ unit: 'mb', value: 1.5 });
  });
});

import { describe, expect, it } from 'vitest';

import { ApiError } from '@/shared/api';

import { customItemError } from './errors';

function failed(status: number, data: unknown): ApiError {
  return new ApiError(`Request failed with status code ${status}`, { response: { status, data, headers: {} }, sentToken: null });
}

describe('customItemError', () => {
  it('maps 422 fields onto our copy', () => {
    expect(customItemError(failed(422, { message: 'x', errors: { label: ['taken'] } }))).toBe('duplicate');
    expect(customItemError(failed(422, { message: 'x', errors: { custom_items: ['max'] } }))).toBe('limit');
    expect(customItemError(failed(422, { message: 'x', errors: { category: ['in'] } }))).toBe('category');
    expect(customItemError(failed(422, { message: 'x' }))).toBe('generic');
  });
  it('tells throttling and anything else apart', () => {
    expect(customItemError(failed(429, {}))).toBe('throttled');
    expect(customItemError(failed(500, {}))).toBe('generic');
    expect(customItemError(new Error('offline'))).toBe('generic');
  });
});

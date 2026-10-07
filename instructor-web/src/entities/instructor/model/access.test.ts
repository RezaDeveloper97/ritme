import { describe, expect, it } from 'vitest';

import { accessOf, initialOf, pathForAccess } from './access';
import { meSchema, type Instructor } from './schema';

const base: Instructor = {
  id: 1,
  display_name: 'لیلا مهدوی',
  title: 'ماما',
  bio: null,
  status: 'pending',
  approved_at: null,
  created_at: '2026-10-01T08:00:00+03:30',
};

describe('instructor access', () => {
  it('maps /me to a route', () => {
    expect(accessOf(null)).toBe('required');
    expect(pathForAccess(accessOf(null))).toBe('/apply');
    expect(pathForAccess(accessOf(base))).toBe('/pending');
    expect(pathForAccess(accessOf({ ...base, status: 'approved' }))).toBe('/');
    expect(pathForAccess(accessOf({ ...base, status: 'revoked' }))).toBe('/apply');
  });
  it('takes the first letter for the avatar', () => {
    expect(initialOf('  لیلا')).toBe('ل');
    expect(initialOf('')).toBe('');
  });
  it('parses the API shape', () => {
    expect(meSchema.parse({ instructor: null })).toEqual({ instructor: null });
    expect(meSchema.parse({ instructor: base }).instructor?.status).toBe('pending');
    expect(() => meSchema.parse({ instructor: { ...base, status: 'nope' } })).toThrow();
  });
});

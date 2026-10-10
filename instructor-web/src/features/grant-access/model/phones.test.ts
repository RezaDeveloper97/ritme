import { describe, expect, it } from 'vitest';

import { chunk, mergePhones, parseCsv, parsePhones } from './phones';

describe('parsePhones', () => {
  it('normalises Persian digits, +98 and spacing, dedupes and flags junk', () => {
    const r = parsePhones('۰۹۱۲ ۳۴۵ ۶۷۸۹\n+98 912 345 6789\n0935-111-2233\nabc\n12345');
    expect(r.valid).toEqual(['09123456789', '09351112233']);
    expect(r.duplicates).toBe(1);
    expect(r.invalid).toEqual(['abc', '12345']);
  });
});

describe('parseCsv', () => {
  it('skips a header row and takes the first phone cell per row', () => {
    const r = parseCsv('﻿name,phone\nمریم,"09123456789"\nسارا;0935 111 2233\nبدون شماره,x');
    expect(r.valid).toEqual(['09123456789', '09351112233']);
    expect(r.invalid).toEqual(['بدون شماره']);
  });
  it('keeps a header-less first row', () => {
    expect(parseCsv('09123456789\n09351112233').valid).toHaveLength(2);
  });
});

describe('chunk / merge', () => {
  it('splits into requests of at most 200', () => {
    expect(chunk(Array.from({ length: 450 }, (_, i) => i)).map((c) => c.length)).toEqual([200, 200, 50]);
  });
  it('merges typed and file numbers without repeats', () => {
    const r = mergePhones(parsePhones('09123456789'), parseCsv('09123456789\n09351112233'));
    expect(r.valid).toEqual(['09123456789', '09351112233']);
    expect(r.duplicates).toBe(1);
  });
});

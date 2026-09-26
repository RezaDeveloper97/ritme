import { describe, expect, it } from 'vitest';

import { emptyForm, formFromPrefill, parseKind, parsePrefillDate, prepFromText, validateForm } from './form';

describe('appointment form model', () => {
  it('parses ?kind=', () => {
    expect(parseKind('phone')).toBe('phone');
    expect(parseKind('bogus')).toBe('in_person');
    expect(parseKind(undefined)).toBe('in_person');
  });

  it('maps textarea lines to prep items, keeping existing ticks', () => {
    let n = 0;
    const items = prepFromText(' a \n\nb\n', [{ id: 'x', text: 'b', done: true }], () => `n${++n}`);
    expect(items).toEqual([
      { id: 'n1', text: 'a', done: false },
      { id: 'x', text: 'b', done: true },
    ]);
  });

  it('validates the required fields', () => {
    const f = emptyForm('online');
    expect(validateForm(f)).toBe('with');
    expect(validateForm({ ...f, withWhom: 'دکتر' })).toBe('date');
    expect(validateForm({ ...f, withWhom: 'دکتر', date: '2026-10-01' })).toBe('time');
    expect(validateForm({ ...f, withWhom: 'دکتر', date: '2026-10-01', time: '09:30' })).toBeNull();
  });

  it('prefills title, date and care_item_key from the query', () => {
    const f = formFromPrefill(
      { kind: 'phone', title: ' NT scan ', date: '2026-10-05', careItemKey: 'nt_scan' },
      '2026-09-26',
    );
    expect(f.kind).toBe('phone');
    expect(f.title).toBe('NT scan');
    expect(f.date).toBe('2026-10-05');
    expect(f.careItemKey).toBe('nt_scan');
    expect(formFromPrefill({}, '2026-09-26')).toEqual(emptyForm('in_person'));
    expect(formFromPrefill({ careItemKey: 'bad key!' }, '2026-09-26').careItemKey).toBe('');
  });

  it('accepts only real, non-past YYYY-MM-DD dates', () => {
    const today = '2026-09-26';
    expect(parsePrefillDate('2026-09-26', today)).toBe('2026-09-26');
    expect(parsePrefillDate('2026-09-25', today)).toBe('');
    expect(parsePrefillDate('2026-02-30', today)).toBe('');
    expect(parsePrefillDate('26-10-01', today)).toBe('');
    expect(parsePrefillDate(undefined, today)).toBe('');
  });
});

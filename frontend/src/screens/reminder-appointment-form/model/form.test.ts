import { describe, expect, it } from 'vitest';

import { emptyForm, parseKind, prepFromText, validateForm } from './form';

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
});

import { describe, expect, it } from 'vitest';

import {
  emptyForm,
  formFromPrefill,
  parseKind,
  parsePrefillDate,
  parseReturnTo,
  parseTopic,
  prepFromText,
  returnPathFor,
  validateForm,
} from './form';

describe('appointment form model', () => {
  it('parses ?kind=', () => {
    expect(parseKind('phone')).toBe('phone');
    expect(parseKind('bogus')).toBe('in_person');
    expect(parseKind(undefined)).toBe('in_person');
  });

  it('parses ?topic= against the known topics', () => {
    expect(parseTopic('ultrasound')).toBe('ultrasound');
    expect(parseTopic('vaccine')).toBe('vaccine');
    expect(parseTopic('bogus')).toBe('checkup');
    expect(parseTopic(null)).toBe('checkup');
    // The NT «رزرو» link: in person, about an ultrasound.
    const nt = formFromPrefill({ kind: 'in_person', topic: 'ultrasound', careItemKey: 'nt_scan' }, '2026-09-26');
    expect(nt.kind).toBe('in_person');
    expect(nt.topic).toBe('ultrasound');
    expect(formFromPrefill({ topic: '<script>' }, '2026-09-26').topic).toBe('checkup');
  });

  it('maps textarea lines to prep items, keeping existing ticks', () => {
    let n = 0;
    const items = prepFromText(' a \n\nb\n', [{ id: 'x', text: 'b', done: true }], () => `n${++n}`);
    expect(items).toEqual([
      { id: 'n1', text: 'a', done: false },
      { id: 'x', text: 'b', done: true },
    ]);
  });

  it('defaults add-to-calendar on, as in the artboard', () => {
    expect(emptyForm('in_person').addToCalendar).toBe(true);
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

describe('return after save (9b, T-M7-20)', () => {
  it('only accepts allow-listed in-app paths', () => {
    expect(parseReturnTo('/pregnancy/calendar')).toBe('/pregnancy/calendar');
    for (const bad of ['https://evil.example', '//evil.example', '/pregnancy/calendar/../x', '/profile', '', null, undefined]) {
      expect(parseReturnTo(bad)).toBeNull();
    }
  });

  it('a new care-plan booking returns to the pregnancy calendar', () => {
    expect(returnPathFor(null, true, 'nt_scan')).toBe('/pregnancy/calendar');
    expect(returnPathFor('/pregnancy/calendar', false, '')).toBe('/pregnancy/calendar');
    expect(returnPathFor(null, false, 'nt_scan')).toBeNull();
    expect(returnPathFor('https://evil.example', true, '')).toBeNull();
  });
});

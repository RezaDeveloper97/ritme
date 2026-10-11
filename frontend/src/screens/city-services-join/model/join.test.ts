import { describe, expect, it } from 'vitest';

import {
  emptyOffer,
  fileProblem,
  foldDigits,
  groupErrors,
  offerFromForm,
  offerProblems,
  pickApplication,
  resumeStep,
  stepOfField,
  toInt,
  validHours,
} from './join';

describe('join helpers', () => {
  it('folds Persian digits and separators', () => {
    expect(foldDigits('۰۹۱۲ ۱۲۳-۴۵۶۷'.replace('-', ''))).toBe('09121234567');
    expect(toInt('۳۲۰٬۰۰۰')).toBe(320000);
    expect(toInt('')).toBeNull();
    expect(toInt('12a')).toBeNull();
  });

  it('maps submit errors to their step', () => {
    expect(stepOfField('city_id')).toBe(1);
    expect(stepOfField('category_slugs.0')).toBe(1);
    expect(stepOfField('location')).toBe(2);
    expect(stepOfField('photos')).toBe(2);
    expect(stepOfField('services')).toBe(3);
    expect(stepOfField('documents.licence')).toBe(4);
    expect(stepOfField('terms')).toBe(4);
    expect(groupErrors({ terms: ['t'], city_id: ['c'] }).map((e) => e.step)).toEqual([1, 4]);
  });

  it('resumes after the last saved step', () => {
    expect(resumeStep({ step: 1, business_name: null })).toBe(1);
    expect(resumeStep({ step: 2, business_name: 'x' })).toBe(3);
    expect(resumeStep({ step: 4, business_name: 'x' })).toBe(4);
  });

  it('picks the open draft before a sent application', () => {
    const row = (id: number, editable: boolean) => ({ id, editable }) as never;
    expect(pickApplication([row(3, false), row(2, true)])).toEqual({ kind: 'open', id: 2 });
    expect(pickApplication([row(3, false)])).toEqual({ kind: 'sent', id: 3 });
    expect(pickApplication([])).toEqual({ kind: 'none', id: null });
  });

  it('checks hours and files', () => {
    expect(validHours({ day: 0, open: '09:00', close: '20:00' })).toBe(true);
    expect(validHours({ day: 0, open: '20:00', close: '09:00' })).toBe(false);
    expect(fileProblem({ type: 'image/gif', size: 10, name: 'a.gif' }, 'photo')).toBe('fileType');
    expect(fileProblem({ type: 'image/png', size: 6 * 1024 * 1024, name: 'a.png' }, 'photo')).toBe('fileSize');
    expect(fileProblem({ type: '', size: 10, name: 'x.PDF' }, 'doc')).toBeNull();
  });

  it('validates and converts a service', () => {
    const f = { ...emptyOffer(), name: ' Swim ', price: '۳۲۰۰۰۰', duration: '45' };
    expect(offerProblems(f)).toEqual([]);
    expect(offerFromForm(f)).toMatchObject({ kind: 'session', name: 'Swim', price_toman: 320000, duration_minutes: 45, package_sessions: null });
    expect(offerProblems({ ...emptyOffer(), name: '' })).toContain('name');
    expect(offerProblems({ ...f, kind: 'package' })).toContain('sessions');
    expect(offerProblems({ ...f, ageMin: '12', ageMax: '6' })).toContain('age');
  });
});

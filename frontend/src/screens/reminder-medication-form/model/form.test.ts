import { describe, expect, it } from 'vitest';

import type { Medication } from '@/entities/care-reminder';

import {
  addDose,
  canSubmit,
  defaultDuration,
  type DoseBlock,
  durationOptions,
  emptyForm,
  everyOptions,
  fromMedication,
  isDoseComplete,
  mapServerErrors,
  MAX_DOSES,
  patchDose,
  removeDose,
  returnHref,
  slotLabel,
  slotPeriod,
  snapEvery,
  toggleMeal,
  validateForm,
} from './form';

const filled = (over: Partial<DoseBlock> = {}): DoseBlock => ({
  key: 'k',
  time: '08:00',
  amount: 1,
  unit: 'pill',
  meal: null,
  ...over,
});

const withDose = (over: Partial<ReturnType<typeof emptyForm>> = {}) => ({
  ...emptyForm('2026-09-26'),
  title: 'Folic acid',
  doses: [filled()],
  ...over,
});

describe('frequency', () => {
  it('offers 1–30 days and the fixed hour steps', () => {
    expect(everyOptions('day')).toHaveLength(30);
    expect(everyOptions('day')[29]).toBe(30);
    expect(everyOptions('hour')).toEqual([1, 2, 3, 4, 6, 8, 12]);
  });

  it('snaps `every` to a legal value when the unit changes', () => {
    expect(snapEvery('hour', 5)).toBe(4);
    expect(snapEvery('hour', 20)).toBe(12);
    expect(snapEvery('hour', 6)).toBe(6);
    expect(snapEvery('day', 3)).toBe(3);
  });
});

describe('dose blocks', () => {
  it('starts with one empty, incomplete block', () => {
    const f = emptyForm('2026-09-26');
    expect(f.doses).toHaveLength(1);
    expect(isDoseComplete(f.doses[0]!)).toBe(false);
    expect(f.frequency).toEqual({ every: 1, unit: 'day' });
  });

  it('adds up to the cap of 6', () => {
    let doses: DoseBlock[] = [filled()];
    for (let i = 0; i < 10; i++) doses = addDose(doses);
    expect(doses).toHaveLength(MAX_DOSES);
    expect(new Set(doses.map((d) => d.key)).size).toBe(MAX_DOSES);
  });

  it('removes any block but the first', () => {
    const doses = addDose(addDose([filled({ key: 'a' })]));
    const second = doses[1]!.key;
    expect(removeDose(doses, 'a')).toHaveLength(3);
    expect(removeDose(doses, second)).toHaveLength(2);
    expect(removeDose([filled({ key: 'a' })], 'a')).toHaveLength(1);
  });

  it('patches only the named block', () => {
    const doses = [filled({ key: 'a' }), filled({ key: 'b' })];
    expect(patchDose(doses, 'b', { amount: 3 }).map((d) => d.amount)).toEqual([1, 3]);
  });

  it('toggles the meal and clears it on a second tap', () => {
    expect(toggleMeal(null, 'before')).toBe('before');
    expect(toggleMeal('before', 'after')).toBe('after');
    expect(toggleMeal('after', 'after')).toBeNull();
  });

  it('the save gate needs a name and one complete dose', () => {
    expect(canSubmit(emptyForm('2026-09-26'))).toBe(false);
    expect(canSubmit(withDose({ title: ' ' }))).toBe(false);
    expect(canSubmit(withDose({ doses: [filled({ unit: null })] }))).toBe(false);
    expect(canSubmit(withDose())).toBe(true);
    expect(canSubmit(withDose({ doses: [filled({ unit: null }), filled()] }))).toBe(true);
  });
});

describe('fromMedication', () => {
  const med = (over: Partial<Medication> = {}): Medication => ({
    id: 1,
    title: 'X',
    subtitle: null,
    dose: '',
    unit: '',
    form: 'tablet',
    times: ['08:00', '20:00'],
    weekdays: [0, 1, 2, 3, 4, 5, 6],
    amount: 2,
    duration: 'ongoing',
    notify: true,
    frequency: null,
    doses: [],
    startsOn: null,
    endsOn: null,
    isActive: true,
    notes: null,
    ...over,
  });

  it('prefills an old medication from its times, amount and form', () => {
    const s = fromMedication(med({ form: 'drops' }), '2026-09-26');
    expect(s.frequency).toEqual({ every: 1, unit: 'day' });
    expect(s.doses.map((d) => [d.time, d.amount, d.unit, d.meal])).toEqual([
      ['08:00', 2, 'drop', null],
      ['20:00', 2, 'drop', null],
    ]);
    expect(s.startsOn).toBe('2026-09-26');
  });

  it('maps every old form to a dose unit', () => {
    const unit = (form: Medication['form']) => fromMedication(med({ form, times: ['08:00'] }), 'x').doses[0]!.unit;
    expect([unit('tablet'), unit('capsule'), unit('drops'), unit('syrup'), unit('injection')]).toEqual([
      'pill',
      'capsule',
      'drop',
      'ml',
      'injection',
    ]);
  });

  it('uses the new fields when present', () => {
    const s = fromMedication(
      med({
        frequency: { every: 2, unit: 'hour' },
        doses: [{ time: '07:00', amount: 3, unit: 'spoon', unitLabel: 'قاشق', meal: 'before' }],
      }),
      'x',
    );
    expect(s.frequency).toEqual({ every: 2, unit: 'hour' });
    expect(s.doses).toMatchObject([{ time: '07:00', amount: 3, unit: 'spoon', meal: 'before' }]);
  });
});

describe('validateForm', () => {
  it('accepts a valid form and emits MedicationInput', () => {
    const result = validateForm(withDose({ doses: [filled({ meal: 'after' })] }));
    expect(result.ok).toBe(true);
    if (result.ok) {
      expect(result.input).toEqual({
        title: 'Folic acid',
        startsOn: '2026-09-26',
        frequency: { every: 1, unit: 'day' },
        doses: [{ time: '08:00', amount: 1, unit: 'pill', meal: 'after' }],
        duration: 'ongoing',
        endsOn: null,
        notify: true,
        notes: null,
      });
    }
  });

  it('requires a name', () => {
    const result = validateForm(withDose({ title: '  ' }));
    expect(!result.ok && result.errors.fields.title).toBe('titleRequired');
  });

  it('flags an incomplete block on its own tiles', () => {
    const result = validateForm(withDose({ doses: [filled(), filled({ time: null, amount: null, unit: null })] }));
    expect(result.ok).toBe(false);
    if (!result.ok) {
      expect(result.errors.doses[1]).toMatchObject({ time: 'timesInvalid', amount: 'amountRange' });
      expect(result.errors.doses[0]).toBeUndefined();
    }
  });

  it('rejects duplicate times, pinned on the later block', () => {
    const result = validateForm(withDose({ doses: [filled(), filled()] }));
    expect(!result.ok && result.errors.doses[1]?.time).toBe('timesDuplicate');
  });

  it('keeps amounts within 1–99 and frequency within the rules', () => {
    const big = validateForm(withDose({ doses: [filled({ amount: 100 })] }));
    expect(!big.ok && big.errors.doses[0]?.amount).toBe('amountRange');
    const badHour = validateForm(withDose({ frequency: { every: 5, unit: 'hour' } }));
    expect(!badHour.ok && badHour.errors.fields.frequency).toBe('frequencyInvalid');
    const badDay = validateForm(withDose({ frequency: { every: 31, unit: 'day' } }));
    expect(!badDay.ok && badDay.errors.fields.frequency).toBe('frequencyInvalid');
    expect(validateForm(withDose({ frequency: { every: 8, unit: 'hour' } })).ok).toBe(true);
  });

  it('until_date needs an end on or after the start', () => {
    const missing = validateForm(withDose({ duration: 'until_date', endsOn: null }));
    expect(!missing.ok && missing.errors.fields.endsOn).toBe('endsOnRequired');
    const early = validateForm(withDose({ duration: 'until_date', endsOn: '2026-09-01' }));
    expect(!early.ok && early.errors.fields.endsOn).toBe('endsOnBeforeStart');
  });
});

describe('mapServerErrors', () => {
  it('maps snake_case, frequency and indexed dose keys onto fields', () => {
    const { fields, doses, unknown } = mapServerErrors({
      success: false,
      errors: {
        'doses.1.time': ['bad time'],
        'doses.0.amount': ['bad amount'],
        'frequency.every': ['bad every'],
        doses: ['too many'],
        ends_on: ['too early'],
        title: ['required'],
        foo: ['x'],
      },
    });
    expect(fields).toEqual({ frequency: 'bad every', doses: 'too many', endsOn: 'too early', title: 'required' });
    expect(doses).toEqual({ 1: { time: 'bad time' }, 0: { amount: 'bad amount' } });
    expect(unknown).toBe('x');
  });

  it('tolerates a body without errors', () => {
    expect(mapServerErrors({ message: 'nope' })).toEqual({ fields: {}, doses: {}, unknown: null });
  });
});

describe('slotLabel', () => {
  it('drops the leading zero and keeps 24-hour time', () => {
    expect(slotLabel('08:00')).toBe('8:00');
    expect(slotLabel('20:05')).toBe('20:05');
    expect(slotLabel('00:30')).toBe('0:30');
  });
});

describe('slotPeriod', () => {
  it('buckets the day like the reminders list', () => {
    expect(slotPeriod('08:00')).toBe('morning');
    expect(slotPeriod('14:00')).toBe('noon');
    expect(slotPeriod('17:30')).toBe('evening');
    expect(slotPeriod('20:00')).toBe('night');
    expect(slotPeriod('23:00')).toBe('night');
  });
});

describe('navigation + durations', () => {
  it('returns to the origin', () => {
    expect(returnHref('home')).toBe('/home');
    expect(returnHref(undefined)).toBe('/reminders');
    expect(returnHref('companion')).toBe('/companion');
    expect(returnHref('https://evil.example')).toBe('/reminders');
  });

  it('offers «until end of pregnancy» only in pregnancy mode', () => {
    expect(durationOptions(false, 'ongoing')).toEqual(['ongoing', 'until_date']);
    expect(durationOptions(true, 'ongoing')).toContain('pregnancy_end');
    expect(durationOptions(false, 'pregnancy_end')).toContain('pregnancy_end');
  });
});

describe('default duration', () => {
  it('starts a pregnant user on «تا پایان بارداری», everyone else on «بدون تاریخ پایان»', () => {
    expect(defaultDuration(true)).toBe('pregnancy_end');
    expect(defaultDuration(false)).toBe('ongoing');
    expect(emptyForm('2026-09-29', true).duration).toBe('pregnancy_end');
    expect(emptyForm('2026-09-29').duration).toBe('ongoing');
    expect(emptyForm('2026-09-29', true).endsOn).toBeNull();
  });

  it('a pregnancy_end medication validates without an end date', () => {
    const r = validateForm({ ...emptyForm('2026-09-29', true), title: 'Folic acid', doses: [filled()] });
    expect(r.ok).toBe(true);
    if (r.ok) expect(r.input).toMatchObject({ duration: 'pregnancy_end', endsOn: null });
  });
});

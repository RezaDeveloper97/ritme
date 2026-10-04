import { describe, expect, it } from 'vitest';

import type { IvfMed, IvfMedPreset } from '@/entities/ivf';

import { addTime, applyPreset, draftFromMed, draftProblems, draftToInput, emptyDraft } from './form';

const TODAY = '2026-09-23';

const FSH: IvfMedPreset = {
  code: 'fsh',
  title: 'هورمون تحریک (FSH)',
  role: 'stimulation',
  route: 'subcutaneous',
  unit: 'iu',
  times: ['20:00'],
  stockUnit: 'pen',
};
const PROGESTERONE: IvfMedPreset = {
  code: 'progesterone_vaginal',
  title: 'پروژسترون واژینال',
  role: 'luteal_support',
  route: 'vaginal',
  unit: 'mg',
  times: ['08:00', '20:00'],
  stockUnit: 'box',
};

describe('presets', () => {
  it('fill the class name, type, route, unit, times and stock unit', () => {
    const d = applyPreset(emptyDraft(TODAY), PROGESTERONE, 'پروژسترون واژینال', null);
    expect(d).toMatchObject({
      presetCode: 'progesterone_vaginal',
      name: 'پروژسترون واژینال',
      role: 'luteal_support',
      route: 'vaginal',
      unit: 'mg',
      times: ['08:00', '20:00'],
      stockUnit: 'box',
    });
  });

  it('keep a name she typed, replace the previous preset name', () => {
    const typed = applyPreset({ ...emptyDraft(TODAY), name: 'Gonal-f' }, FSH, 'FSH', null);
    expect(typed.name).toBe('Gonal-f');
    const swapped = applyPreset(applyPreset(emptyDraft(TODAY), FSH, 'FSH', null), PROGESTERONE, 'P', 'FSH');
    expect(swapped.name).toBe('P');
  });
});

describe('draft → input', () => {
  it('a daily medicine with stock', () => {
    const d = { ...applyPreset(emptyDraft(TODAY), FSH, 'FSH', null), dose: '150', trackStock: true, stockUnits: 3 };
    expect(draftProblems(d)).toEqual({});
    expect(draftToInput(d)).toMatchObject({
      name: 'FSH',
      dose: '150',
      unit: 'iu',
      times: ['20:00'],
      triggerAt: null,
      startsOn: TODAY,
      stockUnits: 3,
      stockUnit: 'pen',
      dosesPerUnit: 1,
    });
  });

  it('the trigger needs a date and sends trigger_at instead of times', () => {
    const d = { ...emptyDraft(TODAY), name: 'hCG', role: 'trigger' as const };
    expect(draftProblems(d)).toEqual({ trigger: 'missing' });
    const ready = { ...d, triggerDate: '2026-09-25', triggerTime: '22:30' };
    expect(draftToInput(ready)).toMatchObject({ times: null, triggerAt: '2026-09-25 22:30', stockUnits: null, stockUnit: null });
  });

  it('flags a missing name, a non-numeric dose, no times and an end before the start', () => {
    const d = { ...emptyDraft(TODAY), name: ' ', dose: '15o', times: [], endsOn: '2026-09-01' };
    expect(draftProblems(d)).toEqual({ name: 'missing', dose: 'format', times: 'missing', endsOn: 'order' });
  });

  it('drops the unit without an amount', () => {
    expect(draftToInput({ ...emptyDraft(TODAY), name: 'x' })).toMatchObject({ dose: null, unit: null });
  });
});

describe('edit draft', () => {
  const med: IvfMed = {
    id: 1,
    reminderId: 11,
    name: 'FSH',
    role: 'stimulation',
    route: 'subcutaneous',
    isTrigger: false,
    triggerAt: null,
    dose: '150',
    unit: 'iu',
    times: ['20:00'],
    startsOn: '2026-09-17',
    endsOn: null,
    isActive: true,
    notes: null,
    inventory: {
      stockUnits: 5,
      stockUnit: 'pen',
      dosesPerUnit: 1,
      countedAt: '2026-09-20 10:00:00',
      dosesLeft: 2,
      unitsLeft: 2,
      daysLeft: 2,
      runsOutOn: '2026-09-25',
      low: true,
    },
  };

  it('starts from the units left today, so an untouched save keeps the count', () => {
    const d = draftFromMed(med, TODAY);
    expect(d).toMatchObject({ trackStock: true, stockUnits: 2, startsOn: '2026-09-17', dose: '150' });
    expect(draftToInput(d).stockUnits).toBe(2);
  });

  it('splits the trigger time', () => {
    const d = draftFromMed({ ...med, role: 'trigger', isTrigger: true, triggerAt: '2026-09-25 22:30:00', times: [], inventory: null }, TODAY);
    expect(d).toMatchObject({ triggerDate: '2026-09-25', triggerTime: '22:30', trackStock: false });
  });
});

describe('addTime', () => {
  it('adds 12 h after the last, sorted, at most four', () => {
    expect(addTime(['08:00'])).toEqual(['08:00', '20:00']);
    expect(addTime(['08:00', '20:00'])).toEqual(['08:00', '09:00', '20:00']);
    expect(addTime(['01:00', '02:00', '03:00', '04:00'])).toHaveLength(4);
  });
});

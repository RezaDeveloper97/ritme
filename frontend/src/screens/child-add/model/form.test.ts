import { describe, expect, it } from 'vitest';

import { EMPTY_FORM, formFromChild, toChildInput, validateChildForm } from './form';

const TODAY = '2026-10-04';

describe('validateChildForm', () => {
  it('requires a name and a birth date', () => {
    expect(validateChildForm(EMPTY_FORM, TODAY)).toEqual([
      { field: 'name', key: 'nameRequired' },
      { field: 'birthDate', key: 'dateRequired' },
    ]);
  });

  it('rejects a future or too old birth date and out-of-range birth values', () => {
    const base = { ...EMPTY_FORM, name: 'آوا' };
    expect(validateChildForm({ ...base, birthDate: '2026-10-05' }, TODAY)).toEqual([{ field: 'birthDate', key: 'future' }]);
    expect(validateChildForm({ ...base, birthDate: '2000-01-01' }, TODAY)).toEqual([{ field: 'birthDate', key: 'tooOld' }]);
    expect(
      validateChildForm({ ...base, birthDate: '2026-06-12', weightKg: '12', lengthCm: '50', headCm: '10' }, TODAY),
    ).toEqual([
      { field: 'weightKg', key: 'range' },
      { field: 'headCm', key: 'range' },
    ]);
  });

  it('accepts a complete form', () => {
    const s = { ...EMPTY_FORM, name: 'آوا', birthDate: '2026-06-12', weightKg: '3.2', lengthCm: '50' };
    expect(validateChildForm(s, TODAY)).toEqual([]);
    expect(toChildInput(s)).toEqual({
      name: 'آوا',
      birthDate: '2026-06-12',
      sex: null,
      deliveryType: null,
      birthWeightKg: 3.2,
      birthLengthCm: 50,
      birthHeadCm: null,
    });
  });
});

describe('formFromChild', () => {
  it('round-trips the stored values', () => {
    const s = formFromChild({
      name: 'Ava',
      birthDate: '2026-06-12',
      sex: 'girl',
      deliveryType: 'vaginal',
      birth: { weightKg: 3.2, lengthCm: 50, headCm: null },
    });
    expect(s).toMatchObject({ weightKg: '3.2', lengthCm: '50', headCm: '' });
  });
});

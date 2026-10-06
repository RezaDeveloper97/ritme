import { describe, expect, it } from 'vitest';

import type { ParamField } from '../api/alert-rules';
import { paramsBody } from './params';

const schema: ParamField[] = [
  { key: 'from_week', kind: 'integer', nullable: false, min: 1, max: 42 },
  { key: 'from_weekday', kind: 'integer', nullable: true, min: 0, max: 6 },
  { key: 'symptoms', kind: 'enum_list', nullable: false, values: ['a', 'b'] },
];

describe('alert rule params body', () => {
  it('types integers and keeps schema keys only', () => {
    expect(paramsBody(schema, { from_week: '3', from_weekday: '5', symptoms: ['a'], extra: 1 })).toEqual({
      from_week: 3,
      from_weekday: 5,
      symptoms: ['a'],
    });
  });

  it('sends an empty optional integer as null (engine default), a required one as typed', () => {
    expect(paramsBody(schema, { from_week: '', from_weekday: '' })).toEqual({
      from_week: '',
      from_weekday: null,
      symptoms: undefined,
    });
    expect(paramsBody(schema, { from_week: 0 }).from_weekday).toBeNull();
  });

  it('trims a text param and sends an empty one as null (contact_phone)', () => {
    const phone: ParamField[] = [{ key: 'contact_phone', kind: 'text', nullable: true, max_length: 20 }];
    expect(paramsBody(phone, { contact_phone: ' 021 6612 3456 ' })).toEqual({ contact_phone: '021 6612 3456' });
    expect(paramsBody(phone, { contact_phone: '  ' })).toEqual({ contact_phone: null });
    expect(paramsBody(phone, {})).toEqual({ contact_phone: null });
  });
});

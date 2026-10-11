import { describe, expect, it } from 'vitest';

import { asciiDigits, emptyBookingForm, validateBookingForm } from './form';

describe('city-service booking form', () => {
  it('normalises Persian digits and separators', () => {
    expect(asciiDigits('۰۹۱۲ ۱۲۳-۴۵۶۷')).toBe('09121234567');
    expect(asciiDigits('٠٩١٢(١٢٣)٤٥٦٧')).toBe('09121234567');
  });

  it('needs a name and a phone', () => {
    expect(validateBookingForm(emptyBookingForm('2026-10-12'))).toEqual({ name: 'required', phone: 'required' });
  });

  it('rejects a malformed phone and accepts a valid one', () => {
    const base = { ...emptyBookingForm('2026-10-12'), name: 'Sara' };
    expect(validateBookingForm({ ...base, phone: 'abc' })).toEqual({ phone: 'phoneInvalid' });
    expect(validateBookingForm({ ...base, phone: '۰۹۱۲ ۱۲۳ ۴۵۶۷' })).toEqual({});
    expect(validateBookingForm({ ...base, phone: '+98 912 123 4567' })).toEqual({});
  });
});

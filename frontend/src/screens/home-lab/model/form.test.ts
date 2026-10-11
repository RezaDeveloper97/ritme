import { describe, expect, it } from 'vitest';

import { asciiDigits, emptyLabForm, validateLabForm } from './form';

const filled = () => ({
  ...emptyLabForm('2026-10-14'),
  fullName: 'Sara',
  phone: '۰۹۱۲ ۰۰۰ ۰۰۰۰',
  city: 'Tehran',
  address: 'Valiasr',
  consentShare: true,
});

describe('home lab form', () => {
  it('turns Persian digits into ASCII', () => {
    expect(asciiDigits(' ۰۹۱۲-۰۰۰ ۰۰۰۰ ')).toBe('09120000000');
    expect(asciiDigits('٠٩١٢٠٠٠٠٠٠٠')).toBe('09120000000');
  });

  it('accepts a complete form', () => {
    expect(validateLabForm(filled())).toEqual({});
  });

  it('asks for every required field and the share consent', () => {
    const errors = validateLabForm(emptyLabForm('2026-10-14'));
    expect(errors).toEqual({ fullName: 'required', city: 'required', address: 'required', phone: 'required', consentShare: 'consentRequired' });
  });

  it('refuses a phone that is not one', () => {
    expect(validateLabForm({ ...filled(), phone: 'call me' }).phone).toBe('phoneInvalid');
  });

  it('does not require the import consent', () => {
    expect(validateLabForm({ ...filled(), consentImport: false })).toEqual({});
  });
});

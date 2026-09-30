import { describe, expect, it } from 'vitest';

import { firstLetter, groupMobile, localizeDigits, maskMobile } from './me-format';

describe('me-format', () => {
  it('masks the middle of the mobile number, in the locale digits', () => {
    expect(maskMobile('09121234545', 'en')).toBe('0912 ••• ••45');
    expect(maskMobile('09121234545', 'fa')).toBe('۰۹۱۲ ••• ••۴۵');
  });

  it('groups the number 4-3-4 on the account screen', () => {
    expect(groupMobile('09123456789', 'en')).toBe('0912 345 6789');
    expect(groupMobile('0912', 'fa')).toBe('۰۹۱۲');
  });

  it('localizes digits only in fa', () => {
    expect(localizeDigits('1.0.0', 'fa')).toBe('۱.۰.۰');
    expect(localizeDigits('1.0.0', 'en')).toBe('1.0.0');
  });

  it('takes the first letter of a name', () => {
    expect(firstLetter('  مریم')).toBe('م');
    expect(firstLetter('Sara')).toBe('S');
    expect(firstLetter('')).toBe('');
  });
});

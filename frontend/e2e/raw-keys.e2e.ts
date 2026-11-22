import { expect, test } from '@playwright/test';

import { findRawKeys } from './support/raw-keys';

// Unit-level checks of the detector itself (no browser): a detector that never fires would make every page check green.
test.describe('raw i18n key detector', () => {
  test('flags a leaked namespace.key', () => {
    expect(findRawKeys('سلام home.ring.noPredictionValue روز')).toEqual(['home.ring.noPredictionValue']);
    expect(findRawKeys('auth.errors.invalidCode')).toEqual(['auth.errors.invalidCode']);
    expect(findRawKeys('log.tile:bleeding common.save')).toEqual(['common.save']);
  });

  test('ignores domains, versions, numbers and Persian copy', () => {
    expect(findRawKeys('ritme.app و support@ritme.app و ritmeapp.ir')).toEqual([]);
    expect(findRawKeys('نسخه 1.2.3 و ۳.۵ روز و 36.6')).toEqual([]);
    expect(findRawKeys('پریود روز ۱ — معمولاً ۵ روز.')).toEqual([]);
    expect(findRawKeys('')).toEqual([]);
  });
});

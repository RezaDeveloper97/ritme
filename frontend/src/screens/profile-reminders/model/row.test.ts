import { describe, expect, it } from 'vitest';

import { appointmentParts, byReminderId, clockLabel, isCancelled } from './row';

describe('legacy reminders sheet rows', () => {

  it('shows the hour without a leading zero (B-2)', () => {
    expect(clockLabel('08:00')).toBe('8:00');
    expect(clockLabel('08:00:00')).toBe('8:00');
    expect(clockLabel('20:30')).toBe('20:30');
    expect(clockLabel('00:15')).toBe('0:15');
    expect(clockLabel('later')).toBe('later');
  });

  it('splits the appointment subtitle into its set parts', () => {
    expect(appointmentParts({ withWhom: 'دکتر احمدی', specialty: 'زنان و زایمان' })).toEqual([
      'دکتر احمدی',
      'زنان و زایمان',
    ]);
    expect(appointmentParts({ withWhom: ' ', specialty: null })).toEqual([]);
  });

  it('flags a cancelled appointment (B-1)', () => {
    expect(isCancelled({ status: 'cancelled' })).toBe(true);
    expect(isCancelled({ status: 'scheduled' })).toBe(false);
    expect(isCancelled(undefined)).toBe(false);
  });

  it('keys care rows by the legacy string id', () => {
    const map = byReminderId([{ id: 14 }, { id: 19 }]);
    expect(map.get('14')).toEqual({ id: 14 });
    expect(byReminderId(undefined).size).toBe(0);
  });
});

import { describe, expect, it } from 'vitest';

import { emergencyCardSchema, lockEmergencyCardSchema } from './emergency';

describe('lockEmergencyCardSchema (CB-PRIV-01, GET /health-record/emergency-card/lock)', () => {
  it('is null while the owner keeps the card off the lock screen', () => {
    expect(lockEmergencyCardSchema.parse({ enabled: false })).toBeNull();
  });

  it('maps the minimal card and never carries insurance', () => {
    const c = lockEmergencyCardSchema.parse({
      enabled: true,
      card: {
        name: 'سارا',
        blood_type: 'O+',
        allergies: [],
        conditions: { chronic_illnesses: ['thyroid'] },
        medications: [],
        pregnancy: null,
        emergency_contact: { name: 'علی', relation: 'همسر', phone: '09120000000' },
        insurance: { label: 'x', masked: '•••• 4821' },
      },
    });
    expect(c?.name).toBe('سارا');
    expect(c?.insurance).toBeNull();
    expect(c?.contact?.relation).toBe('همسر');
  });
});

const owner = {
  card: {
    name: 'سارا رضایی',
    blood_type: 'O+',
    allergies: ['پنی‌سیلین'],
    conditions: { chronic_illnesses: ['thyroid'], gyn_conditions: ['pcos'] },
    medications: [{ title: 'لووتیروکسین', dose: '50mcg' }],
    profile_medications: ['x'],
    pregnancy: { week: 8 },
    emergency_contact: { name: 'علی', relation: 'همسر', phone: '09120000000' },
    insurance: { label: 'تکمیلی', last4: '4821', masked: '•••• 4821' },
  },
  settings: { show_on_lock_screen: true, show_pregnancy: true, allergies_on_card: true },
  public_link: { enabled: false, enabled_at: null, views: 0, last_viewed_at: null },
};

describe('emergencyCardSchema (CB-REC-03 owner view, read by CB-PRIV-01)', () => {
  it('maps the owner card and the lock-screen flag', () => {
    const c = emergencyCardSchema.parse(owner);
    expect(c.showOnLockScreen).toBe(true);
    expect(c.bloodType).toBe('O+');
    expect(c.conditions).toEqual(['thyroid']);
    expect(c.pregnancyWeek).toBe(8);
    expect(c.insurance).toEqual({ label: 'تکمیلی', masked: '•••• 4821' });
    expect(c.contact?.phone).toBe('09120000000');
  });

  it('keeps the link hidden and tolerates an empty card', () => {
    const c = emergencyCardSchema.parse({
      card: {
        name: null,
        blood_type: null,
        allergies: null,
        conditions: { chronic_illnesses: [] },
        medications: [],
        pregnancy: null,
        emergency_contact: null,
        insurance: null,
      },
      settings: { show_on_lock_screen: false },
    });
    expect(c.showOnLockScreen).toBe(false);
    expect(c.allergies).toBeNull();
    expect(c.pregnancyWeek).toBeNull();
    expect(c.insurance).toBeNull();
  });
});

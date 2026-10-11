import { describe, expect, it } from 'vitest';

import { centreSchema, CENTRE_TYPES, insurersApi, centresApi, insurerSchema } from './insurance';

describe('insurance admin api (admin-api.md §27)', () => {
  it('parses an insurer row', () => {
    const row = { id: 1, code: 'sina', name: 'Sina', website: null, phone: '021', is_active: true, sort_order: 0, updated_at: 'x' };
    expect(insurerSchema.parse(row).code).toBe('sina');
  });

  it('parses a centre row with null insurer and coordinates', () => {
    const row = {
      id: 2, insurer_id: null, insurer_name: null, name: 'Lab', type: 'lab', services: null, city: null, address: null,
      phone: null, lat: '35.7000000', lng: '51.4000000', is_direct: true, is_active: true, sort_order: 3, updated_at: 'x',
    };
    expect(centreSchema.parse(row).is_direct).toBe(true);
    expect(CENTRE_TYPES).toContain('lab');
  });

  it('exposes list/save/remove hooks for both resources', () => {
    for (const r of [insurersApi, centresApi]) {
      expect(typeof r.useList).toBe('function');
      expect(typeof r.useSave).toBe('function');
      expect(typeof r.useRemove).toBe('function');
    }
  });
});

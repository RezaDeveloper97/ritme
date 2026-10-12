import { describe, expect, it } from 'vitest';

import { centresSchema, claimsSchema, policyHistorySchema } from './schema';

describe('centresSchema', () => {
  it('parses string coordinates and nulls', () => {
    const rows = centresSchema.parse({
      centres: [
        { id: 1, name: 'A', type: 'lab', services: null, city: 'Tehran', address: null, phone: '021', lat: '35.7001', lng: '51.4', is_direct: true, insurer_name: null },
        { id: 2, name: 'B', type: 'gyn', lat: null, lng: 'x', is_direct: false },
      ],
    });
    expect(rows[0]).toMatchObject({ lat: 35.7001, lng: 51.4, isDirect: true, services: null });
    expect(rows[1]).toMatchObject({ lat: null, lng: null, isDirect: false });
  });
});

describe('policyHistorySchema', () => {
  it('maps the history rows and the current id', () => {
    const h = policyHistorySchema.parse({
      current_id: 2,
      policies: [
        { id: 2, insurer_name: 'Alpha', plan_name: null, valid_from: '2026-03-21', valid_to: '2027-03-20', status: 'current', read_only: false, can_renew: true, claims_count: 0 },
        { id: 1, insurer_name: 'Alpha', plan_name: 'Silver', valid_from: '2025-03-21', valid_to: '2026-03-20', status: 'past', read_only: true, can_renew: false, claims_count: 3 },
        { id: 9, insurer_name: 'Beta', status: 'weird' },
      ],
    });
    expect(h.currentId).toBe(2);
    expect(h.policies[1]).toMatchObject({ id: 1, planName: 'Silver', status: 'past', readOnly: true, claimsCount: 3 });
    expect(h.policies[2]).toMatchObject({ status: 'current', readOnly: false, validFrom: null, claimsCount: 0 });
  });

  it('carries the policy of a claims list (null for older responses)', () => {
    const base = { claims: [], totals: { requested: 0, paid: 0, count: 0 } };
    expect(claimsSchema.parse({ ...base, policy: { id: 1, status: 'past', read_only: true } }).policy).toEqual({ id: 1, readOnly: true });
    expect(claimsSchema.parse(base).policy).toBeNull();
  });
});

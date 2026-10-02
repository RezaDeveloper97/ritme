import { describe, expect, it } from 'vitest';

import type { OwnerCompanion } from '../model/types';
import { auditSchema, ownerCompanionSchema, createdCompanionSchema } from '../api/schema';
import {
  companionName,
  emptyGrants,
  familySpouse,
  grantsByLevel,
  hoursUntil,
  sameGrants,
  sharedSections,
} from './grants';

const row = {
  id: 7,
  type: 'spouse',
  status: 'active',
  display_name: null,
  name: 'Ali',
  invited_at: '2026-10-01T10:00:00Z',
  accepted_at: '2026-10-01T11:00:00Z',
  grants: { cycle: 'view', symptoms: 'none', meds: 'edit', appointments: 'edit', pregnancy: 'view' },
  invite: null,
  family: { id: 3, shared_child_ids: [] },
};

describe('companion schema', () => {
  it('maps an owner link', () => {
    const c = ownerCompanionSchema.parse(row);
    expect(c.name).toBe('Ali');
    expect(c.grants.meds).toBe('edit');
    expect(c.family).toEqual({ id: 3, sharedChildIds: [] });
  });

  it('reads unknown levels as none and fills missing sections', () => {
    const c = ownerCompanionSchema.parse({ ...row, grants: { cycle: 'admin', future: 'edit' } });
    expect(c.grants).toEqual(emptyGrants());
  });

  it('maps a created invite', () => {
    const created = createdCompanionSchema.parse({
      companion: { ...row, status: 'invited', invite: { expires_at: '2026-10-02T10:00:00Z', phone: '0912****567' } },
      invite: { code: 'RT7K2M', expires_at: '2026-10-02T10:00:00Z', phone: '0912****567', sms_sent: false },
    });
    expect(created.invite).toEqual({ code: 'RT7K2M', expiresAt: '2026-10-02T10:00:00Z', phone: '0912****567', smsSent: false });
    expect(created.companion.invite?.phone).toBe('0912****567');
  });

  it('skips audit rows it cannot read', () => {
    const rows = auditSchema.parse([
      { id: 1, action: 'read', section: 'meds', companion_id: 7, actor: { id: 2, name: null, is_me: false }, at: 'x' },
      { id: 2, action: 'teleported', section: null, companion_id: 7, actor: null, at: 'x' },
    ]);
    expect(rows.map((r) => r.id)).toEqual([1]);
  });
});

describe('grant helpers', () => {
  const grants = ownerCompanionSchema.parse(row).grants;

  it('groups by level in section order', () => {
    expect(grantsByLevel(grants)).toEqual({ edit: ['meds', 'appointments'], view: ['cycle', 'pregnancy'] });
    expect(sharedSections(grants)[0]).toEqual({ section: 'meds', level: 'edit' });
  });

  it('compares grants', () => {
    expect(sameGrants(grants, { ...grants })).toBe(true);
    expect(sameGrants(grants, emptyGrants())).toBe(false);
  });

  it('names a companion', () => {
    expect(companionName({ displayName: ' Sara ', name: 'Ali' })).toBe('Sara');
    expect(companionName({ displayName: '', name: null })).toBeNull();
  });

  it('counts hours left', () => {
    const now = new Date('2026-10-01T10:00:00Z');
    expect(hoursUntil('2026-10-02T10:00:00Z', now)).toBe(24);
    expect(hoursUntil('2026-10-01T09:00:00Z', now)).toBe(0);
    expect(hoursUntil('nope', now)).toBe(0);
  });

  it('picks the active spouse for the family', () => {
    const a = ownerCompanionSchema.parse(row);
    const pending: OwnerCompanion = { ...a, id: 8, status: 'invited' };
    const partner: OwnerCompanion = { ...a, id: 9, type: 'partner' };
    expect(familySpouse([pending, a, partner])?.id).toBe(7);
    expect(familySpouse([partner])).toBeNull();
  });
});

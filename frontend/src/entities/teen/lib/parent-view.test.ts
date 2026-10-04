import { describe, expect, it } from 'vitest';

import { teenLinkedSchema, teenParentCreatedSchema, teenGrantsBody } from '../api/schema';
import { NO_TEEN_GRANTS, type TeenParentLink, type TeenParentView } from '../model/types';
import { isEmptyParentView, maskParentView, primaryParentLink, sameTeenGrants, withShare } from './parent-view';

const FULL: TeenParentView = { nextPeriodWeek: 'next_week', kitReady: true, note: '  Cramps today ' };

describe('maskParentView', () => {
  it('shows nothing while every switch is off (the default)', () => {
    const view = maskParentView(FULL, NO_TEEN_GRANTS);
    expect(view).toEqual({ nextPeriodWeek: null, kitReady: null, note: null });
    expect(isEmptyParentView(view)).toBe(true);
  });

  it('fills only the granted sections and trims the note', () => {
    const grants = withShare(withShare(NO_TEEN_GRANTS, 'teenPeriodWeek', true), 'teenNotes', true);
    expect(maskParentView(FULL, grants)).toEqual({ nextPeriodWeek: 'next_week', kitReady: null, note: 'Cramps today' });
  });

  it('treats a blank note as no note', () => {
    const grants = withShare(NO_TEEN_GRANTS, 'teenNotes', true);
    const view = maskParentView({ ...FULL, note: '   ' }, grants);
    expect(view.note).toBeNull();
    expect(isEmptyParentView(view)).toBe(true);
  });
});

describe('grants helpers', () => {
  it('toggles one section and compares', () => {
    const on = withShare(NO_TEEN_GRANTS, 'teenKit', true);
    expect(on.teenKit).toBe('view');
    expect(sameTeenGrants(on, NO_TEEN_GRANTS)).toBe(false);
    expect(sameTeenGrants(withShare(on, 'teenKit', false), NO_TEEN_GRANTS)).toBe(true);
  });

  it('sends view-only levels in API keys', () => {
    expect(teenGrantsBody(withShare(NO_TEEN_GRANTS, 'teenNotes', true))).toEqual({
      teen_period_week: 'none',
      teen_kit: 'none',
      teen_notes: 'view',
    });
  });

  it('manages an active link before a pending invite', () => {
    const link = (id: number, status: TeenParentLink['status']): TeenParentLink => ({
      id,
      status,
      displayName: null,
      grants: NO_TEEN_GRANTS,
    });
    expect(primaryParentLink([])).toBeNull();
    expect(primaryParentLink([link(1, 'invited'), link(2, 'active')])?.id).toBe(2);
    expect(primaryParentLink([link(3, 'invited')])?.id).toBe(3);
  });
});

describe('parent API payloads', () => {
  it('parses GET /teen/linked cards (contract teen/parent_link_card)', () => {
    const cards = teenLinkedSchema.parse([
      {
        link_id: 1,
        teen: { id: 1004, name: 'Contract regular' },
        accepted_at: '2026-09-23T10:00:00+03:30',
        grants: { teen_period_week: 'view', teen_kit: 'view', teen_notes: 'none' },
        next_period_week: 'later',
        kit_ready: false,
        note: null,
        read_only: true,
      },
      { link_id: 'broken' },
    ]);
    expect(cards).toEqual([
      {
        linkId: 1,
        teenName: 'Contract regular',
        grants: { teenPeriodWeek: 'view', teenKit: 'view', teenNotes: 'none' },
        nextPeriodWeek: 'later',
        kitReady: false,
        note: null,
      },
    ]);
  });

  it('reads an unknown grant level as not shared', () => {
    const [card] = teenLinkedSchema.parse([
      {
        link_id: 2,
        teen: { name: null },
        grants: { teen_period_week: 'edit', teen_kit: 'view', teen_notes: 'view' },
        next_period_week: 'someday',
        kit_ready: true,
        note: 'hi',
      },
    ]);
    expect(card?.grants.teenPeriodWeek).toBe('none');
    expect(card?.nextPeriodWeek).toBeNull();
  });

  it('parses the created parent invite', () => {
    const created = teenParentCreatedSchema.parse({
      companion: {
        id: 1,
        type: 'parent',
        status: 'invited',
        display_name: 'Mom',
        grants: { teen_period_week: 'view', teen_kit: 'view', teen_notes: 'none' },
      },
      invite: { code: 'ABC234', expires_at: '2026-09-24T10:00:00+03:30', phone: '0990****001', sms_sent: true },
    });
    expect(created.link).toMatchObject({ id: 1, status: 'invited', displayName: 'Mom' });
    expect(created.invite).toEqual({
      code: 'ABC234',
      expiresAt: '2026-09-24T10:00:00+03:30',
      phone: '0990****001',
      smsSent: true,
    });
  });
});

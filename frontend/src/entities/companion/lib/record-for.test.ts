import { describe, expect, it } from 'vitest';

import { ApiError } from '@/shared/api';

import type { ViewerLink } from '../model/companion-side';
import type { CompanionGrants, CompanionSection } from '../model/types';
import {
  canRecordFor,
  initialRecordTarget,
  isCompanionForbidden,
  parseForUserId,
  recordTargets,
  showRecordForPicker,
} from './record-for';
import { emptyGrants } from './grants';

function link(id: number, ownerId: number, grants: Partial<CompanionGrants>, canRecord: CompanionSection[] = []): ViewerLink {
  return {
    id,
    type: 'partner',
    owner: { id: ownerId, name: `owner-${ownerId}` },
    acceptedAt: null,
    grants: { ...emptyGrants(), ...grants },
    canRecordFor: canRecord,
    family: null,
  };
}

describe('record-for picker visibility', () => {
  it('is hidden without links', () => {
    expect(showRecordForPicker(undefined, 'meds')).toBe(false);
    expect(showRecordForPicker(null, 'appointments')).toBe(false);
    expect(showRecordForPicker([], 'meds')).toBe(false);
  });

  it('is hidden when the owner only lets the viewer see the section', () => {
    const links = [link(1, 10, { meds: 'view', appointments: 'view', cycle: 'edit' })];
    expect(showRecordForPicker(links, 'meds')).toBe(false);
    expect(showRecordForPicker(links, 'appointments')).toBe(false);
  });

  it('is per section: edit on meds shows it on the medication form only', () => {
    const links = [link(1, 10, { meds: 'edit', appointments: 'view' }, ['meds'])];
    expect(showRecordForPicker(links, 'meds')).toBe(true);
    expect(showRecordForPicker(links, 'appointments')).toBe(false);
  });

  it('trusts either can_record_for or an edit grant', () => {
    expect(canRecordFor(link(1, 10, {}, ['appointments']), 'appointments')).toBe(true);
    expect(canRecordFor(link(1, 10, { appointments: 'edit' }), 'appointments')).toBe(true);
    expect(canRecordFor(link(1, 10, { appointments: 'view' }), 'appointments')).toBe(false);
  });

  it('lists each owner once, in link order, skipping view-only links', () => {
    const links = [
      link(1, 10, { meds: 'edit' }),
      link(2, 20, { meds: 'view' }),
      link(3, 30, { meds: 'edit' }),
      link(4, 10, { meds: 'edit' }),
    ];
    expect(recordTargets(links, 'meds')).toEqual([
      { ownerId: 10, name: 'owner-10', linkId: 1 },
      { ownerId: 30, name: 'owner-30', linkId: 3 },
    ]);
  });
});

describe('record-for target', () => {
  const targets = recordTargets([link(1, 10, { meds: 'edit' })], 'meds');

  it('parses ?for= strictly', () => {
    expect(parseForUserId('10')).toBe(10);
    expect(parseForUserId('0')).toBeNull();
    expect(parseForUserId('-3')).toBeNull();
    expect(parseForUserId('1e3')).toBeNull();
    expect(parseForUserId('')).toBeNull();
    expect(parseForUserId(null)).toBeNull();
    expect(parseForUserId('9999999999999999')).toBeNull();
  });

  it('preselects the owner only when the viewer may still record for her', () => {
    expect(initialRecordTarget(targets, 10)).toBe(10);
    expect(initialRecordTarget(targets, 20)).toBeNull();
    expect(initialRecordTarget(targets, null)).toBeNull();
    expect(initialRecordTarget([], 10)).toBeNull();
  });
});

describe('isCompanionForbidden', () => {
  const failed = (status: number, data: unknown) =>
    new ApiError(`Request failed with status code ${status}`, { response: { status, data, headers: {} }, sentToken: null });

  it('matches the companion 403 only', () => {
    expect(isCompanionForbidden(failed(403, { error_code: 'companion_forbidden' }))).toBe(true);
    expect(isCompanionForbidden(failed(403, { message: 'x' }))).toBe(true);
    expect(isCompanionForbidden(failed(403, { error_code: 'plus_required' }))).toBe(false);
    expect(isCompanionForbidden(failed(404, {}))).toBe(false);
    expect(isCompanionForbidden(new Error('x'))).toBe(false);
  });
});

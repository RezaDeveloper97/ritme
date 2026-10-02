import { getApiErrorCode, getApiErrorStatus } from '@/shared/api';

import type { ViewerLink } from '../model/companion-side';

/*
 * «ثبت برای چه کسی؟» (B-N4-06, Hamdam_RecordFor on B-N4-02): a companion with
 * `edit` on an owner's meds / appointments may create and edit them in her list
 * by sending `for_user_id`. The picker lists only those owners — never someone
 * with view-only access — and is not shown at all when there is nobody.
 */

/** The sections a companion can record into. */
export type RecordSection = 'meds' | 'appointments';

/** One owner the viewer may record for. */
export interface RecordTarget {
  ownerId: number;
  name: string | null;
  linkId: number;
}

/** True when this link lets the viewer write the owner's section. */
export function canRecordFor(link: ViewerLink, section: RecordSection): boolean {
  return link.canRecordFor.includes(section) || link.grants[section] === 'edit';
}

/** The owners the viewer may record `section` for, one per owner, in link order. */
export function recordTargets(links: readonly ViewerLink[] | null | undefined, section: RecordSection): RecordTarget[] {
  const out: RecordTarget[] = [];
  for (const link of links ?? []) {
    if (!canRecordFor(link, section) || out.some((t) => t.ownerId === link.owner.id)) continue;
    out.push({ ownerId: link.owner.id, name: link.owner.name, linkId: link.id });
  }
  return out;
}

/** The picker shows only when at least one owner granted edit on the section. */
export function showRecordForPicker(links: readonly ViewerLink[] | null | undefined, section: RecordSection): boolean {
  return recordTargets(links, section).length > 0;
}

/** `?for=<ownerId>` → a positive integer id, else null. */
export function parseForUserId(raw: string | null | undefined): number | null {
  if (!raw || !/^\d{1,15}$/.test(raw)) return null;
  const id = Number(raw);
  return Number.isSafeInteger(id) && id > 0 ? id : null;
}

/**
 * The initial target of a new record: the owner named by `?for=` when the
 * viewer may still record for her, else the viewer (null).
 */
export function initialRecordTarget(targets: readonly RecordTarget[], forUserId: number | null): number | null {
  return forUserId !== null && targets.some((t) => t.ownerId === forUserId) ? forUserId : null;
}

/** The 403 a delegated read/write gets once the owner revoked the grant (`companion_forbidden`). */
export function isCompanionForbidden(error: unknown): boolean {
  if (getApiErrorStatus(error) !== 403) return false;
  const code = getApiErrorCode(error);
  return code === undefined || code === 'companion_forbidden';
}

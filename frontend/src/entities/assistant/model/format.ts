import type { IconName } from '@/shared/ui';

import type { ProfileItem, TriageLevel } from '../api/schema';

/** Department `icon` codes of the seed (backend-go migration 00052) → the app's icon set. */
const DEPT_ICONS: Record<string, IconName> = {
  stethoscope: 'stetho',
  flower: 'sprout',
  heart: 'heart',
  sparkle: 'sparkle',
  leaf: 'apple',
  moon: 'moon',
};
export const deptIcon = (icon: string | null): IconName => (icon && DEPT_ICONS[icon]) || 'sparkle';

export const TRIAGE_TONE = { self_care: 'success', non_urgent: 'data', soon: 'warm', emergency: 'danger' } as const satisfies Record<TriageLevel, string>;

/** The one-line value of a shared profile item, when it has a scalar one (age, life mode); null → show the label alone. */
export function scalarOf(item: ProfileItem): string | number | null {
  return typeof item.data === 'string' || typeof item.data === 'number' ? item.data : null;
}

/** 115 is a fixed constant of the product; the hub payload's `emergency.phone` wins when present. */
export const EMERGENCY_PHONE = '115';

export const DRAFT_KEY = (id: number) => `assistant:draft:${id}`;

/** Keeps the first message typed on the hub / lab link for the chat screen — never in a URL (privacy §11). */
export function stashDraft(id: number, text: string) {
  try {
    sessionStorage.setItem(DRAFT_KEY(id), text);
  } catch {
    /* private mode: the user types it again */
  }
}

export function takeDraft(id: number): string {
  try {
    const v = sessionStorage.getItem(DRAFT_KEY(id));
    sessionStorage.removeItem(DRAFT_KEY(id));
    return v ?? '';
  } catch {
    return '';
  }
}

import type { IconName } from '@/shared/ui';

import type { CheckupCategory, CheckupPerformer } from './types';

/*
 * `checkup_types.icon` is an admin-entered name "from the frontend set". The
 * admin may type it kebab-case (`shield-check`) or use the artboard's names, so
 * both spellings and a few domain aliases resolve; anything unknown falls back
 * by performer / category, never to a blank tile.
 */
const ICONS: Record<string, IconName> = {
  ribbon: 'ribbon',
  breast: 'ribbon',
  shield: 'shield',
  shieldcheck: 'shield',
  flask: 'flask',
  lab: 'flask',
  blood: 'flask',
  tooth: 'tooth',
  dentist: 'tooth',
  stetho: 'stetho',
  stethoscope: 'stetho',
  heart: 'heart',
  calendar: 'calendar',
  drop: 'drop',
  pill: 'pill',
  note: 'note',
  file: 'note',
};

const BY_PERFORMER: Record<CheckupPerformer, IconName> = {
  self: 'ribbon',
  doctor: 'stetho',
  lab: 'flask',
  dentist: 'tooth',
};

/** Resolve a catalog icon name to one the `Icon` component draws. */
export function checkupIcon(
  name: string | null | undefined,
  fallback?: { performedBy?: CheckupPerformer; category?: CheckupCategory },
): IconName {
  const normalized = (name ?? '').toLowerCase().replace(/[-_\s]/g, '');
  const known = ICONS[normalized];
  if (known) return known;
  if (fallback?.performedBy) return BY_PERFORMER[fallback.performedBy];
  if (fallback?.category === 'monthly') return 'ribbon';
  return 'stetho';
}

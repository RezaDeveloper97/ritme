import type { IconName } from '@/shared/ui';

/** Catalog icon names (options.icons) → the admin Icon set; aliases as the app resolves them. */
const ICONS: Record<string, IconName> = {
  ribbon: 'ribbon',
  breast: 'ribbon',
  shield: 'shield',
  shieldCheck: 'shieldCheck',
  flask: 'flask',
  blood: 'flask',
  tooth: 'tooth',
  stetho: 'stetho',
  heart: 'heart',
  calendar: 'calendar',
  drop: 'drop',
  pill: 'pill',
  note: 'note',
};

const BY_PERFORMER: Record<string, IconName> = { self: 'ribbon', doctor: 'stetho', lab: 'flask', dentist: 'tooth' };

export function checkupIcon(name: string | null | undefined, performedBy: string): IconName {
  return (name && ICONS[name]) || BY_PERFORMER[performedBy] || 'calendar';
}

/** Sections of the feeding screen (`/children/[id]/feeding`, Log_Feed + baby sleep + diapers). */
export type BabyLogSection = 'feeding' | 'sleep' | 'diapers';

/** Link into the feeding screen; `sleep` / `diapers` scroll to their card (`?section=`). */
export function feedingHref(childId: number | null, section: BabyLogSection = 'feeding'): string {
  const base = childId ? `/children/${childId}/feeding` : '/children/feeding';
  return section === 'feeding' ? base : `${base}?section=${section}`;
}

/** `?section=` back into a section (anything else = the feed timer). */
export function sectionOf(value: string | null | undefined): BabyLogSection {
  return value === 'sleep' || value === 'diapers' ? value : 'feeding';
}

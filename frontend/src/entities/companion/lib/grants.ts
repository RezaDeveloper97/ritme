import {
  COMPANION_SECTIONS,
  type AccessLevel,
  type CompanionGrants,
  type CompanionSection,
  type OwnerCompanion,
} from '../model/types';

/** Every section on `none` — the most private default (CLAUDE.md §11) for a new companion. */
export function emptyGrants(): CompanionGrants {
  return Object.fromEntries(COMPANION_SECTIONS.map((s) => [s, 'none'])) as CompanionGrants;
}

/** The sections per shared level, in section order (summary rows, list chips). */
export function grantsByLevel(grants: CompanionGrants): Record<Exclude<AccessLevel, 'none'>, CompanionSection[]> {
  return {
    edit: COMPANION_SECTIONS.filter((s) => grants[s] === 'edit'),
    view: COMPANION_SECTIONS.filter((s) => grants[s] === 'view'),
  };
}

/** The shared (non-none) sections, edit first — the chips of a list card. */
export function sharedSections(grants: CompanionGrants): Array<{ section: CompanionSection; level: 'view' | 'edit' }> {
  const { edit, view } = grantsByLevel(grants);
  return [
    ...edit.map((section) => ({ section, level: 'edit' as const })),
    ...view.map((section) => ({ section, level: 'view' as const })),
  ];
}

export function sameGrants(a: CompanionGrants, b: CompanionGrants): boolean {
  return COMPANION_SECTIONS.every((s) => a[s] === b[s]);
}

/** The name to show: what she typed, else the account's name, else null (caller shows a placeholder). */
export function companionName(c: Pick<OwnerCompanion, 'name' | 'displayName'>): string | null {
  const name = (c.displayName ?? '').trim() || (c.name ?? '').trim();
  return name || null;
}

/** Whole hours until an ISO instant, never below 0 (invite validity «تا ۲۴ ساعت معتبر»). */
export function hoursUntil(iso: string, now: Date = new Date()): number {
  const t = Date.parse(iso);
  if (Number.isNaN(t)) return 0;
  return Math.max(0, Math.ceil((t - now.getTime()) / 3_600_000));
}

/** The spouse that forms the owner's family: the active one first, else a pending invite. */
export function familySpouse(companions: readonly OwnerCompanion[]): OwnerCompanion | null {
  const spouses = companions.filter((c) => c.type === 'spouse');
  return spouses.find((c) => c.status === 'active') ?? spouses[0] ?? null;
}

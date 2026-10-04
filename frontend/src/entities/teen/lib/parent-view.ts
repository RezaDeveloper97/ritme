import {
  TEEN_SHARE_KEYS,
  type TeenGrants,
  type TeenParentLink,
  type TeenParentView,
  type TeenShareKey,
} from '../model/types';

/**
 * What a parent sees with these grants: `parent_preview` (everything) masked
 * section by section — the same rule the API applies to GET /teen/linked. An
 * empty note shows nothing.
 */
export function maskParentView(preview: TeenParentView, grants: TeenGrants): TeenParentView {
  const note = preview.note?.trim() ? preview.note.trim() : null;
  return {
    nextPeriodWeek: grants.teenPeriodWeek === 'view' ? preview.nextPeriodWeek : null,
    kitReady: grants.teenKit === 'view' ? preview.kitReady : null,
    note: grants.teenNotes === 'view' ? note : null,
  };
}

/** True when nothing of the view is filled (no section shared, or no note yet). */
export function isEmptyParentView(view: TeenParentView): boolean {
  return view.nextPeriodWeek === null && view.kitReady === null && view.note === null;
}

/** Grants with one section switched on / off. */
export function withShare(grants: TeenGrants, key: TeenShareKey, on: boolean): TeenGrants {
  return { ...grants, [key]: on ? 'view' : 'none' };
}

export function sameTeenGrants(a: TeenGrants, b: TeenGrants): boolean {
  return TEEN_SHARE_KEYS.every((k) => a[k] === b[k]);
}

/** The link the sharing screen manages: an active one first, else the pending invite. */
export function primaryParentLink(links: TeenParentLink[]): TeenParentLink | null {
  return links.find((l) => l.status === 'active') ?? links.find((l) => l.status === 'invited') ?? null;
}

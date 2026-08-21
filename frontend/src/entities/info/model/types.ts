// Domain shape for the admin-managed text screens (راهنما و پشتیبانی, حریم
// خصوصی, قوانین, درباره ما). Every screen is a list of boxes — a heading, a
// body, and an optional call-to-action link — and the backend already resolved
// them to the active locale, so nothing here is bilingual.

/** Which text screen a set of boxes belongs to. */
export const INFO_GROUPS = ['help', 'privacy', 'terms', 'about'] as const;

export type InfoGroup = (typeof INFO_GROUPS)[number];

export function isInfoGroup(value: string): value is InfoGroup {
  return (INFO_GROUPS as readonly string[]).includes(value);
}

export interface InfoSection {
  id: number;
  heading: string;
  body: string;
  /** Both link fields are present together or not at all — see the API. */
  linkLabel: string | null;
  linkUrl: string | null;
}

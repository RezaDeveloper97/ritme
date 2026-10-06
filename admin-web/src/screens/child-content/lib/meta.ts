/**
 * The child catalogs (B-N5-02 / B-N5-09, admin-api.md §18): catalog_items groups and the `meta` shape
 * backend-go/internal/children reads. An item whose meta lacks a required key is silently skipped by the app,
 * so the forms validate it here before saving.
 */

/** The editable kinds, their catalog group and the list page they belong to. */
export const KINDS = {
  vaccines: { group: 'child_vaccines', page: 'vaccines' },
  milestones: { group: 'child_milestones', page: 'milestones' },
  activities: { group: 'child_milestone_activities', page: 'milestones' },
  notes: { group: 'child_milestone_notes', page: 'milestones' },
  'age-notes': { group: 'child_age_notes', page: 'milestones' },
  learn: { group: 'child_learn', page: 'learn' },
} as const;

export type ChildKind = keyof typeof KINDS;
export const CHILD_KINDS = Object.keys(KINDS) as ChildKind[];

export function isChildKind(value: string): value is ChildKind {
  return (CHILD_KINDS as string[]).includes(value);
}

/** The kinds of the milestones page, in section order. */
export const MONTH_KINDS = ['milestones', 'activities', 'notes', 'age-notes'] as const satisfies readonly ChildKind[];

/** The visits of the seeded Iran national schedule, in age order (a new snake-case code is allowed too). */
export const VISITS = ['birth', 'm2', 'm4', 'm6', 'm12', 'm18', 'y6'] as const;
/** The seeded age of each visit, used to prefill a new dose. */
export const VISIT_MONTHS: Record<string, number> = { birth: 0, m2: 2, m4: 4, m6: 6, m12: 12, m18: 18, y6: 72 };

export const DOMAINS = ['social', 'language', 'motor', 'cognitive'] as const;
export const TOPICS = ['sleep', 'feeding', 'play', 'health', 'mother'] as const;

/** WHO average month (365.25 / 12): the approximate day of an age in months (the app uses calendar months). */
export const DAYS_PER_MONTH = 30.4375;

export function approxDays(months: number): number {
  return Math.round(months * DAYS_PER_MONTH);
}

/** Months 0–120 (10 years) are accepted by the forms; the app's age bands are data. */
export const MAX_MONTHS = 120;

const VISIT_RE = /^[a-z][a-z0-9_]*$/;

/** The form state of every kind (strings, as the inputs hold them). */
export interface MetaDraft {
  visit: string;
  ageMonths: string;
  domain: string;
  topic: string;
  fromMonths: string;
  toMonths: string;
  minutes: string;
  featured: boolean;
  articleSlug: string;
  adminNote: string;
}

const str = (v: unknown) => (typeof v === 'string' ? v : typeof v === 'number' ? String(v) : '');

/** Stored meta → the form state. */
export function draftOf(meta: Record<string, unknown> | null | undefined): MetaDraft {
  const m = meta ?? {};
  return {
    visit: str(m.visit),
    ageMonths: str(m.age_months),
    domain: str(m.domain),
    topic: str(m.topic),
    fromMonths: str(m.from_months),
    toMonths: str(m.to_months),
    minutes: str(m.minutes),
    featured: m.featured === true,
    articleSlug: str(m.article_slug),
    adminNote: str(m.admin_note),
  };
}

export type MetaError = 'required' | 'months' | 'visit' | 'range' | 'minutes' | 'slug';
export type MetaErrors = Partial<Record<keyof MetaDraft, MetaError>>;

function months(text: string): number | null {
  if (!/^\d+$/.test(text.trim())) return null;
  const n = Number(text.trim());
  return n <= MAX_MONTHS ? n : null;
}

/**
 * The form state → the meta to store, or the field errors. Keys of the stored meta this form does not edit
 * are kept (`previous`), so an admin never loses data another tool wrote.
 */
export function buildMeta(
  kind: ChildKind,
  draft: MetaDraft,
  previous: Record<string, unknown> | null = null,
): { ok: true; meta: Record<string, unknown> } | { ok: false; errors: MetaErrors } {
  const out: Record<string, unknown> = { ...(previous ?? {}) };
  const errors: MetaErrors = {};
  const setOptional = (key: string, value: unknown) => {
    if (value === null || value === '') delete out[key];
    else out[key] = value;
  };

  if (kind === 'learn') {
    for (const k of ['visit', 'age_months', 'domain']) delete out[k];
    if (!(TOPICS as readonly string[]).includes(draft.topic)) errors.topic = 'required';
    const from = months(draft.fromMonths);
    const to = months(draft.toMonths);
    if (draft.fromMonths.trim() === '') errors.fromMonths = 'required';
    else if (from === null) errors.fromMonths = 'months';
    if (draft.toMonths.trim() === '') errors.toMonths = 'required';
    else if (to === null) errors.toMonths = 'months';
    else if (from !== null && to < from) errors.toMonths = 'range';
    let minutes: number | null = null;
    if (draft.minutes.trim() !== '') {
      minutes = /^\d+$/.test(draft.minutes.trim()) ? Number(draft.minutes.trim()) : NaN;
      if (Number.isNaN(minutes) || minutes > 240) errors.minutes = 'minutes';
    }
    const slug = draft.articleSlug.trim();
    if (slug && !/^[a-z0-9][a-z0-9-]*$/i.test(slug)) errors.articleSlug = 'slug';
    out.topic = draft.topic;
    out.from_months = from;
    out.to_months = to;
    out.minutes = minutes ?? 0;
    out.featured = draft.featured;
    setOptional('article_slug', slug);
  } else {
    const age = months(draft.ageMonths);
    if (draft.ageMonths.trim() === '') errors.ageMonths = 'required';
    else if (age === null) errors.ageMonths = 'months';
    out.age_months = age;
    if (kind === 'vaccines') {
      const visit = draft.visit.trim();
      if (!visit) errors.visit = 'required';
      else if (!VISIT_RE.test(visit) || visit.length > 32) errors.visit = 'visit';
      out.visit = visit;
      setOptional('admin_note', draft.adminNote.trim());
    }
    if (kind === 'milestones') {
      if (!(DOMAINS as readonly string[]).includes(draft.domain)) errors.domain = 'required';
      out.domain = draft.domain;
    }
  }
  return Object.keys(errors).length ? { ok: false, errors } : { ok: true, meta: out };
}

/** A row's age in months (null when its meta has none). */
export function ageOf(meta: Record<string, unknown> | null | undefined): number | null {
  const v = meta?.age_months;
  return typeof v === 'number' && Number.isInteger(v) && v >= 0 ? v : null;
}

/** The distinct month bands of rows, ascending (rows without an age are left out). */
export function bandsOf(rows: ReadonlyArray<{ meta: Record<string, unknown> | null }>): number[] {
  const set = new Set<number>();
  for (const r of rows) {
    const a = ageOf(r.meta);
    if (a !== null) set.add(a);
  }
  return [...set].sort((a, b) => a - b);
}

export interface VisitGroup<T> {
  visit: string;
  ageMonths: number | null;
  rows: T[];
}

/**
 * Doses grouped into visits the way the app builds the schedule: catalog order, a visit sits where its first dose
 * is; rows without a visit are collected under `''` (the app skips them).
 */
export function groupByVisit<T extends { meta: Record<string, unknown> | null }>(rows: readonly T[]): VisitGroup<T>[] {
  const out: VisitGroup<T>[] = [];
  const index = new Map<string, number>();
  for (const r of rows) {
    const visit = typeof r.meta?.visit === 'string' ? r.meta.visit : '';
    let i = index.get(visit);
    if (i === undefined) {
      out.push({ visit, ageMonths: ageOf(r.meta), rows: [] });
      i = out.length - 1;
      index.set(visit, i);
    }
    out[i]?.rows.push(r);
  }
  return out;
}

/** Whether a row would be skipped by the app (its meta lacks what the group needs). */
export function isIncomplete(kind: ChildKind, meta: Record<string, unknown> | null): boolean {
  return !buildMeta(kind, draftOf(meta)).ok;
}

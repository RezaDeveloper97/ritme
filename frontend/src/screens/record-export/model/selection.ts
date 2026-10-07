import { create } from 'zustand';

import type { ReportRange, ReportSection, ReportSelection } from '@/entities/health-record';

/**
 * The «چه چیزهایی در گزارش باشد؟» toggles (nbl_Record_Export) and the API sections each one includes. The selection
 * lives in memory only (never in the URL or storage): the export screen writes it, the preview reads it.
 */
export const REPORT_GROUPS = ['basics', 'health', 'cycle', 'vitals', 'checkups', 'pregnancies'] as const;
export type ReportGroup = (typeof REPORT_GROUPS)[number];

const GROUP_SECTIONS: Record<ReportGroup, ReportSection[]> = {
  basics: ['basics'],
  health: ['conditions', 'medications', 'allergies'],
  cycle: ['cycle'],
  vitals: ['vitals'],
  checkups: ['checkups', 'labs'],
  pregnancies: ['pregnancies'],
};

/** Artboard defaults: the first four on, checkups off; pregnancies (not on the artboard) off. */
export const DEFAULT_GROUPS: Record<ReportGroup, boolean> = {
  basics: true,
  health: true,
  cycle: true,
  vitals: true,
  checkups: false,
  pregnancies: false,
};

/**
 * `?section=<group>` (B-N6-04b): the share entry points («اشتراک با پزشک» on the vitals report → `vitals`, on a lab
 * result → `checkups`) open the builder with that group preselected. Unknown values are ignored.
 */
export const SECTION_PARAM = 'section';

/** The group named by a `?section=` value, or null. */
export function parseGroup(value: string | null): ReportGroup | null {
  return (REPORT_GROUPS as readonly string[]).includes(value ?? '') ? (value as ReportGroup) : null;
}

/** Toggles for an entry point: the basics (who the patient is) plus the group it came from, nothing else. */
export function preselectedGroups(group: ReportGroup): Record<ReportGroup, boolean> {
  const out = Object.fromEntries(REPORT_GROUPS.map((g) => [g, false])) as Record<ReportGroup, boolean>;
  return { ...out, basics: true, [group]: true };
}

/** API section keys of the chosen groups, in API screen order. */
export function sectionsOf(groups: Record<ReportGroup, boolean>): ReportSection[] {
  const picked = new Set(REPORT_GROUPS.filter((g) => groups[g]).flatMap((g) => GROUP_SECTIONS[g]));
  const order: ReportSection[] = ['basics', 'conditions', 'medications', 'allergies', 'cycle', 'vitals', 'pregnancies', 'checkups', 'labs'];
  return order.filter((s) => picked.has(s));
}

/** Whether `from` (Gregorian YYYY-MM-DD) is an accepted custom start: before `today`, at most 3 years back. */
export function isValidFrom(from: string | null, today: string): boolean {
  if (!from || !/^\d{4}-\d{2}-\d{2}$/.test(from)) return false;
  const [y, m, d] = today.split('-').map(Number) as [number, number, number];
  const min = `${String(y - 3).padStart(4, '0')}-${String(m).padStart(2, '0')}-${String(d).padStart(2, '0')}`;
  return from >= min && from < today;
}

interface ReportDraft {
  range: ReportRange;
  from: string | null;
  groups: Record<ReportGroup, boolean>;
  question: string;
  setRange: (range: ReportRange, from?: string | null) => void;
  toggle: (group: ReportGroup, on: boolean) => void;
  setQuestion: (q: string) => void;
  preselect: (group: ReportGroup) => void;
}

export const useReportDraft = create<ReportDraft>((set) => ({
  range: '3m',
  from: null,
  groups: DEFAULT_GROUPS,
  question: '',
  setRange: (range, from = null) => set({ range, from: range === 'custom' ? from : null }),
  toggle: (group, on) => set((s) => ({ groups: { ...s.groups, [group]: on } })),
  setQuestion: (question) => set({ question }),
  preselect: (group) => set({ groups: preselectedGroups(group) }),
}));

/** The draft as the API selection. */
export function selectionOf(d: Pick<ReportDraft, 'range' | 'from' | 'groups' | 'question'>): ReportSelection {
  return { range: d.range, from: d.from, sections: sectionsOf(d.groups), question: d.question };
}

/** The public URL of a share token (the page renders the frozen report read-only). */
export function sharedReportUrl(origin: string, locale: string, token: string): string {
  return `${origin}/${locale}/shared/report/${token}`;
}

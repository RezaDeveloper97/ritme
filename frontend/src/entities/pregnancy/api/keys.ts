import type { ReportRange } from '../model/v2-types';

const ALL = ['pregnancy'] as const;
const V2 = [...ALL, 'v2'] as const;

/**
 * Query-key factory for pregnancy mode (CLAUDE.md §8). Weekly resources are
 * keyed by week, daily ones by API date, so each caches independently. All
 * reads and post-mutation invalidation go through here — never raw arrays.
 *
 * `v2.*` (M7, `/pregnancy/v2/*`) nests under `all`, so a v1 mode switch
 * (`invalidateQueries(pregnancyKeys.all)`) refreshes the v2 screens too. The
 * `*All()` keys are prefixes for invalidating every entry of one resource.
 * No locale in the keys: the API localizes off `Accept-Language` and a locale
 * switch clears the whole cache (`features/switch-locale`).
 */
export const pregnancyKeys = {
  all: ALL,
  status: () => [...ALL, 'status'] as const,
  profile: () => [...ALL, 'profile'] as const,
  enums: () => [...ALL, 'enums'] as const,
  symptomEnums: () => [...ALL, 'symptom-enums'] as const,
  weeklyEnums: () => [...ALL, 'weekly-enums'] as const,
  content: (week: number, locale: string) => [...ALL, 'content', week, locale] as const,
  symptom: (date: string) => [...ALL, 'symptom', date] as const,
  weeklyLogAll: () => [...ALL, 'weekly-log'] as const,
  weeklyLog: (week: number) => [...ALL, 'weekly-log', week] as const,
  fetalMovement: (date: string) => [...ALL, 'fetal-movement', date] as const,
  alerts: () => [...ALL, 'alerts'] as const,
  alertSummary: () => [...ALL, 'alert-summary'] as const,

  v2: {
    all: () => V2,
    setupCopy: () => [...V2, 'setup-copy'] as const,
    today: () => [...V2, 'today'] as const,
    weekAll: () => [...V2, 'week'] as const,
    week: (week: number) => [...V2, 'week', week] as const,
    dayAll: () => [...V2, 'day'] as const,
    day: (date: string) => [...V2, 'day', date] as const,
    calendarAll: () => [...V2, 'calendar'] as const,
    /** `month` = `YYYY-MM` (Gregorian anchor; the server renders the Jalali label). */
    calendar: (month: string) => [...V2, 'calendar', month] as const,
    alerts: () => [...V2, 'alerts'] as const,
    datingPreviewAll: () => [...V2, 'dating-preview'] as const,
    /** Keyed by the serialized input — the preview is a pure function of it. */
    datingPreview: (input: string) => [...V2, 'dating-preview', input] as const,
    reportAll: () => [...V2, 'report'] as const,
    report: (range: ReportRange) => [...V2, 'report', range.from, range.to] as const,
  },
};

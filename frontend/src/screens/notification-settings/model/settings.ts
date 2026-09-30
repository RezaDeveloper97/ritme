import { z } from 'zod';

/**
 * Notification settings (B-N1-11) — GET/PUT /profile/notification-settings.
 * The server owns the category list and order (cycle · health · other); the
 * screen only renders what it gets, so a new category needs no client release
 * beyond its copy.
 */

export type CategoryCode =
  | 'before_period'
  | 'pms'
  | 'fertile_window'
  | 'daily_log'
  | 'medications'
  | 'appointments'
  | 'checkups'
  | 'vitals'
  | 'learning'
  | 'companion'
  | 'articles';

export type GroupCode = 'cycle' | 'health' | 'other';

/** The codes this build has copy for; unknown codes from a newer server are hidden. */
export const KNOWN_CATEGORIES: readonly CategoryCode[] = [
  'before_period',
  'pms',
  'fertile_window',
  'daily_log',
  'medications',
  'appointments',
  'checkups',
  'vitals',
  'learning',
  'companion',
  'articles',
];
const KNOWN_GROUPS: readonly GroupCode[] = ['cycle', 'health', 'other'];

export interface CategoryItem {
  code: CategoryCode;
  enabled: boolean;
}

export interface CategoryGroup {
  code: GroupCode;
  items: CategoryItem[];
}

export interface NotificationSettings {
  groups: CategoryGroup[];
  quietHours: { enabled: boolean; start: string; end: string };
  neutralCopy: boolean;
}

/** A partial PUT body — omitted keys keep their stored value. */
export interface SettingsPatch {
  categories?: Partial<Record<CategoryCode, boolean>>;
  quiet_hours?: { enabled?: boolean; start?: string; end?: string };
  neutral_copy?: boolean;
}

const isCategory = (code: string): code is CategoryCode => (KNOWN_CATEGORIES as readonly string[]).includes(code);
const isGroup = (code: string): code is GroupCode => (KNOWN_GROUPS as readonly string[]).includes(code);

/** Boundary parser (CLAUDE.md §10): unknown groups / categories are dropped. */
export const settingsSchema = z
  .object({
    groups: z
      .array(
        z.object({
          code: z.string(),
          items: z.array(z.object({ code: z.string(), enabled: z.boolean() })).default([]),
        }),
      )
      .default([]),
    quiet_hours: z
      .object({
        enabled: z.boolean().default(true),
        start: z.string().default('23:00'),
        end: z.string().default('08:00'),
      })
      .default({}),
    neutral_copy: z.boolean().default(true),
  })
  .transform(
    (raw): NotificationSettings => ({
      groups: raw.groups.flatMap((g) => {
        if (!isGroup(g.code)) return [];
        const items = g.items.flatMap((i) => (isCategory(i.code) ? [{ code: i.code, enabled: i.enabled }] : []));
        return items.length ? [{ code: g.code, items }] : [];
      }),
      quietHours: raw.quiet_hours,
      neutralCopy: raw.neutral_copy,
    }),
  );

/** The settings with `patch` applied — the optimistic cache value while a PUT is in flight. */
export function applyPatch(s: NotificationSettings, patch: SettingsPatch): NotificationSettings {
  const cats = patch.categories ?? {};
  return {
    groups: s.groups.map((g) => ({
      ...g,
      items: g.items.map((i) => (cats[i.code] === undefined ? i : { ...i, enabled: Boolean(cats[i.code]) })),
    })),
    quietHours: { ...s.quietHours, ...patch.quiet_hours },
    neutralCopy: patch.neutral_copy ?? s.neutralCopy,
  };
}

const CLOCK = /^([01]\d|2[0-3]):([0-5]\d)$/;

/** Whether `value` is the `HH:MM` the API accepts (what `<input type="time">` yields). */
export function isClock(value: string): boolean {
  return CLOCK.test(value);
}

/** «08:00» → «8:00» as drawn on the board; digits are localized by the caller. */
export function displayClock(value: string): string {
  const m = CLOCK.exec(value);
  return m ? `${Number(m[1])}:${m[2]}` : value;
}

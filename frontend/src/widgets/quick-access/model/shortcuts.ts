import { ALL_MODULES_ON, type AppModules, moduleOfRoute } from '@/entities/app-module';
import type { IconName, Tone } from '@/shared/ui';

/**
 * Today «دسترسی سریع» shortcuts (CB-NAV-03, nbd_Nav_Today / IA_Map «دسترسی سریع قابل تنظیم»).
 *
 * Every shortcut is an in-app route. `href: null` = its screen hasn't shipped yet (doctor directory B-N7 /
 * CB-DIR, condition programs CB-COND, courses B-N8 front end): never offered, never linked — the owning task
 * sets the path and it appears in the edit sheet. `modes` limits a shortcut to the life stages it makes sense
 * in (omitted = every owner mode).
 */
export type QuickMode = 'cycle' | 'ttc' | 'ivf' | 'pregnancy' | 'postpartum' | 'menopause' | 'teen';

export interface Shortcut {
  id: string;
  href: string | null;
  icon: IconName;
  tone: Tone;
  modes?: readonly QuickMode[];
  /** Opens this sheet (`app/sheets/registry.tsx`) instead of navigating; `href` still names the owning module. */
  sheet?: string;
  /** Teen only with an active parent link (IA_Nav «نوجوان»: doctor & labs ride on the mother; CB-NAV-04b). */
  teenNeedsParent?: boolean;
}

const ADULT: readonly QuickMode[] = ['cycle', 'ttc', 'ivf', 'pregnancy', 'postpartum', 'menopause'];

export const SHORTCUTS: readonly Shortcut[] = [
  { id: 'record', href: '/record', icon: 'fileDoc', tone: 'brand' },
  // IA_Nav «نوجوان»: doctor & labs only with a parent link (CB-NAV-04b) — hidden from a teen without one.
  { id: 'doctor', href: '/services/doctors', icon: 'stetho', tone: 'data', modes: [...ADULT, 'teen'], teenNeedsParent: true },
  { id: 'painDiary', href: '/programs/pain', icon: 'flame', tone: 'period', modes: ['cycle', 'ttc', 'postpartum', 'menopause'] },
  { id: 'courses', href: '/learn', icon: 'gradCap', tone: 'warm' },
  { id: 'checkups', href: '/checkups', icon: 'clock', tone: 'data' },
  { id: 'reminders', href: '/reminders', icon: 'bellRing', tone: 'brand' },
  { id: 'vitals', href: '/vitals', sheet: 'vitals-add', icon: 'heartLine', tone: 'period' },
  { id: 'labs', href: '/labs', icon: 'flask', tone: 'period', modes: [...ADULT, 'teen'], teenNeedsParent: true },
  { id: 'analysis', href: '/analysis', icon: 'chart', tone: 'data' },
  { id: 'todo', href: '/todo', icon: 'todo', tone: 'brand' },
  { id: 'contraception', href: '/contraception', icon: 'pill', tone: 'brand', modes: ['cycle', 'postpartum', 'menopause'] },
  { id: 'children', href: '/children', icon: 'user', tone: 'bloom', modes: ['cycle', 'ttc', 'ivf', 'pregnancy', 'postpartum', 'menopause'] },
  { id: 'kicks', href: '/pregnancy/kicks', icon: 'hand', tone: 'bloom', modes: ['pregnancy'] },
  { id: 'hotFlash', href: '/menopause/hot-flash', icon: 'flame', tone: 'warm', modes: ['menopause'] },
  { id: 'ivfMeds', href: '/ivf/meds', icon: 'syringe', tone: 'warm', modes: ['ivf'] },
  { id: 'search', href: '/search', icon: 'search', tone: 'neutral' },
];

/** Up to this many shortcuts (two rows of five with the «ویرایش» tile). */
export const QUICK_ACCESS_MAX = 9;

/** The board's picks first (record, doctor, vitals, pain diary, courses), then mode-specific ones, then fillers. */
const DEFAULTS: Record<QuickMode, readonly string[]> = {
  cycle: ['record', 'doctor', 'vitals', 'painDiary', 'courses', 'checkups', 'reminders', 'analysis'],
  ttc: ['record', 'doctor', 'vitals', 'courses', 'checkups', 'reminders', 'analysis'],
  ivf: ['ivfMeds', 'record', 'doctor', 'vitals', 'labs', 'reminders'],
  pregnancy: ['kicks', 'record', 'vitals', 'doctor', 'courses', 'checkups', 'reminders'],
  postpartum: ['children', 'record', 'vitals', 'doctor', 'courses', 'checkups', 'reminders'],
  menopause: ['hotFlash', 'record', 'vitals', 'doctor', 'painDiary', 'checkups', 'reminders'],
  teen: ['reminders', 'doctor', 'analysis', 'search'],
};

/** A shortcut into a module the admin switched off is never offered. */
function isModuleOff(href: string, modules: AppModules): boolean {
  const owner = moduleOfRoute(href);
  return owner !== null && !modules[owner];
}

/** How many defaults show (the board's single row of four + «ویرایش»). */
const DEFAULT_COUNT = 4;

/** Shortcuts that can be offered in `mode` (screen shipped + mode allowed + module on), in catalogue order. */
export function availableShortcuts(mode: QuickMode, parentLinked = false, modules: AppModules = ALL_MODULES_ON): Shortcut[] {
  return SHORTCUTS.filter(
    (s) =>
      s.href !== null &&
      (!s.modes || s.modes.includes(mode)) &&
      !isModuleOff(s.href, modules) &&
      !(mode === 'teen' && s.teenNeedsParent && !parentLinked),
  );
}

/**
 * What the Today row shows: the stored picks that are still available in `mode` (in the stored order),
 * else the mode's defaults. An empty stored list is a choice (show only «ویرایش»); `null` = never edited.
 */
export function visibleShortcuts(
  mode: QuickMode,
  stored: readonly string[] | null,
  parentLinked = false,
  modules: AppModules = ALL_MODULES_ON,
): Shortcut[] {
  const available = new Map(availableShortcuts(mode, parentLinked, modules).map((s) => [s.id, s]));
  const pick = (ids: readonly string[]) =>
    ids.map((id) => available.get(id)).filter((s): s is Shortcut => Boolean(s));
  if (stored !== null) {
    const kept = pick(stored).slice(0, QUICK_ACCESS_MAX);
    // Picks that all became unavailable (a mode switch) fall back to the defaults rather than an empty row.
    if (kept.length > 0 || stored.length === 0) return kept;
  }
  return pick(DEFAULTS[mode]).slice(0, DEFAULT_COUNT);
}

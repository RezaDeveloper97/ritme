import type { PilotCatalog, PilotStep } from '@/shared/pilot';

import { CYCLE_ROUTES, CYCLE_SHEETS, CYCLE_TARGETS } from './groups/cycle';
import { FAMILY_ROUTES, FAMILY_SHEETS, FAMILY_TARGETS } from './groups/family';
import { CARE_ROUTES, CARE_SHEETS, CARE_TARGETS } from './groups/care';
import { PROGRAMS_ROUTES, PROGRAMS_SHEETS, PROGRAMS_TARGETS } from './groups/programs';
import { ACCOUNT_ROUTES, ACCOUNT_SHEETS, ACCOUNT_TARGETS } from './groups/account';
import { SERVICES_ROUTES, SERVICES_SHEETS, SERVICES_TARGETS } from './groups/services';

/**
 * THE autopilot catalog — every route, sheet and control a script may use (frontend/CLAUDE.md §11.6,
 * docs/autopilot.md). It feeds the agent manifest (what the model is told exists) and tells the engine how
 * to reach a control that isn't on screen.
 *
 * Rule: a control that gets `pilot('…')` / `data-pilot` is registered here in the same change, and so is a
 * new route or sheet users should be able to ask for. `catalog.test.ts` fails otherwise.
 *
 * Lives in `app` for the same reason as `sheets/registry.tsx`: it is the one layer allowed to know every
 * screen. Titles are message keys of the `autopilotCatalog` namespace (messages/<locale>/autopilot-catalog.json).
 */

const LOG_SHEET = { sheet: 'log' } as const;

/** `bleeding.flow` → `bleeding`. */
const categoryOf = (key: string): string => key.split('.')[0];

/** A log param sits inside its category's accordion, sometimes behind «جزئیات بیشتر». */
const revealLogParam = (key: string): PilotStep[] => [
  { do: 'expand', target: `log.cat:${categoryOf(key)}` },
  // Most categories have no «جزئیات بیشتر»: don't wait long for one.
  { do: 'expand', target: `log.more:${categoryOf(key)}`, optional: true, timeoutMs: 400 },
];

export const PILOT_CATALOG: PilotCatalog = {
  routes: [
    { route: '/home', titleKey: 'routes.home' },
    { route: '/calendar', titleKey: 'routes.calendar' },
    { route: '/cycle', titleKey: 'routes.cycle' },
    { route: '/log', titleKey: 'routes.log' },
    { route: '/log/customize', titleKey: 'routes.logCustomize' },
    { route: '/profile', titleKey: 'routes.profile' },
    { route: '/profile/notifications', titleKey: 'routes.profileNotifications' },
    { route: '/profile/credits', titleKey: 'routes.profileCredits' },
    { route: '/plus/credits', titleKey: 'routes.plusCredits' },
    { route: '/services', titleKey: 'routes.services' },
    { route: '/analysis', titleKey: 'routes.analysis' },
    { route: '/reminders', titleKey: 'routes.reminders', module: 'reminders' },
    { route: '/checkups', titleKey: 'routes.checkups' },
    { route: '/vitals', titleKey: 'routes.vitals' },
    { route: '/todo', titleKey: 'routes.todo', module: 'todo' },
    { route: '/labs', titleKey: 'routes.labs' },
    { route: '/record', titleKey: 'routes.record', module: 'record' },
    { route: '/learn', titleKey: 'routes.learn', module: 'learning' },
    { route: '/assistant', titleKey: 'routes.assistant' },
    { route: '/search', titleKey: 'routes.search' },
    { route: '/fertility/log', titleKey: 'routes.fertilityLog', module: 'ttc' },
    { route: '/fertility/bbt', titleKey: 'routes.fertilityBbt', module: 'ttc' },
    { route: '/pregnancy', titleKey: 'routes.pregnancy' },
    { route: '/pregnancy/log', titleKey: 'routes.pregnancyLog' },
    { route: '/menopause/log', titleKey: 'routes.menopauseLog', module: 'menopause' },
    { route: '/children', titleKey: 'routes.children', module: 'children' },
    ...CYCLE_ROUTES,
    ...FAMILY_ROUTES,
    ...CARE_ROUTES,
    ...PROGRAMS_ROUTES,
    ...ACCOUNT_ROUTES,
    ...SERVICES_ROUTES,
  ],

  sheets: [
    { sheet: 'log', titleKey: 'sheets.log', arg: 'YYYY-MM-DD (optional, today when absent)' },
    { sheet: 'notifications', titleKey: 'sheets.notifications' },
    { sheet: 'todo-add', titleKey: 'sheets.todoAdd', module: 'todo' },
    { sheet: 'vitals-add', titleKey: 'sheets.vitalsAdd' },
    ...CYCLE_SHEETS,
    ...FAMILY_SHEETS,
    ...CARE_SHEETS,
    ...PROGRAMS_SHEETS,
    ...ACCOUNT_SHEETS,
    ...SERVICES_SHEETS,
  ],

  targets: [
    // ── Log sheet v2 (features/log-day). Members of the families come from /logs/taxonomy. ──
    { id: 'log.save', value: 'action', titleKey: 'targets.logSave', surface: LOG_SHEET },
    { id: 'log.cat:*', value: 'section', surface: LOG_SHEET },
    { id: 'log.more:*', value: 'section', surface: LOG_SHEET, reveal: (key) => [{ do: 'expand', target: `log.cat:${key}` }] },
    { id: 'log.param:*', value: 'single', surface: LOG_SHEET, reveal: revealLogParam },
    { id: 'log.level:*', value: 'single', surface: LOG_SHEET, reveal: revealLogParam },
    { id: 'log.date:*', value: 'action', surface: LOG_SHEET },

    // ── Shell ──
    { id: 'nav.log', value: 'action', titleKey: 'targets.navLog' },
    { id: 'nav.tab:*', value: 'action' },
    { id: 'sheet.close', value: 'action', titleKey: 'targets.sheetClose' },

    // ── The rest of the app, by area (./groups) ──
    ...CYCLE_TARGETS,
    ...FAMILY_TARGETS,
    ...CARE_TARGETS,
    ...PROGRAMS_TARGETS,
    ...ACCOUNT_TARGETS,
    ...SERVICES_TARGETS,
  ],
};

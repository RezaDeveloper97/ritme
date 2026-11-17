import { matchesAny, type PilotCatalog, type PilotValueType } from './catalog';
import { PILOT_ACTIONS, SCRIPT_VERSION } from './script';
import { isPattern } from './target';

/**
 * The agent manifest: the vocabulary sent to the model next to a request (a voice note's transcript), so
 * it can answer with a version-1 script. It lists what exists — routes, sheets, targets, the values each
 * accepts — and nothing about the user: no logged data, no dates, no names (§11).
 */

export interface ManifestOption {
  value: string;
  label: string;
}

export interface ManifestTarget {
  id: string;
  kind: PilotValueType;
  label: string;
  /** The section a field sits in («خونریزی»), to disambiguate similar labels. */
  group?: string;
  options?: ManifestOption[];
  min?: number;
  max?: number;
  unit?: string;
  maxLength?: number;
}

export interface AutopilotManifest {
  version: typeof SCRIPT_VERSION;
  guide: string;
  actions: readonly string[];
  routes: { route: string; title: string }[];
  sheets: { sheet: string; title: string; arg?: string }[];
  targets: ManifestTarget[];
}

/** Output rules for the model. English on purpose: it is an instruction to the agent, never shown to a user. */
export const MANIFEST_GUIDE = [
  'Reply with ONE JSON object and nothing else: {"version":1,"steps":[...]}.',
  'Each step is {"do": <action>, ...}. Actions: open {route | sheet, arg?}, set {target, value}, tap {target}, expand {target}, collapse {target}, wait {ms | target}, close {}.',
  'Use only the routes, sheets and target ids listed here. Never invent an id.',
  'set values by kind — single: one option "value" (or null to clear); multi: an array of option values (the exact final set); bool: true/false; number/integer: a number inside min..max; text: a string; date: "YYYY-MM-DD" (Gregorian); time: "HH:MM" (24h). Use option "value" codes, never labels.',
  'Do not add steps to open a sheet or a section before a set: the app reaches the field by itself. Use open only when the user asks to go somewhere.',
  'Only fill what the user actually said. Do not tap a save target unless the user asked to save.',
  'If nothing in the request maps to this list, reply {"version":1,"steps":[]}.',
  'Optional per step: "say" (a short caption in the user\'s language), "optional": true.',
].join('\n');

export interface BuildManifestInput {
  catalog: PilotCatalog;
  /** Concrete members of the catalog's families, built from data (the log taxonomy). */
  dynamic?: readonly ManifestTarget[];
  /** Translator of the `autopilot` namespace. */
  t: (key: string) => string;
  /** Is this app module on? Omitted = everything is. */
  moduleEnabled?: (module: string) => boolean;
  blockedTargets?: readonly string[];
  blockedRoutes?: readonly string[];
}

export function buildAutopilotManifest(input: BuildManifestInput): AutopilotManifest {
  const { catalog, t } = input;
  const on = (module?: string) => !module || !input.moduleEnabled || input.moduleEnabled(module);
  const blocked = input.blockedTargets ?? [];
  const blockedRoutes = input.blockedRoutes ?? [];

  const fixed: ManifestTarget[] = catalog.targets
    .filter((e) => !isPattern(e.id) && on(e.module))
    .map((e) => ({
      id: e.id,
      kind: e.value,
      label: e.titleKey ? t(e.titleKey) : e.id,
      ...(e.options ? { options: e.options.map((o) => ({ value: o.value, label: t(o.titleKey) })) } : {}),
      ...(e.min !== undefined ? { min: e.min } : {}),
      ...(e.max !== undefined ? { max: e.max } : {}),
      ...(e.unitKey ? { unit: t(e.unitKey) } : {}),
    }));

  // A dynamic target is only listed when the catalog declares its family: the engine can't reach the rest.
  const families = catalog.targets.filter((e) => isPattern(e.id) && on(e.module)).map((e) => e.id);
  const dynamic = (input.dynamic ?? []).filter((d) => matchesAny(families, d.id));

  const seen = new Set<string>();
  const targets = [...fixed, ...dynamic].filter((target) => {
    if (seen.has(target.id) || matchesAny(blocked, target.id)) return false;
    seen.add(target.id);
    return true;
  });

  return {
    version: SCRIPT_VERSION,
    guide: MANIFEST_GUIDE,
    actions: PILOT_ACTIONS,
    routes: catalog.routes
      .filter((r) => on(r.module) && !blockedRoutes.some((p) => r.route === p || r.route.startsWith(`${p}/`)))
      .map((r) => ({ route: r.route, title: t(r.titleKey) })),
    sheets: catalog.sheets.filter((s) => on(s.module)).map((s) => ({ sheet: s.sheet, title: t(s.titleKey), ...(s.arg ? { arg: s.arg } : {}) })),
    targets,
  };
}

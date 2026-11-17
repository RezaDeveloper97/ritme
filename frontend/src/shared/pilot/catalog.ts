import type { PilotStep } from './script';
import { isPattern, matchTargetId } from './target';

/**
 * The catalog: every route, sheet and control the autopilot may use, declared once
 * (`src/app/autopilot/catalog.ts`). It answers two questions from one list:
 *
 * - for the **agent** — "what exists and what may I set it to?" (→ the manifest it is sent with a request);
 * - for the **engine** — "this control isn't on screen; how do I get to it?" (`surface` + `reveal`).
 *
 * `shared` only knows the shape; the entries live in `app`, which is allowed to know every screen.
 */

/** What a target accepts, in the agent's terms (not the DOM's). */
export type PilotValueType =
  | 'action' // tap only
  | 'section' // expand / collapse
  | 'bool'
  | 'single' // one option code, or null
  | 'multi' // an array of option codes
  | 'number'
  | 'integer'
  | 'text'
  | 'date' // ISO `YYYY-MM-DD`
  | 'time'; // `HH:MM`

export interface PilotSurface {
  /** A registered sheet id: opened when the target isn't on screen. */
  sheet?: string;
  /** A route: navigated to when the target isn't on screen. */
  route?: string;
  /** Query the route needs for the target to show (`/pregnancy/log?tab=day`). */
  query?: Record<string, string>;
}

export interface CatalogTarget {
  /** Exact id (`log.save`) or a family (`log.param:*`) whose members come from data. */
  id: string;
  value: PilotValueType;
  /** Message key under the `autopilot` namespace — the label the agent reads. Families take theirs from data. */
  titleKey?: string;
  surface?: PilotSurface;
  /**
   * Steps that bring the target on screen once its surface is open (open the accordion it sits in, press
   * «جزئیات بیشتر»). Run in order until the target appears; `key` is the family member's key.
   */
  reveal?: (key: string) => PilotStep[];
  /** App module (`entities/app-module`) that must be on. */
  module?: string;
  /** Fixed choices of a `single` / `multi` target: the code scripts use and its title key. */
  options?: readonly { value: string; titleKey: string }[];
  /** Bounds and unit of a `number` / `integer` target. */
  min?: number;
  max?: number;
  unitKey?: string;
}

export interface CatalogRoute {
  route: string;
  titleKey: string;
  module?: string;
}

export interface CatalogSheet {
  sheet: string;
  titleKey: string;
  /** What `arg` means, for the agent (`YYYY-MM-DD`). */
  arg?: string;
  module?: string;
}

export interface PilotCatalog {
  routes: readonly CatalogRoute[];
  sheets: readonly CatalogSheet[];
  targets: readonly CatalogTarget[];
}

export const EMPTY_CATALOG: PilotCatalog = { routes: [], sheets: [], targets: [] };

export interface CatalogMatch {
  entry: CatalogTarget;
  /** The family member's key; `''` for an exact id. */
  key: string;
}

/** The entry a target id belongs to. Exact ids win over families. */
export function findCatalogTarget(catalog: PilotCatalog, id: string): CatalogMatch | null {
  let family: CatalogMatch | null = null;
  for (const entry of catalog.targets) {
    const key = matchTargetId(entry.id, id);
    if (key === null) continue;
    if (!isPattern(entry.id)) return { entry, key };
    family ??= { entry, key };
  }
  return family;
}

/** Does `id` fall under one of `patterns` (exact ids or `family:*`)? */
export function matchesAny(patterns: readonly string[], id: string): boolean {
  return patterns.some((p) => matchTargetId(p, id) !== null);
}

import type { LogCategory, LogDayValues } from '@/entities/health-log';
import { MANIFEST_GUIDE, SCRIPT_VERSION, type AutopilotManifest, type ManifestTarget, type PilotScript, type PilotStep } from '@/shared/pilot';

/**
 * Voice → autopilot, the pure part: which controls a voice note may touch, and what is left of a model's
 * script once everything else is dropped. The script is untrusted (§11.6): it came from a model that was
 * handed what the user said. Only `set` on a field the manifest listed survives — never `open`, never a
 * save, never a locator that isn't an id — and the engine's own `checkScript` still runs afterwards.
 */

/** The fields of the log sheet's accordions: one value each, reachable through the catalog's `reveal`. */
const FIELD_PREFIXES = ['log.param:', 'log.level:'] as const;

function isField(id: string): boolean {
  return FIELD_PREFIXES.some((p) => id.startsWith(p));
}

/** Enough for one spoken minute; the server caps at 40 and the engine at 60. */
export const MAX_VOICE_STEPS = 30;

/**
 * The manifest of the log sheet alone — what is sent next to the recording — from the sheet's own targets
 * (`logTargetsOf`). No routes, no sheets, no save button: a voice note fills the form, the user presses
 * «ذخیره» herself. Vocabulary only: nothing the user logged (§11).
 */
export function logSheetManifest(targets: readonly ManifestTarget[]): AutopilotManifest {
  return {
    version: SCRIPT_VERSION,
    guide: MANIFEST_GUIDE,
    actions: ['set'],
    routes: [],
    sheets: [],
    targets: targets.filter((t) => isField(t.id) && t.kind !== 'section' && t.kind !== 'action'),
  };
}

function selectedCodes(values: LogDayValues, category: string, param: string): string[] {
  const current = values[category]?.[param];
  if (Array.isArray(current)) return current.filter((c): c is string => typeof c === 'string');
  // A graded `items` param is stored as { <item>: { level, score } }.
  if (current && typeof current === 'object') return Object.keys(current);
  return [];
}

function paramOf(target: string, categories: readonly LogCategory[]): { category: string; param: string } | null {
  if (!target.startsWith('log.param:')) return null;
  const key = target.slice('log.param:'.length);
  const category = categories.find((c) => key.startsWith(`${c.code}.`));
  if (!category) return null;
  return { category: category.code, param: key.slice(category.code.length + 1) };
}

/**
 * The script the engine may run for one voice note, or null when nothing usable is left.
 *
 * A multi-select `set` means "the exact final set" to the engine, but someone who says «نفخ دارم» is adding
 * to her day, not replacing it — so what is already picked in the draft is kept and the new codes join it.
 */
export function voiceScript(
  script: { steps: readonly unknown[] },
  manifest: AutopilotManifest,
  values: LogDayValues,
  categories: readonly LogCategory[],
): PilotScript | null {
  const allowed = new Set(manifest.targets.map((t) => t.id));
  const steps: PilotStep[] = [];
  for (const raw of script.steps) {
    if (steps.length >= MAX_VOICE_STEPS) break;
    if (!raw || typeof raw !== 'object') continue;
    const step = raw as { do?: unknown; target?: unknown; value?: unknown };
    if (step.do !== 'set' || typeof step.target !== 'string' || !allowed.has(step.target)) continue;
    let value = step.value;
    if (Array.isArray(value)) {
      const codes = value.filter((c): c is string => typeof c === 'string');
      const at = paramOf(step.target, categories);
      const kept = at ? selectedCodes(values, at.category, at.param) : [];
      value = [...new Set([...kept, ...codes])];
    } else if (!(typeof value === 'string' || typeof value === 'boolean' || (typeof value === 'number' && Number.isFinite(value)))) {
      continue; // null would clear a field: a voice note never erases
    }
    steps.push({
      do: 'set',
      target: step.target,
      value: value as PilotStep extends { value: infer V } ? V : never,
      // One field the model got wrong must not stop the rest. A model's `say` is never kept: a caption is shown
      // as Ritme's own words, and what was said near the microphone must not be able to write one.
      optional: true,
    });
  }
  return steps.length ? { version: SCRIPT_VERSION, steps } : null;
}

/**
 * How the autopilot names things on screen.
 *
 * A target id is a stable, language-independent name: `<area>.<type>` or `<area>.<type>:<key>`
 * (`log.save`, `log.param:bleeding.flow`, `nav.tab:calendar`). Components put it on the DOM with
 * {@link pilot}; scripts and the agent manifest refer to it by the same string. Labels, classes and
 * `useId()` ids are none of those things — they change with the language, the design and the render.
 */

export const PILOT_ATTR = 'data-pilot';
export const PILOT_OPTION_ATTR = 'data-pilot-option';
export const PILOT_KIND_ATTR = 'data-pilot-kind';

/** How a control is operated. Inferred from the DOM when a component doesn't say. */
export type PilotKind =
  | 'button'
  | 'toggle' // aria-pressed chip
  | 'switch' // role=switch / checkbox / radio / tab
  | 'choice' // a group whose children carry data-pilot-option
  | 'text'
  | 'number' // a text input that commits on blur
  | 'stepper' // − <output> +
  | 'wheel' // scroll-snap column (WheelPicker)
  | 'ruler' // scroll-snap ruler (RulerPicker)
  | 'calendar' // month grid with prev/next (CalendarPicker)
  | 'select' // native <select>
  | 'disclosure'; // aria-expanded

export interface PilotAttrs {
  'data-pilot': string;
  'data-pilot-kind'?: PilotKind;
}

export interface PilotOptionAttrs {
  'data-pilot-option': string;
}

/** Attributes that make a control addressable: `<button {...pilot('log.save')}>`. */
export function pilot(id: string, kind?: PilotKind): PilotAttrs {
  return kind ? { [PILOT_ATTR]: id, [PILOT_KIND_ATTR]: kind } : { [PILOT_ATTR]: id };
}

/** Marks one option inside a `choice` group with its value code. */
export function pilotOption(value: string): PilotOptionAttrs {
  return { [PILOT_OPTION_ATTR]: value };
}

const ID_RE = /^[a-z][a-z0-9-]*(\.[a-z][a-z0-9-]*)+(:[A-Za-z0-9_.-]+)?$/;

export function isTargetId(value: string): boolean {
  return ID_RE.test(value);
}

export interface ParsedTargetId {
  /** Everything before the colon: `log.param`. */
  head: string;
  /** Everything after it (`bleeding.flow`), or `null` for a keyless id. */
  key: string | null;
}

export function parseTargetId(id: string): ParsedTargetId | null {
  if (!isTargetId(id)) return null;
  const colon = id.indexOf(':');
  return colon < 0 ? { head: id, key: null } : { head: id.slice(0, colon), key: id.slice(colon + 1) };
}

/** A catalog id is exact (`log.save`) or a family (`log.param:*`). */
export function isPattern(catalogId: string): boolean {
  return catalogId.endsWith(':*');
}

/**
 * Does `id` belong to `catalogId`? Returns the key for a family (`''` for an exact id), `null` otherwise.
 */
export function matchTargetId(catalogId: string, id: string): string | null {
  if (!isPattern(catalogId)) return catalogId === id ? '' : null;
  const prefix = catalogId.slice(0, -1);
  return id.startsWith(prefix) && id.length > prefix.length ? id.slice(prefix.length) : null;
}

/** A script's `target`: a catalog id, or a fallback locator for a place nobody instrumented yet. */
export type TargetRef = string | { role: string; name: string } | { css: string };

/** A value-free description for logs and reports (§11: never the user's data). */
export function describeTarget(ref: TargetRef): string {
  if (typeof ref === 'string') return ref;
  return 'css' in ref ? 'css' : `role:${ref.role}`;
}

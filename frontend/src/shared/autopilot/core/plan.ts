/**
 * Pure decisions the DOM drivers act on: given what a control shows and what the script wants, what has
 * to be pressed. Kept out of the drivers so it is testable without a DOM.
 */

const FA_DIGITS = '۰۱۲۳۴۵۶۷۸۹';
const AR_DIGITS = '٠١٢٣٤٥٦٧٨٩';

/** Reads a number a screen shows — Persian/Arabic digits, «٫» / «/» decimal marks, a unit after it. */
export function readNumber(text: string): number | null {
  let out = '';
  for (const ch of text) {
    const fa = FA_DIGITS.indexOf(ch);
    const ar = AR_DIGITS.indexOf(ch);
    if (fa >= 0) out += fa;
    else if (ar >= 0) out += ar;
    else if (ch === '٫' || ch === '/' || ch === ',') out += '.';
    else if (ch === '−') out += '-';
    else out += ch;
  }
  const match = out.match(/-?\d+(\.\d+)?/);
  if (!match) return null;
  const n = Number(match[0]);
  return Number.isFinite(n) ? n : null;
}

export interface ChoicePlan {
  /** Option codes to press, in order: the ones to switch off first, then the ones to switch on. */
  press: string[];
  /** Wanted codes the group doesn't offer. */
  unknown: string[];
}

/**
 * Presses that take a chip group from `selected` to `wanted`.
 *
 * `single` (the script gave one value): the group may keep at most that one — pressing the wanted chip is
 * enough when the group is exclusive, and pressing an already-on chip would clear it, so it is skipped.
 * Otherwise the result is the exact set.
 */
export function planChoice(available: readonly string[], selected: readonly string[], wanted: readonly string[], single: boolean): ChoicePlan {
  const offered = new Set(available);
  const unknown = wanted.filter((w) => !offered.has(w));
  const want = new Set(wanted.filter((w) => offered.has(w)));
  const on = new Set(selected);
  const turnOn = [...want].filter((w) => !on.has(w));
  // An exclusive group drops the old pick by itself when the new one is pressed.
  const turnOff = single && turnOn.length ? [] : [...on].filter((s) => !want.has(s));
  return { press: [...turnOff, ...turnOn], unknown };
}

/** Which way a stepper must move; `null` when it already shows the value (within half a step). */
export function stepperDirection(current: number, target: number, tolerance = 1e-6): 'up' | 'down' | null {
  if (Math.abs(current - target) <= tolerance) return null;
  return target > current ? 'up' : 'down';
}

/** A press went the right way if the reading got strictly closer (or the stepper only just adopted a base value). */
export function stepperProgressed(before: number | null, after: number | null, target: number): boolean {
  if (after === null) return false;
  if (before === null) return true;
  return Math.abs(after - target) < Math.abs(before - target);
}

/** Text an input should receive for a script value. */
export function textOf(value: string | number | boolean | null | readonly string[]): string {
  if (value === null) return '';
  if (Array.isArray(value)) return value.join(' ');
  return String(value);
}

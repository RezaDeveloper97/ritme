import { planChoice, readNumber, stepperDirection, stepperProgressed, textOf } from '../core/plan';
import { fail, OK, type ActContext, type ActResult, type PilotVerb } from '../core/runner';
import type { PilotValue } from '@/shared/pilot';
import { PILOT_KIND_ATTR, PILOT_OPTION_ATTR, type PilotKind } from '@/shared/pilot';
import { typingDelay, typingFrames } from '../core/timing';
import { nextFrame, optionsOf, optionValue, until } from './locate';

/**
 * One driver per kind of control. They operate the real elements with real DOM events, so the component's
 * own handlers run — no React state is written from outside, and a control that refuses a value (a clamped
 * number, a disabled chip) refuses the autopilot the same way it refuses a finger.
 */

/** The visible hand: the env implements it with the pointer, the ring and the ripple. */
export interface Hand {
  /** Scroll to the control, move the pointer onto it, press it (dispatches the click). */
  press(el: HTMLElement, ctx: ActContext): Promise<void>;
  /** Press again where the pointer already is (stepper repeats). */
  pressAgain(el: HTMLElement, ctx: ActContext): Promise<void>;
  /** Move onto a control without pressing (before typing). */
  reach(el: HTMLElement, ctx: ActContext): Promise<void>;
}

const FOCUS_ATTR = 'data-pilot-focus';
const MAX_STEPPER_PRESSES = 400;

export function kindOf(el: HTMLElement): PilotKind {
  const declared = el.getAttribute(PILOT_KIND_ATTR) as PilotKind | null;
  if (declared) return declared;
  if (el instanceof HTMLSelectElement) return 'select';
  if (optionsOf(el).length) return 'choice';
  if (el.matches('input[type="checkbox"], input[type="radio"]') || el.querySelector('input[type="checkbox"], input[type="radio"]')) return 'switch';
  const role = el.getAttribute('role');
  if (role === 'switch' || role === 'tab' || role === 'radio' || role === 'checkbox') return 'switch';
  if (el.hasAttribute('aria-pressed')) return 'toggle';
  if (el.matches('input, textarea')) return 'text';
  if (el.hasAttribute('aria-expanded')) return 'disclosure';
  return 'button';
}

const isDisabled = (el: HTMLElement): boolean =>
  (el as HTMLButtonElement).disabled === true || el.getAttribute('aria-disabled') === 'true';

/** Is this toggle-like control on? */
export function isOn(el: HTMLElement): boolean {
  const input = el.matches('input') ? (el as HTMLInputElement) : el.querySelector<HTMLInputElement>('input[type="checkbox"], input[type="radio"]');
  if (input) return input.checked;
  return ['aria-checked', 'aria-pressed', 'aria-selected'].some((name) => el.getAttribute(name) === 'true');
}

async function setOn(el: HTMLElement, want: boolean, ctx: ActContext, hand: Hand): Promise<ActResult> {
  if (isOn(el) === want) return OK;
  if (isDisabled(el)) return fail('disabled');
  await hand.press(el, ctx);
  return (await until(() => isOn(el) === want, ctx.config.timeouts.verifyMs)) ? OK : fail('not_verified');
}

async function setChoice(group: HTMLElement, value: PilotValue, ctx: ActContext, hand: Hand): Promise<ActResult> {
  if (typeof value === 'boolean') return fail('invalid_value');
  const single = !Array.isArray(value);
  const wanted = value === null ? [] : Array.isArray(value) ? value : [String(value)];
  const selected = () => optionsOf(group).filter(isOn).map(optionValue);
  const plan = planChoice(optionsOf(group).map(optionValue), selected(), wanted, single);
  if (plan.unknown.length) return fail('invalid_value');

  for (const code of plan.press) {
    // Re-query each time: pressing a chip can re-render the row.
    const option = optionsOf(group).find((o) => optionValue(o) === code);
    if (!option) return fail('target_not_found');
    if (isDisabled(option)) return fail('disabled');
    const before = isOn(option);
    await hand.press(option, ctx);
    await until(() => {
      const now = optionsOf(group).find((o) => optionValue(o) === code);
      return now !== undefined && isOn(now) !== before;
    }, ctx.config.timeouts.verifyMs);
  }

  const want = new Set(wanted);
  const now = selected();
  const matches = single ? wanted.every((w) => now.includes(w)) && (wanted.length > 0 || now.length === 0) : now.length === want.size && now.every((n) => want.has(n));
  return matches ? OK : fail('not_verified');
}

function field(el: HTMLElement): HTMLInputElement | HTMLTextAreaElement | null {
  if (el instanceof HTMLInputElement || el instanceof HTMLTextAreaElement) return el;
  return el.querySelector<HTMLInputElement | HTMLTextAreaElement>('input:not([type="checkbox"]):not([type="radio"]), textarea');
}

/** React reads the value through its own tracker, so the native setter + an `input` event is the only way in. */
function writeValue(input: HTMLInputElement | HTMLTextAreaElement, value: string): void {
  const proto = input instanceof HTMLTextAreaElement ? HTMLTextAreaElement.prototype : HTMLInputElement.prototype;
  Object.getOwnPropertyDescriptor(proto, 'value')?.set?.call(input, value);
  input.dispatchEvent(new Event('input', { bubbles: true }));
}

async function setText(el: HTMLElement, value: PilotValue, numeric: boolean, ctx: ActContext, hand: Hand): Promise<ActResult> {
  const input = field(el);
  if (!input) return fail('unsupported');
  if (input.disabled || input.readOnly) return fail('disabled');
  if (typeof value === 'boolean' || (numeric && Array.isArray(value))) return fail('invalid_value');

  let text = textOf(value);
  if (input.maxLength > 0) text = Array.from(text).slice(0, input.maxLength).join('');
  const holds = () => (numeric && typeof value === 'number' ? readNumber(input.value) === value : input.value === text);
  if (holds()) return OK;

  await hand.reach(input, ctx);
  const original = input.value;
  const real = ctx.config.typing.focus === 'real';
  // Fields that commit on blur (the log's number inputs) need the blur whether or not we focused.
  const commit = () => {
    if (real) input.blur();
    else input.dispatchEvent(new FocusEvent('focusout', { bubbles: true }));
  };
  if (real) input.focus({ preventScroll: true });
  else input.setAttribute(FOCUS_ATTR, '');
  let typed = false;
  try {
    if (input.value !== '') writeValue(input, '');
    const mode = ctx.config.still ? 'instant' : ctx.config.typing.mode;
    for (const frame of typingFrames(text, mode)) {
      writeValue(input, frame);
      await ctx.wait(typingDelay(frame, ctx.timing, ctx.rng));
    }
    typed = true;
  } finally {
    // Cancelled mid-word: half a value must never be committed — put back what the field held.
    if (!typed) writeValue(input, original);
    input.removeAttribute(FOCUS_ATTR);
    commit();
  }
  await nextFrame();
  if (await until(holds, ctx.config.timeouts.verifyMs)) return OK;
  // The field clamped or rejected it (a number out of range): that is the app saying no. Undo, so a value
  // nobody asked for can't be saved by a later step.
  writeValue(input, original);
  commit();
  return fail(numeric ? 'invalid_value' : 'not_verified');
}

async function setStepper(group: HTMLElement, value: PilotValue, ctx: ActContext, hand: Hand): Promise<ActResult> {
  if (typeof value !== 'number') return fail('invalid_value');
  const output = group.querySelector('output');
  const buttons = group.querySelectorAll<HTMLButtonElement>('button');
  if (!output || buttons.length < 2) return fail('unsupported');
  const read = () => readNumber(output.textContent ?? '');

  for (let presses = 0; presses < MAX_STEPPER_PRESSES; presses++) {
    const before = read();
    const direction = before === null ? 'up' : stepperDirection(before, value);
    if (direction === null) return OK;
    const button = direction === 'up' ? buttons[buttons.length - 1] : buttons[0];
    // Can't go further that way: the value is outside the stepper's range.
    if (button.disabled) return fail('invalid_value');
    if (presses === 0) await hand.press(button, ctx);
    else await hand.pressAgain(button, ctx);
    await until(() => read() !== before, ctx.config.timeouts.verifyMs);
    // Overshooting means the value sits between two steps.
    if (!stepperProgressed(before, read(), value)) return fail('invalid_value');
  }
  return fail('not_verified');
}

/** Scroll-snap pickers only listen to `scroll`: move the column, the component reads the new position itself. */
async function scrollTo(el: HTMLElement, top: number, ctx: ActContext, hand: Hand): Promise<ActResult> {
  await hand.reach(el, ctx);
  el.scrollTo({ top, behavior: ctx.config.still ? 'auto' : 'smooth' });
  await ctx.wait(ctx.timing.scrollSettleMs);
  return (await until(() => Math.abs(el.scrollTop - top) < 2, ctx.config.timeouts.verifyMs)) ? OK : fail('not_verified');
}

async function setWheel(el: HTMLElement, value: PilotValue, ctx: ActContext, hand: Hand): Promise<ActResult> {
  if (value === null || typeof value === 'boolean' || Array.isArray(value)) return fail('invalid_value');
  // Rows carry their value code (a year, a minute); the padding rows around them carry none.
  const rows = Array.from(el.querySelectorAll<HTMLElement>(`[${PILOT_OPTION_ATTR}]`));
  const index = rows.findIndex((row) => optionValue(row) === String(value));
  if (index < 0) return fail('invalid_value');
  return scrollTo(el, index * (rows[0].offsetHeight || 44), ctx, hand);
}

async function setRuler(el: HTMLElement, value: PilotValue, ctx: ActContext, hand: Hand): Promise<ActResult> {
  const min = Number(el.getAttribute('data-pilot-min'));
  const max = Number(el.getAttribute('data-pilot-max'));
  if (typeof value !== 'number' || !Number.isInteger(value) || !(value >= min && value <= max)) return fail('invalid_value');
  const tick = el.querySelector<HTMLElement>('.tk')?.offsetHeight || 12;
  // Ticks run from max (top) down to min.
  return scrollTo(el, (max - value) * tick, ctx, hand);
}

const MAX_MONTH_FLIPS = 240;

/** A month grid: flip months until the day is on screen, then press it. Days are keyed by ISO date. */
async function setCalendar(el: HTMLElement, value: PilotValue, ctx: ActContext, hand: Hand): Promise<ActResult> {
  if (typeof value !== 'string' || !/^\d{4}-\d{2}-\d{2}$/.test(value)) return fail('invalid_value');
  for (let flips = 0; flips <= MAX_MONTH_FLIPS; flips++) {
    const days = optionsOf(el);
    const day = days.find((d) => optionValue(d) === value);
    if (day) {
      if (isOn(day)) return OK;
      if (isDisabled(day)) return fail('invalid_value');
      await hand.press(day, ctx);
      return (await until(() => optionsOf(el).some((d) => optionValue(d) === value && isOn(d)), ctx.config.timeouts.verifyMs)) ? OK : fail('not_verified');
    }
    if (!days.length) return fail('unsupported');
    const first = optionValue(days[0]);
    const nav = el.querySelector<HTMLElement>(`[data-pilot-nav="${value < first ? 'prev' : 'next'}"]`);
    if (!nav || isDisabled(nav)) return fail('invalid_value');
    if (flips === 0) await hand.press(nav, ctx);
    else await hand.pressAgain(nav, ctx);
    if (!(await until(() => optionValue(optionsOf(el)[0] ?? el) !== first, ctx.config.timeouts.verifyMs))) return fail('not_verified');
  }
  return fail('invalid_value');
}

/** Native `<select>`: React hears `change`. */
async function setSelect(el: HTMLElement, value: PilotValue, ctx: ActContext, hand: Hand): Promise<ActResult> {
  const select = el instanceof HTMLSelectElement ? el : el.querySelector('select');
  if (!select) return fail('unsupported');
  if (select.disabled) return fail('disabled');
  const wanted = value === null ? '' : String(value);
  if (Array.isArray(value) || !Array.from(select.options).some((o) => o.value === wanted)) return fail('invalid_value');
  if (select.value === wanted) return OK;
  await hand.reach(select, ctx);
  Object.getOwnPropertyDescriptor(HTMLSelectElement.prototype, 'value')?.set?.call(select, wanted);
  select.dispatchEvent(new Event('change', { bubbles: true }));
  return (await until(() => select.value === wanted, ctx.config.timeouts.verifyMs)) ? OK : fail('not_verified');
}

async function disclose(el: HTMLElement, open: boolean, ctx: ActContext, hand: Hand): Promise<ActResult> {
  const toggle = el.hasAttribute('aria-expanded') ? el : el.querySelector<HTMLElement>('[aria-expanded]');
  if (!toggle) {
    // A button that opens a panel or an inline sheet has no open state to read: opening it is a press.
    // (Reveal steps only run while their target is missing, so this never closes what is already open.)
    if (!open) return fail('unsupported');
    if (isDisabled(el)) return fail('disabled');
    await hand.press(el, ctx);
    return OK;
  }
  const isOpen = () => toggle.getAttribute('aria-expanded') === 'true';
  if (isOpen() === open) return OK;
  if (isDisabled(toggle)) return fail('disabled');
  await hand.press(toggle, ctx);
  return (await until(() => isOpen() === open, ctx.config.timeouts.verifyMs)) ? OK : fail('not_verified');
}

export async function operate(verb: PilotVerb, el: HTMLElement, value: PilotValue, ctx: ActContext, hand: Hand): Promise<ActResult> {
  if (verb === 'expand' || verb === 'collapse') return disclose(el, verb === 'expand', ctx, hand);
  if (verb === 'tap') {
    if (isDisabled(el)) return fail('disabled');
    await hand.press(el, ctx);
    return OK;
  }
  switch (kindOf(el)) {
    case 'choice':
      return setChoice(el, value, ctx, hand);
    case 'toggle':
    case 'switch':
      return typeof value === 'boolean' || value === null ? setOn(el, value === true, ctx, hand) : fail('invalid_value');
    case 'text':
      return setText(el, value, false, ctx, hand);
    case 'number':
      return setText(el, value, true, ctx, hand);
    case 'stepper':
      return setStepper(el, value, ctx, hand);
    case 'wheel':
      return setWheel(el, value, ctx, hand);
    case 'ruler':
      return setRuler(el, value, ctx, hand);
    case 'calendar':
      return setCalendar(el, value, ctx, hand);
    case 'select':
      return setSelect(el, value, ctx, hand);
    case 'disclosure':
      return typeof value === 'boolean' ? disclose(el, value, ctx, hand) : fail('invalid_value');
    default:
      return fail('unsupported');
  }
}

/**
 * «هنوز نه» / «هنوز ادامه دارد» hide the hero prompt for the rest of the day.
 * A per-device convenience: the date is kept in localStorage (never the cycle
 * state itself, §11) and every access is guarded — private windows and blocked
 * storage simply show the prompt again.
 */
export type PromptKind = 'start' | 'end';

const key = (kind: PromptKind) => `ritme_home_prompt_${kind}`;

export function isPromptDismissed(kind: PromptKind, isoDay: string): boolean {
  try {
    return window.localStorage.getItem(key(kind)) === isoDay;
  } catch {
    return false;
  }
}

export function dismissPrompt(kind: PromptKind, isoDay: string): void {
  try {
    window.localStorage.setItem(key(kind), isoDay);
  } catch {
    // storage unavailable — the prompt just comes back on the next render
  }
}

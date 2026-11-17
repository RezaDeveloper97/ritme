import { PILOT_ATTR, PILOT_OPTION_ATTR, type TargetRef } from '@/shared/pilot';

/** Finding things on the page. Everything here reads the DOM; nothing changes it except the scroll helper. */

export function isVisible(el: Element): boolean {
  if (!(el instanceof HTMLElement)) return false;
  if (el.closest('[hidden], [aria-hidden="true"], [inert]')) return false;
  return el.getClientRects().length > 0;
}

const attr = (name: string, value: string): string => `[${name}="${CSS.escape(value)}"]`;

/** The LAST visible match wins: a sheet is painted after the screen it covers, so it is the one in front. */
function lastVisible(nodes: Iterable<Element>): HTMLElement | null {
  let found: HTMLElement | null = null;
  for (const node of nodes) if (isVisible(node)) found = node as HTMLElement;
  return found;
}

const IMPLICIT_ROLE: Record<string, string> = {
  button: 'button, [role="button"]',
  link: 'a[href], [role="link"]',
  tab: '[role="tab"]',
  switch: '[role="switch"]',
  checkbox: 'input[type="checkbox"], [role="checkbox"]',
  radio: 'input[type="radio"], [role="radio"]',
  textbox: 'input:not([type="checkbox"]):not([type="radio"]), textarea, [role="textbox"]',
};

function accessibleName(el: Element): string {
  const labelled = el.getAttribute('aria-labelledby');
  if (labelled) {
    const text = labelled
      .split(/\s+/)
      .map((id) => document.getElementById(id)?.textContent ?? '')
      .join(' ');
    if (text.trim()) return text.trim();
  }
  return (el.getAttribute('aria-label') ?? el.textContent ?? '').replace(/\s+/g, ' ').trim();
}

export function findTarget(ref: TargetRef, root: ParentNode = document): HTMLElement | null {
  if (typeof ref === 'string') return lastVisible(root.querySelectorAll(attr(PILOT_ATTR, ref)));
  if ('css' in ref) {
    try {
      return lastVisible(root.querySelectorAll(ref.css));
    } catch {
      return null;
    }
  }
  const selector = IMPLICIT_ROLE[ref.role] ?? `[role="${CSS.escape(ref.role)}"]`;
  const wanted = ref.name.replace(/\s+/g, ' ').trim();
  const matches: Element[] = [];
  for (const node of root.querySelectorAll(selector)) if (accessibleName(node) === wanted) matches.push(node);
  return lastVisible(matches);
}

/** The options of a `choice` group, in DOM order. */
export function optionsOf(group: HTMLElement): HTMLElement[] {
  return Array.from(group.querySelectorAll<HTMLElement>(`[${PILOT_OPTION_ATTR}]`)).filter(isVisible);
}

export const optionValue = (el: HTMLElement): string => el.getAttribute(PILOT_OPTION_ATTR) ?? '';

export const nextFrame = (): Promise<void> => new Promise((resolve) => requestAnimationFrame(() => resolve()));

/** Polls `check` once per frame until it is true or the time is up. */
export async function until(check: () => boolean, timeoutMs: number): Promise<boolean> {
  const deadline = performance.now() + timeoutMs;
  for (;;) {
    if (check()) return true;
    if (performance.now() >= deadline) return false;
    await nextFrame();
  }
}

/** Waits until the element has stopped moving (a sheet still sliding up, a section still opening). */
export async function settle(el: HTMLElement, timeoutMs = 900): Promise<void> {
  let last = el.getBoundingClientRect();
  let still = 0;
  await until(() => {
    const now = el.getBoundingClientRect();
    still = Math.abs(now.top - last.top) < 0.5 && Math.abs(now.left - last.left) < 0.5 ? still + 1 : 0;
    last = now;
    return still >= 3;
  }, timeoutMs);
}

/** Scrolls the control to the middle of its scroller unless it is already comfortably in view. */
export function scrollIntoComfort(el: HTMLElement, smooth: boolean): boolean {
  const rect = el.getBoundingClientRect();
  // Sticky headers and save bars are about this tall; a control behind one is not in view.
  const margin = 120;
  const inView = rect.top >= margin && rect.bottom <= window.innerHeight - margin;
  if (inView) return false;
  el.scrollIntoView({ block: 'center', inline: 'nearest', behavior: smooth ? 'smooth' : 'auto' });
  return true;
}

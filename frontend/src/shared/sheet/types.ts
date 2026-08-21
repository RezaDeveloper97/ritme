/**
 * How much of the app shell a sheet covers.
 *
 * - `half` — the **content** decides the height. The sheet starts at a
 *   comfortable minimum and grows downward-to-upward as its content grows. It
 *   must never scroll: a half sheet that scrolls is a `full` sheet wearing the
 *   wrong size, and reads as a bug to the user.
 * - `full` — a fixed tall panel that stops just short of the top edge. Content
 *   taller than the panel scrolls **inside** it.
 */
export type SheetSize = 'half' | 'full';

/**
 * The single prop every sheet-hosted screen receives. Sheets are addressed by
 * URL, so their input is one opaque string and nothing richer.
 */
export interface SheetContentProps {
  arg?: string;
}

/** Which sheet is showing, and the single argument it was opened with. */
export interface SheetTarget {
  /** Key into the app-layer sheet registry. */
  id: string;
  /**
   * One free-form argument — an info topic, an article slug. Deliberately a
   * single string: it rides in the URL, so it must stay short, opaque and free
   * of health data (CLAUDE.md §11).
   */
  arg?: string;
}

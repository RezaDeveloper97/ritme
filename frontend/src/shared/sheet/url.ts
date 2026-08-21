import type { SheetTarget } from './types';

/**
 * Sheets live in the query string rather than in a route segment. That buys
 * three things at once: the hardware/browser back button closes a sheet for
 * free, the screen underneath stays mounted (which is the whole point of a
 * sheet), and a sheet is still linkable.
 */
export const SHEET_PARAM = 'sheet';
export const SHEET_ARG_PARAM = 'sheetArg';

/** Reads the sheet a `location.search` string asks for, if any. */
export function readSheetTarget(search: string): SheetTarget | null {
  const params = new URLSearchParams(search);
  const id = params.get(SHEET_PARAM);
  if (!id) return null;
  const arg = params.get(SHEET_ARG_PARAM);
  return arg ? { id, arg } : { id };
}

/**
 * Rewrites `href` so it points at `target` (or at no sheet when it is `null`),
 * leaving every other query parameter untouched — screens keep their own state
 * in the URL (`?date=…`), and opening a sheet must not drop it.
 */
export function hrefWithSheet(href: string, target: SheetTarget | null): string {
  const url = new URL(href);
  const { searchParams } = url;

  if (target) {
    searchParams.set(SHEET_PARAM, target.id);
    if (target.arg) searchParams.set(SHEET_ARG_PARAM, target.arg);
    else searchParams.delete(SHEET_ARG_PARAM);
  } else {
    searchParams.delete(SHEET_PARAM);
    searchParams.delete(SHEET_ARG_PARAM);
  }

  return `${url.pathname}${url.search}${url.hash}`;
}

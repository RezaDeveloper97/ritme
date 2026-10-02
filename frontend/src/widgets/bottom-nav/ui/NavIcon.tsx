import type { NavIconName } from '../model/nav-items';

/**
 * Nav glyphs copied from the Night & Bloom artboards (24-grid, stroke 1.8,
 * drawn at 22px): Cycle_Home (home, calendar, grid, person), PregFull_Main
 * (drop), v19_Main (target), v16_ChildHome (child), nbl_Meno_Score (symptoms bars).
 */
const PATHS: Record<NavIconName, string> = {
  home: '<path d="M4 11l8-7 8 7v9a1 1 0 01-1 1h-5v-6h-4v6H5a1 1 0 01-1-1z"/>',
  calendar: '<rect x="4" y="5" width="16" height="15" rx="3"/><path d="M4 10h16M9 3v4M15 3v4"/>',
  fertility: '<circle cx="12" cy="12" r="8"/><circle cx="12" cy="12" r="3"/>',
  pregnancy: '<path d="M12 3c-4 0-7 6-7 10a7 7 0 0014 0c0-4-3-10-7-10z"/>',
  child: '<circle cx="12" cy="8" r="4"/><path d="M6 21c0-4 2.5-6 6-6s6 2 6 6"/>',
  symptoms: '<path d="M5 19V11M12 19V5M19 19v-6"/>',
  services:
    '<rect x="4" y="4" width="7" height="7" rx="2"/><rect x="13" y="4" width="7" height="7" rx="2"/>' +
    '<rect x="4" y="13" width="7" height="7" rx="2"/><rect x="13" y="13" width="7" height="7" rx="2"/>',
  me: '<circle cx="12" cy="8" r="4"/><path d="M5 21c1-4 4-6 7-6s6 2 7 6"/>',
};

export function NavIcon({ name }: { name: NavIconName }) {
  return (
    <svg
      className="nbnav-ic"
      width="22"
      height="22"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.8"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden
      focusable="false"
      dangerouslySetInnerHTML={{ __html: PATHS[name] }}
    />
  );
}

/**
 * Whether the log sheet hides its «ثبت با صدا» tab. Voice costs credits (the AI gate) and a teen account has no
 * commercial path (/plus redirects her), so the tab would be a dead end for her; everyone else keeps it and
 * the gate sheet explains a missing subscription / spent credits when she starts recording.
 */
export function voiceTabHidden(mode: string | null | undefined): boolean {
  return mode === 'teen';
}

/** Voice logging is paused for now: the tab shows «به‌زودی» and the sheet always stays on manual. */
export const VOICE_LOG_ENABLED = false;

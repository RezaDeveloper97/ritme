/**
 * Whether the log sheet hides its «ثبت با صدا» tab. Voice costs credits (the AI gate) and a teen account has no
 * commercial path (/plus redirects her), so the tab would be a dead end for her; everyone else keeps it and
 * the gate sheet explains a missing subscription / spent credits when she starts recording.
 */
export function voiceTabHidden(mode: string | null | undefined): boolean {
  return mode === 'teen';
}

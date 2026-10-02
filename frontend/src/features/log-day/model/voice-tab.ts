/**
 * Whether the log sheet hides its «ثبت با صدا» tab (B-N3-14b, N3 stage smoke B-9): voice logging is a
 * Plus feature and teens have no Plus path (/plus redirects them), so a locked tab would be a dead-end
 * upsell for a teen. Everyone else keeps the tab (locked ones show the Plus badge).
 */
export function voiceTabHidden(mode: string | null | undefined, voiceLocked: boolean): boolean {
  return mode === 'teen' && voiceLocked;
}

export type InviteShareOutcome = 'shared' | 'copied' | 'cancelled' | 'failed';

/**
 * «شریکم هنوز ریتمی ندارد · لینک دعوت برایش بفرست» (nbl_Onb_Partner): the app's
 * address in a short message through the Web Share sheet, else the clipboard.
 * Carries no code and nothing about the user — only the public app URL.
 */
export async function shareAppInvite(title: string, text: string): Promise<InviteShareOutcome> {
  if (typeof navigator !== 'undefined' && typeof navigator.share === 'function') {
    try {
      await navigator.share({ title, text });
      return 'shared';
    } catch (error) {
      if (error instanceof DOMException && error.name === 'AbortError') return 'cancelled';
    }
  }
  try {
    if (navigator.clipboard?.writeText) {
      await navigator.clipboard.writeText(text);
      return 'copied';
    }
  } catch {
    // denied or insecure origin
  }
  return 'failed';
}

/** The public address a partner installs Ritme from (this origin, default landing). */
export function appInviteUrl(): string {
  return typeof window === 'undefined' ? '' : window.location.origin;
}

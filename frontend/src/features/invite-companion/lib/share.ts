export type ShareOutcome = 'shared' | 'copied' | 'cancelled' | 'failed';

/** Copy text to the clipboard (async API, else the legacy selection fallback). */
export async function copyText(text: string): Promise<boolean> {
  try {
    if (navigator.clipboard?.writeText) {
      await navigator.clipboard.writeText(text);
      return true;
    }
  } catch {
    // fall through to the legacy path (insecure origin, denied permission)
  }
  try {
    const area = document.createElement('textarea');
    area.value = text;
    area.setAttribute('readonly', '');
    area.className = 'cmp-clip';
    document.body.appendChild(area);
    area.select();
    const ok = document.execCommand('copy');
    area.remove();
    return ok;
  } catch {
    return false;
  }
}

/**
 * Web Share API with a clipboard fallback. The invite code is the payload —
 * it goes only to the share target the user picks, never to logs or a URL.
 */
export async function shareText(title: string, text: string): Promise<ShareOutcome> {
  if (typeof navigator !== 'undefined' && typeof navigator.share === 'function') {
    try {
      await navigator.share({ title, text });
      return 'shared';
    } catch (error) {
      // A dismissed share sheet is not an error.
      if (error instanceof DOMException && error.name === 'AbortError') return 'cancelled';
    }
  }
  return (await copyText(text)) ? 'copied' : 'failed';
}

import { mimeEssence } from './store';

/**
 * Showing an on-device file to the user without letting it run as the app.
 *
 * A `blob:` URL document inherits the app's origin, so an HTML or SVG file
 * opened through one could read `localStorage` (the session token) — security
 * audit M3-M7 #2. Only types on the caller's `inlineTypes` allow-list (raster
 * images, PDF) are opened in a tab, re-wrapped with that exact type so the
 * browser can't sniff it into something else; anything else is handed over as
 * an `application/octet-stream` download, which the browser never renders.
 */

export type OpenMode = 'inline' | 'download';

/** How a file of `type` may be shown: inline only when it's on the allow-list. */
export function openModeFor(type: string, inlineTypes: readonly string[]): OpenMode {
  const t = mimeEssence(type);
  return t !== '' && inlineTypes.some((allowed) => mimeEssence(allowed) === t) ? 'inline' : 'download';
}

/** The blob to hand to `URL.createObjectURL` for that mode. */
export function blobForOpen(blob: Blob, mode: OpenMode): Blob {
  return new Blob([blob], { type: mode === 'inline' ? mimeEssence(blob.type) : 'application/octet-stream' });
}

/** Opens an allow-listed file in a new tab, or downloads anything else. Browser only. */
export function openLocalFile(
  file: { blob: Blob; name?: string },
  inlineTypes: readonly string[],
  fallbackName = 'attachment',
): OpenMode {
  const mode = openModeFor(file.blob.type, inlineTypes);
  const url = URL.createObjectURL(blobForOpen(file.blob, mode));
  if (mode === 'inline') {
    window.open(url, '_blank', 'noopener');
    window.setTimeout(() => URL.revokeObjectURL(url), 60_000);
  } else {
    const a = document.createElement('a');
    a.href = url;
    a.download = file.name || fallbackName;
    a.rel = 'noopener';
    document.body.appendChild(a);
    a.click();
    a.remove();
    window.setTimeout(() => URL.revokeObjectURL(url), 1_000);
  }
  return mode;
}

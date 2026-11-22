/**
 * Detector for untranslated i18n keys that leak onto a screen (`home.ring.title` instead of the copy).
 * next-intl prints the `namespace.key` path when a message is missing, so a dotted lower-case identifier in
 * the visible text is a bug. Domains, file names and numbers are not keys.
 */
const DOTTED = /(?<![\w./@-])[a-z][A-Za-z0-9]*(?:\.[A-Za-z][A-Za-z0-9_-]*)+(?![\w@:-])/g;
const NOT_KEY_TAIL = new Set([
  'app', 'ir', 'com', 'org', 'net', 'io', 'co', 'me', 'png', 'jpg', 'jpeg', 'svg', 'webp', 'pdf', 'json', 'js', 'css', 'html',
]);

export function findRawKeys(text: string): string[] {
  const found = new Set<string>();
  for (const match of text.matchAll(DOTTED)) {
    const tail = match[0].split('.').pop()!.toLowerCase();
    if (NOT_KEY_TAIL.has(tail)) continue;
    found.add(match[0]);
  }
  return [...found];
}

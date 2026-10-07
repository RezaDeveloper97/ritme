/** A same-app path from `?next=`, or `/` (no open redirects: `//host`, `/\host`, schemes). */
export function safeNext(raw: string | null | undefined): string {
  if (!raw || !raw.startsWith('/') || raw.startsWith('//') || raw.startsWith('/\\')) return '/';
  if (raw.startsWith('/login')) return '/';
  return raw;
}

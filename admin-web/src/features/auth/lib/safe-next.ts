/**
 * Where to go after login. Only same-origin absolute paths are allowed
 * (`/users?page=2`), never `//evil.example` or `https://…` (open redirect).
 */
export function safeNext(next: string | null | undefined): string {
  if (!next || !next.startsWith('/') || next.startsWith('//') || next.startsWith('/\\')) return '/';
  if (next === '/login' || next.startsWith('/login?') || next.startsWith('/login/')) return '/';
  return next;
}

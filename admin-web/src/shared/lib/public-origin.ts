/**
 * The origin the BROWSER used, for absolute redirects built on the server.
 *
 * Behind nginx the standalone Next server only knows its own bind address, so
 * `request.nextUrl.origin` in middleware is e.g. `https://localhost:3000` — a
 * redirect built from it sends the admin to localhost. nginx forwards the real
 * host (`Host` / `X-Forwarded-Host: $host`, only for its own server_names) and
 * scheme (`X-Forwarded-Proto`), so those win; anything malformed falls back.
 */
const HOST_RE = /^[A-Za-z0-9.-]+(:\d{1,5})?$/;

function first(value: string | null): string {
  return (value ?? '').split(',')[0]!.trim();
}

export function publicOrigin(headers: Headers, fallback: string): string {
  const host = first(headers.get('x-forwarded-host')) || first(headers.get('host'));
  if (!HOST_RE.test(host)) return fallback;
  const proto = first(headers.get('x-forwarded-proto')).toLowerCase();
  const scheme = proto === 'https' || proto === 'http' ? proto : new URL(fallback).protocol.replace(':', '');
  return `${scheme}://${host}`;
}

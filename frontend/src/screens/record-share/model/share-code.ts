import { MAX_SHARE_DOCUMENTS, type ReportSection } from '@/entities/health-record';

/**
 * The URL the QR encodes: the server's `qr_url` only when it is this web app's own origin (same protocol + host, no
 * credentials, no query / hash) and ends in the very `qr_path` it sent; else this origin + `qr_path`. https only,
 * unless the app itself runs on http (local dev). The QR may only ever point at the public report page.
 */
export function qrTarget(created: { qrUrl: string | null; qrPath: string }, origin: string): string | null {
  let app: URL;
  try {
    app = new URL(origin);
  } catch {
    return null;
  }
  if (app.protocol !== 'https:' && app.protocol !== 'http:') return null;
  if (!/^\/[a-z]{2,3}\/shared\/report\/[A-Za-z0-9_-]{43}$/.test(created.qrPath)) return null;
  if (created.qrUrl) {
    try {
      const u = new URL(created.qrUrl);
      const ok =
        u.protocol === app.protocol &&
        u.host === app.host &&
        !u.username &&
        !u.password &&
        u.pathname === created.qrPath &&
        !u.search &&
        !u.hash;
      if (ok) return u.toString();
    } catch {
      // fall through to the origin
    }
  }
  return `${app.origin}${created.qrPath}`;
}

/** Where a doctor types the code by hand: the public `/d` page of this web app. */
export function codeEntryUrl(origin: string, locale: string): string {
  return `${origin.replace(/\/+$/, '')}/${locale}/d`;
}

/** Toggles a section, keeping the canonical order. */
export function toggleSection(current: readonly ReportSection[], section: ReportSection): ReportSection[] {
  return current.includes(section) ? current.filter((s) => s !== section) : [...current, section];
}

/** Toggles a document id; never more than {@link MAX_SHARE_DOCUMENTS}. */
export function toggleDocument(current: readonly number[], id: number): number[] {
  if (current.includes(id)) return current.filter((d) => d !== id);
  if (current.length >= MAX_SHARE_DOCUMENTS) return [...current];
  return [...current, id];
}

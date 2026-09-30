import { z } from 'zod';

/**
 * Support screen data (B-N1-12). FAQ = the admin `help` boxes; contact = the
 * admin `support` boxes, read by link scheme: a `tel:` box is the phone line
 * (label = number, body = hours), the first `https:`/`mailto:` box is where the
 * «گفت‌وگو با پشتیبانی» card goes (its body is the card's caption).
 */

export interface SupportBox {
  id: number;
  heading: string;
  body: string;
  linkLabel: string | null;
  linkUrl: string | null;
}

const SAFE_URL = /^(https?:|mailto:|tel:)/i;

export const boxesSchema = z
  .object({
    sections: z.array(
      z.object({
        id: z.number(),
        heading: z.string(),
        body: z.string(),
        link_label: z.string().nullable(),
        link_url: z.string().nullable(),
      }),
    ),
  })
  .transform((v): SupportBox[] =>
    v.sections.map((s) => {
      const ok = Boolean(s.link_label && s.link_url && SAFE_URL.test(s.link_url.trim()));
      return {
        id: s.id,
        heading: s.heading,
        body: s.body,
        linkLabel: ok ? s.link_label : null,
        linkUrl: ok ? (s.link_url as string).trim() : null,
      };
    }),
  );

export interface SupportContact {
  phone: SupportBox | null;
  chat: SupportBox | null;
}

export function contactOf(boxes: readonly SupportBox[]): SupportContact {
  return {
    phone: boxes.find((b) => b.linkUrl?.toLowerCase().startsWith('tel:')) ?? null,
    chat: boxes.find((b) => b.linkUrl && /^(https:|mailto:)/i.test(b.linkUrl)) ?? null,
  };
}

/** Case- and diacritic-insensitive FAQ search over question and answer. */
export function filterFaq(boxes: readonly SupportBox[], query: string): SupportBox[] {
  const norm = (s: string) =>
    s
      .toLocaleLowerCase()
      .replace(/[ً-ٰٟ‌]/g, '')
      .replace(/ي/g, 'ی')
      .replace(/ك/g, 'ک')
      .trim();
  const q = norm(query);
  if (!q) return [...boxes];
  return boxes.filter((b) => norm(`${b.heading} ${b.body}`).includes(q));
}

export const REPORT_MIN = 10;
export const REPORT_MAX = 2000;
export const SCREENSHOT_MAX_BYTES = 5 * 1024 * 1024;

export function isReportValid(message: string): boolean {
  const n = message.trim().length;
  return n >= REPORT_MIN && n <= REPORT_MAX;
}

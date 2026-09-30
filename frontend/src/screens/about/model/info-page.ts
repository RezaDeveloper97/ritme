import { z } from 'zod';

/**
 * GET /info-pages/{group} (B-N1-12): the admin-managed boxes of one text screen
 * with their stable `key` and the page's last edit day. Keys this screen reads:
 * `summary` (the Legal «خلاصه در ۳ خط», one line per `\n`) and `disclaimer`
 * (About footer). Everything else is ordinary content in admin order.
 */

export type InfoPageGroup = 'about' | 'privacy' | 'terms';

export interface InfoPageSection {
  id: number;
  key: string | null;
  heading: string;
  body: string;
  linkLabel: string | null;
  linkUrl: string | null;
}

export interface InfoPage {
  /** `Y-m-d` of the latest edit, or null for an empty page. */
  updatedAt: string | null;
  sections: InfoPageSection[];
}

export const infoPageSchema = z
  .object({
    updated_at: z.string().nullable(),
    sections: z.array(
      z.object({
        id: z.number(),
        key: z.string().nullable(),
        heading: z.string(),
        body: z.string(),
        link_label: z.string().nullable(),
        link_url: z.string().nullable(),
      }),
    ),
  })
  .transform(
    (v): InfoPage => ({
      updatedAt: v.updated_at,
      sections: v.sections.map((s) => ({
        id: s.id,
        key: s.key,
        heading: s.heading,
        body: s.body,
        linkLabel: s.link_label && s.link_url ? s.link_label : null,
        linkUrl: s.link_label && s.link_url && isSafeUrl(s.link_url) ? s.link_url : null,
      })),
    }),
  );

/** Admin links may be web, mail or phone links — never `javascript:` or `data:`. */
export function isSafeUrl(url: string): boolean {
  return /^(https?:|mailto:|tel:)/i.test(url.trim());
}

/** The Legal screen: summary lines, and the numbered body sections. */
export function splitLegal(page: InfoPage): { summary: string[]; sections: InfoPageSection[] } {
  const summary = page.sections.find((s) => s.key === 'summary');
  return {
    summary: summary ? summary.body.split('\n').map((l) => l.trim()).filter(Boolean) : [],
    sections: page.sections.filter((s) => s.key !== 'summary'),
  };
}

/**
 * The About screen: the intro (first plain box), the rows (other plain boxes),
 * the link chips (boxes with a link — social / web) and the disclaimer box.
 */
export function splitAbout(page: InfoPage): {
  intro: InfoPageSection | null;
  rows: InfoPageSection[];
  links: InfoPageSection[];
  disclaimer: InfoPageSection | null;
} {
  const disclaimer = page.sections.find((s) => s.key === 'disclaimer') ?? null;
  const rest = page.sections.filter((s) => s !== disclaimer);
  const plain = rest.filter((s) => !s.linkUrl);
  return {
    intro: plain[0] ?? null,
    rows: plain.slice(1),
    links: rest.filter((s) => s.linkUrl),
    disclaimer,
  };
}

import { z } from 'zod';

/**
 * GET /info-pages/insurance (B-N10-06): admin boxes of the `insurance` group. Keys read here:
 * `coverage` (text), `claim_steps` (one step per line), `documents` (one item per line),
 * `disclaimer` (small print); every other box is a FAQ entry (heading = question). A box
 * link, when the admin set one, is the insurer link-out. No personal data is read or sent.
 */
export interface InsuranceBox {
  id: number;
  key: string | null;
  heading: string;
  body: string;
  linkLabel: string | null;
  linkUrl: string | null;
}

export interface InsurancePageData {
  updatedAt: string | null;
  sections: InsuranceBox[];
}

/** Web links only — never `javascript:`, `data:`, `mailto:` etc. for an insurer link-out. */
export function isWebUrl(url: string): boolean {
  return /^https?:\/\//i.test(url.trim());
}

export const insurancePageSchema = z
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
    (v): InsurancePageData => ({
      updatedAt: v.updated_at,
      sections: v.sections.map((s) => {
        const ok = Boolean(s.link_label && s.link_url && isWebUrl(s.link_url));
        return {
          id: s.id,
          key: s.key,
          heading: s.heading,
          body: s.body,
          linkLabel: ok ? s.link_label : null,
          linkUrl: ok ? s.link_url : null,
        };
      }),
    }),
  );

const lines = (text: string): string[] => text.split('\n').map((l) => l.trim()).filter(Boolean);

export function splitInsurance(page: InsurancePageData) {
  const by = (k: string) => page.sections.find((s) => s.key === k) ?? null;
  const coverage = by('coverage');
  const steps = by('claim_steps');
  const docs = by('documents');
  const disclaimer = by('disclaimer');
  const known = new Set([coverage, steps, docs, disclaimer]);
  return {
    coverage,
    steps: steps ? lines(steps.body) : [],
    stepsHeading: steps?.heading ?? null,
    documents: docs ? lines(docs.body) : [],
    documentsHeading: docs?.heading ?? null,
    disclaimer,
    faq: page.sections.filter((s) => !known.has(s)),
  };
}

import { z } from 'zod';

import type { InfoSection } from '../model/types';

/**
 * Boundary parser for GET /info/{group} (§10 — validate external data with
 * zod). The API hands back `{ group, sections: [...] }` already localized, and
 * pairs `link_label`/`link_url` so a half-filled link never reaches the UI.
 */
export const infoSectionsSchema = z
  .object({
    sections: z
      .array(
        z.object({
          id: z.number(),
          heading: z.string().default(''),
          body: z.string().default(''),
          link_label: z.string().nullish(),
          link_url: z.string().nullish(),
        }),
      )
      .default([]),
  })
  .transform((v): InfoSection[] =>
    v.sections.map((s) => ({
      id: s.id,
      heading: s.heading,
      body: s.body,
      // Belt and braces: the API already drops half-filled links, but a button
      // with no caption or no destination is worse than no button.
      linkLabel: s.link_label && s.link_url ? s.link_label : null,
      linkUrl: s.link_label && s.link_url ? s.link_url : null,
    })),
  );

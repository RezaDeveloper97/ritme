import { z } from 'zod';

import type { Banner, BannerPosition, BannersByPosition } from '../model/types';

const POSITIONS: readonly BannerPosition[] = [
  'home_top',
  'home_middle',
  'home_bottom',
];

/**
 * Validate a banner at the API boundary (CLAUDE.md §10) and map its snake_case
 * fields onto our camelCase domain shape. Unknown positions/link types are
 * tolerated as-is; the widget simply won't have a slot for an unknown position.
 */
const bannerSchema = z
  .object({
    id: z.number(),
    title: z.string().nullable().default(null),
    image_url: z.string(),
    position: z.string(),
    link_url: z.string().nullable().default(null),
    link_type: z.enum(['internal', 'external']).nullable().default(null),
  })
  .transform(
    (b): Banner => ({
      id: b.id,
      title: b.title,
      imageUrl: b.image_url,
      position: b.position as BannerPosition,
      linkUrl: b.link_url,
      linkType: b.link_type,
    }),
  );

/**
 * `{ positions: { home_top: [...], home_middle: [...], home_bottom: [...] } }`.
 * Every known slot is guaranteed present (empty array when nothing is active),
 * so consumers can index a position without null checks.
 */
export const bannersEnvelopeSchema = z
  .object({
    positions: z.record(z.string(), z.array(bannerSchema)).default({}),
  })
  .transform((e): BannersByPosition => {
    const out = {} as BannersByPosition;
    for (const position of POSITIONS) {
      out[position] = e.positions[position] ?? [];
    }
    return out;
  });

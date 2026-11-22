import { z } from 'zod';

import type { Ad, AdSlot, Banner, BannerPosition, BannersByPosition } from '../model/types';

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
    positions: z.record(z.string(), z.array(bannerSchema)).prefault({}),
  })
  .transform((e): BannersByPosition => {
    const out = {} as BannersByPosition;
    for (const position of POSITIONS) {
      out[position] = e.positions[position] ?? [];
    }
    return out;
  });

const AD_SLOTS: readonly AdSlot[] = ['ad_slider', 'ad_square', 'ad_strip'];

const adSchema = z
  .object({
    id: z.number(),
    title: z.string().nullable().default(null),
    image_url: z.string().nullable().default(null),
    position: z.string(),
    cta_label: z.string().nullable().default(null),
    link_url: z.string().nullable().default(null),
    link_type: z.enum(['internal', 'external']).nullable().default(null),
  })
  .transform(
    (a): Ad => ({
      id: a.id,
      title: a.title,
      imageUrl: a.image_url ?? '',
      slot: a.position as AdSlot,
      ctaLabel: a.cta_label,
      linkUrl: a.link_url,
      linkType: a.link_type,
    }),
  );

/** `{ positions: { ad_slider: [...], ad_square: [...], ad_strip: [...] } }` — every slot present. */
export const adsEnvelopeSchema = z
  .object({ positions: z.record(z.string(), z.array(adSchema)).prefault({}) })
  .transform((e): Record<AdSlot, Ad[]> => {
    const out = {} as Record<AdSlot, Ad[]>;
    for (const slot of AD_SLOTS) out[slot] = e.positions[slot]?.filter((a) => a.imageUrl) ?? [];
    return out;
  });

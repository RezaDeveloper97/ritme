/**
 * Pain locations the figure can place (the `pain.location` items of the log taxonomy, nbl_Log_Pain).
 * Positions live in globals.css (`.bmap-pin.is-<code>`), so this is only the list; a location the figure
 * doesn't know yet (a newer taxonomy) is offered as a chip under the map instead of being dropped.
 */
export const MAPPED_REGIONS = ['head', 'breast', 'abdomen', 'back', 'ovary', 'pelvis', 'leg', 'stitches', 'joints'] as const;

export type MappedRegion = (typeof MAPPED_REGIONS)[number];

export function isMappedRegion(code: string): code is MappedRegion {
  return (MAPPED_REGIONS as readonly string[]).includes(code);
}

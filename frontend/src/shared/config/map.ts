/**
 * Neshan map configuration (CB-CORE-06, roadmap/DECISIONS.md #16).
 *
 * The map is optional by design: with no key the map wrapper renders nothing
 * and callers show their list instead. Pure — no env access here, so the
 * resolution rule is unit-testable; `env.ts` feeds it the raw value.
 */

/** A blank / whitespace-only key is "no key", never a broken map. */
export function resolveNeshanKey(raw: string | undefined | null): string | null {
  const key = raw?.trim();
  return key ? key : null;
}

/**
 * Pinned Neshan web SDK (mapbox-gl 1.13.2 + neshan-sdk 1.1.5). Bumping it means
 * re-checking the CSP hosts listed in docs/canvas-build/map.md.
 */
export const NESHAN_SDK = {
  scriptUrl: 'https://static.neshan.org/sdk/mapboxgl/v1.13.2/neshan-sdk/v1.1.5/index.js',
  styleUrl: 'https://static.neshan.org/sdk/mapboxgl/v1.13.2/neshan-sdk/v1.1.5/index.css',
} as const;

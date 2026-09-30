import { env } from '@/shared/config';

/**
 * The flag callers branch on: `false` → skip the map and render the list
 * (DECISIONS #16). Static per build — the key is inlined at build time.
 */
export function isMapEnabled(): boolean {
  return env.neshanKey !== null;
}

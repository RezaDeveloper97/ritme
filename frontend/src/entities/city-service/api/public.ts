import { env } from '@/shared/config';

import { type CityServiceDetail, listingOneSchema } from './schema';

/**
 * CB-DIR-11: GET /city-services/listings/:id from a server component (no session, no cookies, no user data).
 * `null` for an unknown/inactive place or when the API is unreachable — the caller falls back to client loading.
 */
export async function fetchPublicListing(id: number): Promise<CityServiceDetail | null> {
  try {
    const url = `${env.apiBaseUrl.replace(/\/+$/, '')}/city-services/listings/${id}`;
    const res = await fetch(url, { headers: { Accept: 'application/json' }, next: { revalidate: 300 } });
    if (!res.ok) return null;
    const body = (await res.json()) as { data?: unknown };
    const parsed = listingOneSchema.safeParse(body.data);
    return parsed.success ? parsed.data.listing : null;
  } catch {
    return null;
  }
}

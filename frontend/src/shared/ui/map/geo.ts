import type { MapBbox, MapPoint, MapTheme } from './types';

/** Tehran — the default view before a caller or "my location" says otherwise. */
export const DEFAULT_CENTER: MapPoint = { lat: 35.6892, lng: 51.389 };
export const DEFAULT_ZOOM = 12;
/** Zoom used when centring on the user's own location. */
export const LOCATE_ZOOM = 14;

/** mapbox-gl takes `[lng, lat]`; keep the swap in exactly one place. */
export function toLngLat(p: MapPoint): [number, number] {
  return [p.lng, p.lat];
}

interface BoundsLike {
  getSouth(): number;
  getWest(): number;
  getNorth(): number;
  getEast(): number;
}

/** Normalise mapbox bounds to a plain bbox, rounded to ~1 m (6 dp). */
export function boundsToBbox(b: BoundsLike): MapBbox {
  const r = (n: number): number => Math.round(n * 1e6) / 1e6;
  return {
    south: r(b.getSouth()),
    west: r(b.getWest()),
    north: r(b.getNorth()),
    east: r(b.getEast()),
  };
}

/** Neshan's own day / night vector styles — the closest fit to N&B light/dark. */
export function mapTypeFor(theme: MapTheme): 'neshanVector' | 'neshanVectorNight' {
  return theme === 'dark' ? 'neshanVectorNight' : 'neshanVector';
}

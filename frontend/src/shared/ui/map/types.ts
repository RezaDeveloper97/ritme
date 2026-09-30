/** A WGS84 point. `lat`/`lng` names (not a tuple) so order can't be swapped. */
export interface MapPoint {
  lat: number;
  lng: number;
}

/** The visible area, for "search this area" queries. */
export interface MapBbox {
  south: number;
  west: number;
  north: number;
  east: number;
}

export interface MapPin extends MapPoint {
  /** Stable id — selection and marker reuse are keyed by it. */
  id: string;
  /** Accessible name (and the default pin's label when selected). */
  label: string;
  /** A server-side cluster of N places; the default pin shows the number. */
  count?: number;
}

/** Why the map is not on screen — the caller shows its list in every case. */
export type MapUnavailableReason = 'no-key' | 'load-failed';

export type MapTheme = 'light' | 'dark';

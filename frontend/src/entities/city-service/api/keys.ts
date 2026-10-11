/** Query keys of the city-services directory (CLAUDE.md §8). */
/** CB-DIR-06: search params of the directory list (sent as the CB-DIR-02 query string). */
export interface DirectoryParams {
  city?: number | null;
  category?: string | null;
  q?: string;
  ageMonths?: number | null;
  sort?: 'rank' | 'distance' | 'rating';
  /** CB-DIR-07 filters. `lat`/`lng` come from a user-tapped «near me» and live in memory only. */
  openNow?: boolean;
  amenities?: string[];
  lat?: number;
  lng?: number;
  radiusKm?: number;
  /** `min_lng,min_lat,max_lng,max_lat` of the map's visible area. */
  bbox?: string;
  perPage?: number;
  page?: number;
}

export const cityServiceKeys = {
  all: ['city-service'] as const,
  options: () => [...cityServiceKeys.all, 'options'] as const,
  listings: (city: number | null, category: string | null, page: number) =>
    [...cityServiceKeys.all, 'listings', city, category, page] as const,
  directory: (params: DirectoryParams) => [...cityServiceKeys.all, 'directory', params] as const,
  listing: (id: number) => [...cityServiceKeys.all, 'listing', id] as const,
  slots: (id: number, serviceId: number | null) => [...cityServiceKeys.all, 'slots', id, serviceId] as const,
  bookings: () => [...cityServiceKeys.all, 'bookings'] as const,
  bookingList: (scope: 'upcoming' | 'past') => [...cityServiceKeys.all, 'bookings', 'list', scope] as const,
  booking: (id: number) => [...cityServiceKeys.all, 'bookings', 'one', id] as const,
};

/** Query keys of the city-services directory (CLAUDE.md §8). */
export const cityServiceKeys = {
  all: ['city-service'] as const,
  options: () => [...cityServiceKeys.all, 'options'] as const,
  listings: (city: number | null, category: string | null, page: number) =>
    [...cityServiceKeys.all, 'listings', city, category, page] as const,
  listing: (id: number) => [...cityServiceKeys.all, 'listing', id] as const,
  bookings: () => [...cityServiceKeys.all, 'bookings'] as const,
};

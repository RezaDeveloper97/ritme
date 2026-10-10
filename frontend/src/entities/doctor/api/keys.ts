/** Query keys of the doctors directory and bookings (CLAUDE.md §8). */
export const doctorKeys = {
  all: ['doctors'] as const,
  filters: () => [...doctorKeys.all, 'filters'] as const,
  list: (params: object) => [...doctorKeys.all, 'list', params] as const,
  profile: (id: number) => [...doctorKeys.all, 'profile', id] as const,
  slots: (id: number, mode: string, from: string) => [...doctorKeys.all, 'slots', id, mode, from] as const,
  reviews: (id: number) => [...doctorKeys.all, 'reviews', id] as const,
};

export const bookingKeys = {
  all: ['telemed-bookings'] as const,
  list: (scope: string) => [...bookingKeys.all, 'list', scope] as const,
  one: (id: number) => [...bookingKeys.all, 'one', id] as const,
  quote: (doctorId: number, mode: string) => [...bookingKeys.all, 'quote', doctorId, mode] as const,
  messages: (id: number) => [...bookingKeys.all, 'messages', id] as const,
  prescription: (id: number) => [...bookingKeys.all, 'prescription', id] as const,
  video: (id: number) => [...bookingKeys.all, 'video', id] as const,
};

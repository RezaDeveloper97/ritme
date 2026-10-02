/** Query keys of the teen reads (CB-TEEN-02). */
export const teenKeys = {
  all: ['teen'] as const,
  profile: () => [...teenKeys.all, 'profile'] as const,
  today: () => [...teenKeys.all, 'today'] as const,
};

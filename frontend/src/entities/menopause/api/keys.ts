/** Query keys of the menopause reads (CB-MENO-05). */
export const menopauseKeys = {
  all: ['menopause'] as const,
  profile: () => [...menopauseKeys.all, 'profile'] as const,
  today: () => [...menopauseKeys.all, 'today'] as const,
  messages: () => [...menopauseKeys.all, 'messages'] as const,
};

/** Query keys of the assistant (CLAUDE.md §8). */
export const assistantKeys = {
  all: ['assistant'] as const,
  hub: () => [...assistantKeys.all, 'hub'] as const,
  department: (code: string) => [...assistantKeys.all, 'department', code] as const,
  profile: () => [...assistantKeys.all, 'profile'] as const,
  conversations: () => [...assistantKeys.all, 'conversations'] as const,
  conversation: (id: number) => [...assistantKeys.all, 'conversation', id] as const,
  consent: (code: string) => ['consents', 'assistant', code] as const,
};

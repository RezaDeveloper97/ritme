/** Query keys of the menopause reads (CB-MENO-05). */
export const menopauseKeys = {
  all: ['menopause'] as const,
  profile: () => [...menopauseKeys.all, 'profile'] as const,
  today: () => [...menopauseKeys.all, 'today'] as const,
  messages: () => [...menopauseKeys.all, 'messages'] as const,
  // CB-MENO-07 hot-flash timer (`date` null = today) + the `meno_tips` catalog.
  hotFlashes: () => [...menopauseKeys.all, 'hot-flashes'] as const,
  hotFlashDay: (date: string | null) => [...menopauseKeys.hotFlashes(), date ?? 'today'] as const,
  tips: (locale: string) => [...menopauseKeys.all, 'tips', locale] as const,
  // CB-MENO-08 monthly score + patterns.
  scores: (months: number) => [...menopauseKeys.all, 'scores', months] as const,
  scoreQuestions: () => [...menopauseKeys.all, 'score-questions'] as const,
  patterns: () => [...menopauseKeys.all, 'patterns'] as const,
  // CB-MENO-10 treatment & care (`date` null = this week).
  treatments: () => [...menopauseKeys.all, 'treatment'] as const,
  treatment: (date: string | null) => [...menopauseKeys.treatments(), date ?? 'today'] as const,
};

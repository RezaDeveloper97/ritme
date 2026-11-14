import type { CompanionArticle } from '@/entities/companion';
import type { IconName } from '@/shared/ui';

/*
 * «یادگیری» (companion v2, compLearn): four short, non-medical topics. Each opens a matching article from
 * `GET /companion/home` when one exists, else a half sheet with the static copy under
 * `companionHome.learn.topics.<id>`.
 */

export type LearnTopicId = 'cycle' | 'support' | 'space' | 'empathy';

export interface LearnTopic {
  id: LearnTopicId;
  icon: IconName;
  /** Tint accent of the tile (`cmh2-tile is-<accent>`). */
  accent: 'brand' | 'bloom' | 'data' | 'warm';
  /** Slug fragments that mark an article as being about this topic. */
  keywords: readonly string[];
}

export const LEARN_TOPICS: readonly LearnTopic[] = [
  { id: 'cycle', icon: 'book', accent: 'brand', keywords: ['cycle', 'period', 'menstru', 'phase'] },
  { id: 'support', icon: 'heart', accent: 'bloom', keywords: ['support', 'partner', 'companion', 'judg'] },
  { id: 'space', icon: 'smile', accent: 'data', keywords: ['space', 'ask', 'listen'] },
  { id: 'empathy', icon: 'chat', accent: 'warm', keywords: ['empathy', 'empath', 'solution'] },
];

export function learnTopic(id: LearnTopicId): LearnTopic {
  return LEARN_TOPICS.find((t) => t.id === id)!;
}

/** The first companion article whose slug mentions one of the topic's keywords, or null. */
export function matchArticle(topic: LearnTopic, articles: readonly CompanionArticle[]): CompanionArticle | null {
  return articles.find((a) => topic.keywords.some((k) => a.slug.toLowerCase().includes(k))) ?? null;
}

/**
 * Index of today's tip in a list of `count`, from an ISO day (`YYYY-MM-DD`): days since the epoch, so the
 * tip turns over once a day and walks the whole list.
 */
export function dailyIndex(isoDay: string, count: number): number {
  if (count <= 0) return 0;
  const [y, m, d] = isoDay.split('-').map(Number);
  if (!y || !m || !d) return 0;
  const days = Math.floor(Date.UTC(y, m - 1, d) / 86_400_000);
  return ((days % count) + count) % count;
}

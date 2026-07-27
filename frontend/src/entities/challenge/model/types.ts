/** How demanding a challenge is; drives the badge on the card. */
export type ChallengeDifficulty = 'easy' | 'medium' | 'hard';

/** One day in the 7-day strip under the challenge card. */
export interface ChallengeDay {
  /** Gregorian `YYYY-MM-DD` — formatted for display by the date layer (§7). */
  date: string;
  isCompleted: boolean;
  isToday: boolean;
}

/**
 * Today's challenge as chosen by the backend for this user (cycle phase +
 * recent-log signal + streak-unlocked difficulty), together with the user's
 * progress. The app never picks the challenge itself.
 */
export interface TodayChallenge {
  id: number;
  /** Already localized by the API. */
  title: string;
  description: string | null;
  category: string | null;
  difficulty: ChallengeDifficulty | null;
  isCompleted: boolean;
  /** Consecutive days with at least one completed challenge. */
  streak: number;
  longestStreak: number;
  /** Last 7 days, oldest first. */
  weekDays: ChallengeDay[];
  /** Encouraging line under the card; null when there is nothing to say. */
  statusMessage: string | null;
}

/** What `POST /home/challenges/{id}/toggle` reports back. */
export interface ChallengeToggleResult {
  challengeId: number;
  isCompleted: boolean;
  streak: number;
  longestStreak: number;
  weekDays: ChallengeDay[];
  statusMessage: string | null;
}

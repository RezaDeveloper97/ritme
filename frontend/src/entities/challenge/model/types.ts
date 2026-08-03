/**
 * The stretch of the cycle a challenge was authored for, in cycle days. Both
 * bounds are independent: `{from: null, to: null}` means "any day", and a
 * single open bound means "from day N on" / "up to day N".
 */
export interface ChallengeCycleDayRange {
  from: number | null;
  to: number | null;
}

/**
 * Today's challenge: one task an admin authored, which the user can tick off.
 * The backend picks it (by cycle day and recent logs) and the app only renders
 * it — there is deliberately no streak, score or progression attached.
 */
export interface TodayChallenge {
  id: number;
  /** Already localized by the API. */
  title: string;
  description: string | null;
  category: string | null;
  /**
   * The user's day of cycle the pick was made for, or null when her cycle
   * isn't known yet (no period logged / pregnancy mode).
   */
  cycleDay: number | null;
  /** The day range this challenge is targeted at. */
  cycleDayRange: ChallengeCycleDayRange;
  isCompleted: boolean;
}

/** What `POST /home/challenges/{id}/toggle` reports back. */
export interface ChallengeToggleResult {
  challengeId: number;
  isCompleted: boolean;
}

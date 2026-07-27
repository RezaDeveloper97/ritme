/** Which experience the user is currently in (CLAUDE.md §1). */
export type AppMode = 'cycle' | 'pregnancy';

/** The user's current mode plus the flags that shape the daily message. */
export interface UserMode {
  mode: AppMode;
  modeLabel: string;
  /** Trying-to-conceive vs. not; drives message tone. Informational (§11). */
  userGoal: string;
  isTtc: boolean;
  isPremium: boolean;
}

/** The headline personalized message for the day. */
export interface PrimaryMessage {
  shortMessage: string;
  longMessage: string;
  actionSuggestion: string;
  /** Suggested things to do today (already localized by the backend). */
  dos: string[];
  /** Things to avoid today (already localized by the backend). */
  donts: string[];
}

/**
 * A symptom-to-symptom insight the correlation engine detected for today
 * (message layer 3). Text is already localized by the backend.
 */
export interface MessageCorrelation {
  /** Stable engine id (`sleep_mood`, `pms_symptoms`, …) — safe as a React key. */
  type: string;
  insightMessage: string;
  action: string;
  isPremiumOnly: boolean;
}

/** A multi-day trend the pattern engine found (message layer 4, premium). */
export interface MessagePattern {
  patternType: string;
  /** `info` | `warning` | `alert` — drives emphasis, not content. */
  alertLevel: string;
  message: string;
  recommendation: string;
}

/** Nutrition / sleep / exercise guidance for the day, flattened to what UI shows. */
export interface SupplementTips {
  nutritionFocus: string;
  nutritionTip: string;
  sleepFocus: string;
  sleepTips: string[];
  exerciseTip: string;
}

/** A daily personalized message bundle for the current mode/phase. */
export interface DailyMessage {
  mode: AppMode;
  /** Backend date the message is for (ISO/Gregorian; format via §7 for display). */
  date: string;
  phase: string | null;
  phaseLabel: string | null;
  cycleDay: number | null;
  primary: PrimaryMessage;
  /** Layer 3 insights; empty when the day has no logged symptoms. */
  correlations: MessageCorrelation[];
  /** Layer 4 trends; empty for free accounts or with too little history. */
  patterns: MessagePattern[];
  supplements: SupplementTips;
  /** Free-form extra tips the engine attached to the day. */
  tips: string[];
}

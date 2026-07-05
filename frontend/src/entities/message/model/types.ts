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

/** A daily personalized message bundle for the current mode/phase. */
export interface DailyMessage {
  mode: AppMode;
  /** Backend date the message is for (ISO/Gregorian; format via §7 for display). */
  date: string;
  phase: string | null;
  phaseLabel: string | null;
  cycleDay: number | null;
  primary: PrimaryMessage;
}

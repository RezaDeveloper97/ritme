// Public API of the `pregnancy-dating` widget (CLAUDE.md §3.3): the sign-up
// pregnancy dating form, shared by onboarding and `/pregnancy/setup`.
export {
  EMPTY_PREGNANCY_ANSWERS,
  gestationalDays,
  MAX_GA_DAYS,
  PREGNANCY_BASES,
  pregnancyOnboardingBody,
  pregnancySummary,
  TERM_DAYS,
  type PregnancyAnswers,
  type PregnancyBasis,
} from './model/dating';
export { PregnancyDatingForm } from './ui/PregnancyDatingForm';

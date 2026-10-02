/**
 * Onboarding v2 (B-N2-02, nbl_Onb_*): the order of the screens and where each
 * one leads. Pure so the branching is unit-tested (`flow.test.ts`).
 *
 * Phone → OTP → Name → Gender → (woman) Goal → Cycle | Preg | Meno →
 * Conditions → Health → Ready. A man goes Gender → Partner (companion code,
 * B-N4-05) → Linked → companion panel, or «بعداً» → Ready → companion panel.
 */

export type OnboardingGoal = 'cycle' | 'ttc' | 'pregnancy' | 'menopause';
export type OnboardingGender = 'female' | 'male';

export const GOALS: readonly OnboardingGoal[] = ['cycle', 'ttc', 'pregnancy', 'menopause'];

export type FlowStep =
  | 'name'
  | 'gender'
  | 'goal'
  | 'cycle'
  | 'pregnancy'
  | 'menopause'
  | 'conditions'
  | 'health'
  | 'partner'
  | 'ready';

/** Route of each screen (the legacy paths are kept where the screen is a restyle). */
export const FLOW_ROUTES: Record<FlowStep, string> = {
  name: '/onboarding/name',
  gender: '/onboarding/gender',
  goal: '/onboarding/intention',
  cycle: '/onboarding/cycle',
  pregnancy: '/onboarding/pregnancy-basis',
  menopause: '/onboarding/menopause',
  conditions: '/onboarding/conditions',
  health: '/onboarding/health',
  partner: '/onboarding/partner',
  ready: '/onboarding/setting-up',
};

/** Segments of the header progress bar, phone + OTP included. */
export const PROGRESS_TOTAL = 8;

/** Filled segments per screen (phone 1, OTP 2, … health 8). */
export const PROGRESS: Record<'phone' | 'otp' | Exclude<FlowStep, 'ready'>, number> = {
  phone: 1,
  otp: 2,
  name: 3,
  gender: 4,
  goal: 5,
  cycle: 6,
  pregnancy: 6,
  menopause: 6,
  partner: 5,
  conditions: 7,
  health: 8,
};

/** The goal's own branch screen. */
export function branchStep(goal: OnboardingGoal): FlowStep {
  if (goal === 'pregnancy') return 'pregnancy';
  if (goal === 'menopause') return 'menopause';
  return 'cycle';
}

/** Where «ادامه» (or «رد کردن») leads from `step`. */
export function nextStep(step: FlowStep, ctx: { gender?: OnboardingGender | null; goal?: OnboardingGoal | null }): FlowStep {
  switch (step) {
    case 'name':
      return 'gender';
    case 'gender':
      return ctx.gender === 'male' ? 'partner' : 'goal';
    case 'goal':
      return branchStep(ctx.goal ?? 'cycle');
    case 'cycle':
    case 'pregnancy':
    case 'menopause':
      return 'conditions';
    case 'conditions':
      return 'health';
    default:
      return 'ready';
  }
}

/**
 * Where the header back arrow leads — a known route, never history
 * (CLAUDE.md §4.2). Name goes back to the phone screen.
 */
export function previousRoute(step: FlowStep, goal: OnboardingGoal | null): string {
  switch (step) {
    case 'name':
      return '/signup';
    case 'gender':
      return FLOW_ROUTES.name;
    case 'goal':
    case 'partner':
      return FLOW_ROUTES.gender;
    case 'cycle':
    case 'pregnancy':
    case 'menopause':
      return FLOW_ROUTES.goal;
    case 'conditions':
      return FLOW_ROUTES[branchStep(goal ?? 'cycle')];
    case 'health':
      return FLOW_ROUTES.conditions;
    default:
      return FLOW_ROUTES.health;
  }
}

/**
 * The resume marker written for each screen. The edge middleware (and the OTP
 * screen) only understand the legacy step keys of `entities/user` (owned by
 * another task), so each new screen is stored as the legacy key whose route
 * leads back to it — directly or through the redirect of a merged route
 * (`cycleLen` → /onboarding/cycle, `birthday` → /onboarding/health). Gender,
 * Meno and Partner have no legacy twin: they resume one screen earlier.
 */
export const RESUME_KEYS: Record<Exclude<FlowStep, 'ready'>, string> = {
  name: 'name',
  gender: 'name',
  goal: 'intention',
  cycle: 'cycleLen',
  pregnancy: 'pregnancyBasis',
  menopause: 'intention',
  conditions: 'conditions',
  health: 'birthday',
  partner: 'name',
};

/** The flow screen a pathname shows (locale prefix allowed), or null. */
export function flowStepFromPath(pathname: string): FlowStep | null {
  const segments = pathname.split('/').filter(Boolean);
  const at = segments.indexOf('onboarding');
  if (at === -1) return null;
  const route = `/onboarding/${segments[at + 1] ?? ''}`;
  const hit = (Object.entries(FLOW_ROUTES) as [FlowStep, string][]).find(([, r]) => r === route);
  return hit?.[0] ?? null;
}

/** The «به … وصل شدی» screen after a code was accepted (nbl_Onb_PartnerLinked). */
export const PARTNER_LINKED_ROUTE = '/onboarding/partner/linked';

/**
 * Where «ورود به ریتمی» lands: each mode's home; a man's is `/companion`. A pregnant user who skipped
 * the dating screen has no active pregnancy yet, so she finishes it on the
 * pregnancy onboarding screen instead of an empty pregnancy home.
 */
export function landingRoute(
  goal: OnboardingGoal | null,
  pregnancyActive: boolean,
  gender: OnboardingGender | null = null,
): string {
  // A man has no cycle of his own: his home is the companion panel (B-N4-05, N2 stage bug B-1).
  if (gender === 'male') return '/companion';
  if (goal === 'pregnancy') return pregnancyActive ? '/pregnancy' : '/pregnancy/onboarding';
  return '/home';
}

export function isGoal(value: unknown): value is OnboardingGoal {
  return typeof value === 'string' && (GOALS as readonly string[]).includes(value);
}

import { OnboardingResumeTracker } from '@/entities/user';

/**
 * Wraps every onboarding step so the resume marker is maintained in one place
 * (see `OnboardingResumeTracker`) instead of in each step's "next" handler.
 */
export default function OnboardingLayout({ children }: { children: React.ReactNode }) {
  return (
    <>
      <OnboardingResumeTracker />
      {children}
    </>
  );
}

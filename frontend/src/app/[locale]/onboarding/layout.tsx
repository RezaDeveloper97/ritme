import { FlowResumeTracker } from '@/screens/onboarding-flow';

/**
 * Wraps every onboarding step so the resume marker is maintained in one place
 * (see `FlowResumeTracker`) instead of in each step's "next" handler.
 */
export default function OnboardingLayout({ children }: { children: React.ReactNode }) {
  return (
    <>
      <FlowResumeTracker />
      {children}
    </>
  );
}

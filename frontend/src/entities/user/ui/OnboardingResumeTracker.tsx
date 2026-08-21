'use client';

import { usePathname } from 'next/navigation';
import { useEffect } from 'react';

import { setOnboardingPending } from '@/shared/session';

import { onboardingStepFromPath } from '../model/steps';

/**
 * Records the onboarding step currently on screen, so an interrupted signup
 * resumes where it stopped instead of dumping the visitor into the app (which
 * the auth cookie alone would allow — it is granted at OTP verification, long
 * before registration is finished).
 *
 * Mounted once in the onboarding layout rather than written by each step: the
 * steps are independent routes and several of them navigate backwards, so
 * deriving the position from the pathname is the only account that can't drift.
 * The save screen is deliberately not a step — it clears the marker itself once
 * the answers are persisted.
 */
export function OnboardingResumeTracker() {
  const pathname = usePathname();

  useEffect(() => {
    const step = onboardingStepFromPath(pathname);
    if (step) setOnboardingPending(step);
  }, [pathname]);

  return null;
}

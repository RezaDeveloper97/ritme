'use client';

import { usePathname } from 'next/navigation';
import { useEffect } from 'react';

import { setOnboardingPending } from '@/shared/session';

import { flowStepFromPath, RESUME_KEYS } from '../model/flow';

/**
 * Keeps the «registration unfinished» marker on the screen being shown, so an
 * interrupted signup resumes there (the middleware reads it). Mounted once in
 * the onboarding layout. The Ready screen is not a step: it clears the marker
 * itself once onboarding is complete.
 */
export function FlowResumeTracker() {
  const pathname = usePathname();
  useEffect(() => {
    const step = flowStepFromPath(pathname);
    if (step && step !== 'ready') setOnboardingPending(RESUME_KEYS[step]);
  }, [pathname]);
  return null;
}

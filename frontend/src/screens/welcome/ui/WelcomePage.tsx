'use client';

import { useRouter } from '@/shared/i18n';
import { markIntroSeen } from '@/shared/session';
import { IntroCarousel } from '@/widgets/intro-carousel';

/**
 * The pre-signup welcome screen. First-time visitors land here (routed from the
 * splash); once they finish or skip the intro we remember it and never show it
 * again — the next stop is always signup.
 */
export function WelcomePage() {
  const router = useRouter();

  const handleComplete = () => {
    markIntroSeen();
    router.replace('/signup');
  };

  return <IntroCarousel onComplete={handleComplete} />;
}

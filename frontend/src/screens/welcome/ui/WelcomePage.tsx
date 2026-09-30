'use client';

import { useSyncExternalStore } from 'react';

import { useRouter } from '@/shared/i18n';
import { hasSeenIntro, markIntroSeen } from '@/shared/session';
import { IntroCarousel } from '@/widgets/intro-carousel';

import { resolveWelcomeView, type WelcomeView } from '../lib/view';
import { WelcomeCard } from './WelcomeCard';

const noop = () => () => {};
// A string snapshot so useSyncExternalStore can compare it by value.
const clientView = (): string =>
  JSON.stringify(resolveWelcomeView(window.location.search, hasSeenIntro()));
const serverView = (): string => JSON.stringify(resolveWelcomeView('', false));

/**
 * The pre-signup welcome route. First-time visitors get the five intro slides;
 * someone who has already seen them (or arrives with `?step=welcome`) gets the
 * short Night & Bloom welcome card instead. Either way the next stop is
 * signup, and finishing marks the intro as seen.
 *
 * The view is read from the cookie + query on the client. A client-side
 * navigation (the usual way in, from the splash) reads it before painting;
 * only a cold load hydrates from the first-visit default first.
 */
export function WelcomePage() {
  const router = useRouter();
  const view = JSON.parse(useSyncExternalStore(noop, clientView, serverView)) as WelcomeView;

  const toSignup = () => {
    markIntroSeen();
    router.replace('/signup');
  };

  if (view.kind === 'card') return <WelcomeCard onStart={toSignup} onLogin={toSignup} />;
  return <IntroCarousel key={view.slide} initialIndex={view.slide} onComplete={toSignup} onLogin={toSignup} />;
}

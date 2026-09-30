'use client';

import { useTranslations } from 'next-intl';
import { useEffect } from 'react';

import { useRouter } from '@/shared/i18n';
import { hasSeenIntro } from '@/shared/session';
import { CycleDotRing } from '@/widgets/intro-carousel';

/**
 * Night & Bloom `Splash`: the dotted cycle ring around the wordmark on the
 * page canvas (light and dark), a soft glow and three loading dots.
 */
export function SplashPage() {
  const t = useTranslations('auth.splash');
  const router = useRouter();

  // First-time visitors see the welcome intro before signup; once they've seen
  // it, the splash goes straight to signup. Decided per render so a fresh visit
  // (cookies cleared) shows the intro again — the route bakes the same decision
  // into its no-JS fallback.
  const next = () => router.replace(hasSeenIntro() ? '/signup' : '/welcome');

  useEffect(() => {
    // The page is hydrated, so the route's no-JS fallback has nothing left to
    // rescue — cancel it before it navigates on top of a working screen.
    window.__ritmeSplashFallback?.();
    const timer = setTimeout(next, 2200);
    return () => clearTimeout(timer);
    // `next` reads only refs/router; re-running on router identity is enough.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [router]);

  return (
    <div className="view ib-splash" onClick={next}>
      <span className="ib-splash-glow" aria-hidden />
      <CycleDotRing today={0} className="is-splash">
        <h1 className="ib-brand is-xl">{t('brand')}</h1>
      </CycleDotRing>
      <p className="ib-splash-tag">{t('tagline')}</p>
      <span className="ib-loading" role="status" aria-label={t('loading')}>
        <span />
        <span />
        <span />
      </span>
    </div>
  );
}

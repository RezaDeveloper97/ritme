'use client';

import { useTranslations } from 'next-intl';
import Image from 'next/image';
import { useEffect } from 'react';

import { Icon } from '@/shared/ui';
import { useRouter } from '@/shared/i18n';
import { hasSeenIntro } from '@/shared/session';

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
    <div className="view splash" onClick={next}>
      {/* Figma: faint concentric-circles mark peeking from the top corner */}
      <div aria-hidden className="splash-mark">
        <svg width="180" height="180" viewBox="0 0 86 86" fill="none">
          <circle cx="43" cy="43" r="36" stroke="var(--on-accent)" strokeWidth="1.5" />
          <circle cx="43" cy="43" r="25" stroke="var(--on-accent)" strokeWidth="1.5" />
          <circle cx="43" cy="43" r="14.5" fill="var(--on-accent)" />
        </svg>
      </div>

      <div className="splash-center">
        {/* App icon */}
        <Image
          src="/logo.webp"
          alt=""
          aria-hidden
          width={84}
          height={84}
          priority
          className="splash-logo"
        />

        <div className="splash-name">ریـتمی</div>
        <div className="splash-tagline">{t('tagline')}</div>
      </div>

      {/* Bottom */}
      <div className="splash-bottom">
        <span className="splash-spinner">
          <Icon name="loader" size={24} stroke="var(--on-accent)" />
        </span>
        <span className="splash-copy">{t('copyright')}</span>
      </div>
    </div>
  );
}

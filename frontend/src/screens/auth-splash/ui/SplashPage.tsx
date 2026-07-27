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
  // (localStorage cleared) shows the intro again.
  const next = () => router.replace(hasSeenIntro() ? '/signup' : '/welcome');

  useEffect(() => {
    const timer = setTimeout(next, 2200);
    return () => clearTimeout(timer);
    // `next` reads only refs/router; re-running on router identity is enough.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [router]);

  return (
    <div
      className="view"
      style={{
        background: 'linear-gradient(160deg,var(--brand) 0%,var(--purple) 100%)',
        color: 'var(--on-accent)',
        cursor: 'pointer',
        position: 'relative',
        height: '100%',
        overflow: 'hidden',
      }}
      onClick={next}
    >
      {/* Figma: faint concentric-circles mark peeking from the top corner */}
      <div aria-hidden style={{ position: 'absolute', top: -60, insetInlineEnd: -40, opacity: .12, pointerEvents: 'none' }}>
        <svg width="180" height="180" viewBox="0 0 86 86" fill="none">
          <circle cx="43" cy="43" r="36" stroke="var(--on-accent)" strokeWidth="1.5" />
          <circle cx="43" cy="43" r="25" stroke="var(--on-accent)" strokeWidth="1.5" />
          <circle cx="43" cy="43" r="14.5" fill="var(--on-accent)" />
        </svg>
      </div>

      <div
        style={{
          position: 'absolute', inset: 0,
          display: 'flex', flexDirection: 'column',
          alignItems: 'center', justifyContent: 'center', gap: 14,
        }}
      >
        {/* App icon */}
        <Image
          src="/logo.webp"
          alt=""
          aria-hidden
          width={84}
          height={84}
          priority
          style={{
            width: 84, height: 84, borderRadius: 26,
            boxShadow: '0 18px 40px -12px rgba(0,0,0,.4)',
          }}
        />

        <div style={{ fontWeight: 900, fontSize: 34, letterSpacing: 1, marginTop: 6 }}>ریـتمی</div>
        <div style={{ fontSize: 13, opacity: .92, fontWeight: 500 }}>{t('tagline')}</div>
      </div>

      {/* Bottom */}
      <div
        style={{
          position: 'absolute', bottom: 30, insetInline: 0,
          display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 10,
        }}
      >
        <span style={{ animation: 'spin 1s linear infinite', display: 'inline-flex', opacity: .95 }}>
          <Icon name="loader" size={24} stroke="var(--on-accent)" />
        </span>
        <span style={{ fontSize: 10, opacity: .8 }}>{t('copyright')}</span>
      </div>
    </div>
  );
}

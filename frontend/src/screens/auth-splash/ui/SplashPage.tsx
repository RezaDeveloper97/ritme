'use client';

import { useTranslations } from 'next-intl';
import { useEffect } from 'react';

import { Icon } from '@/shared/ui';
import { useRouter } from '@/shared/i18n';

export function SplashPage() {
  const t = useTranslations('auth.splash');
  const router = useRouter();

  useEffect(() => {
    const timer = setTimeout(() => router.replace('/signup'), 2200);
    return () => clearTimeout(timer);
  }, [router]);

  return (
    <div
      className="view"
      style={{
        background: 'linear-gradient(160deg,#E91E63 0%,#BA68C8 100%)',
        color: '#fff',
        cursor: 'pointer',
        position: 'relative',
        height: '100%',
        overflow: 'hidden',
      }}
      onClick={() => router.replace('/signup')}
    >
      {/* Figma: faint concentric-circles mark peeking from the top corner */}
      <div aria-hidden style={{ position: 'absolute', top: -60, insetInlineEnd: -40, opacity: .12, pointerEvents: 'none' }}>
        <svg width="180" height="180" viewBox="0 0 86 86" fill="none">
          <circle cx="43" cy="43" r="36" stroke="#fff" strokeWidth="1.5" />
          <circle cx="43" cy="43" r="25" stroke="#fff" strokeWidth="1.5" />
          <circle cx="43" cy="43" r="14.5" fill="#fff" />
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
        <div
          style={{
            width: 84, height: 84, borderRadius: 26,
            background: 'rgba(255,255,255,.16)',
            backdropFilter: 'blur(6px)',
            display: 'flex', alignItems: 'center', justifyContent: 'center',
            boxShadow: '0 18px 40px -12px rgba(0,0,0,.4)',
          }}
        >
          <Icon name="sparkle" size={46} fill="#fff" strokeWidth={0} />
        </div>

        <div style={{ fontWeight: 900, fontSize: 34, letterSpacing: 1, marginTop: 6 }}>ریـتمی</div>
        <div style={{ fontSize: 13, opacity: .92, fontWeight: 500 }}>{t('tagline')}</div>
      </div>

      {/* Bottom */}
      <div
        style={{
          position: 'absolute', bottom: 30, left: 0, right: 0,
          display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 10,
        }}
      >
        <span style={{ animation: 'spin 1s linear infinite', display: 'inline-flex', opacity: .95 }}>
          <Icon name="loader" size={24} stroke="#fff" />
        </span>
        <span style={{ fontSize: 10, opacity: .8 }}>{t('copyright')}</span>
      </div>
    </div>
  );
}

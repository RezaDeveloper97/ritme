'use client';

import { useTranslations } from 'next-intl';

import { LoginForm } from '@/features/auth';
import { ThemeToggle, UiLocaleSelect } from '@/shared/ui';

// A 28-day strip: the rhythm the product is named for. Decorative only; the
// turquoise bar is "today" (turquoise = data, frontend/CLAUDE.md §10.2).
const DAYS = Array.from({ length: 28 }, (_, i) => i);
const TODAY = 13;

export function LoginScreen() {
  const t = useTranslations('auth');
  const tc = useTranslations('common');
  return (
    <main className="login-canvas">
      <section className="login-art" aria-hidden="true">
        <div className="rhythm">
          {DAYS.map((d) => (
            <span key={d} data-now={d === TODAY || undefined} />
          ))}
        </div>
        <span className="brand-wordmark">{tc('appName')}</span>
        <p className="m-0 mt-3 max-w-[32ch] text-[15px] text-ink-3">{t('tagline')}</p>
      </section>
      <section className="flex flex-col px-5 py-6 sm:px-10">
        <div className="flex items-center justify-end gap-2">
          <UiLocaleSelect />
          <ThemeToggle />
        </div>
        <div className="mx-auto flex w-full max-w-sm flex-1 flex-col justify-center gap-6 py-8">
          <div className="flex flex-col gap-1">
            <h1 className="m-0 text-2xl font-extrabold">{t('title')}</h1>
            <p className="m-0 text-ink-3">{t('subtitle')}</p>
          </div>
          <LoginForm />
        </div>
      </section>
    </main>
  );
}

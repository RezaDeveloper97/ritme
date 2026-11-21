'use client';

import clsx from 'clsx';
import { useTranslations } from 'next-intl';
import type { ReactNode } from 'react';

import { Link, usePathname } from '@/shared/i18n';
import { useMounted } from '@/shared/lib/use-mounted';
import { isAuthenticated } from '@/shared/session';

import { useAppRedirect } from './useAppRedirect';
import '@/shared/ui/styles/ws.css';

/** One header/footer link. `href` is locale-less (`/places`); `match` is the prefix that marks it active. */
export interface WebNavItem {
  href: string;
  label: string;
  match?: string;
}

export interface WebShellProps {
  children: ReactNode;
  /** Header navigation. Each web page group passes its own (the directory: «خدمات مادر و کودک»). */
  nav?: WebNavItem[];
  /** Extra footer links (terms, contact…). Omitted → the footer shows the brand and the privacy note only. */
  footerLinks?: WebNavItem[];
  /** Header call-to-action (the directory: «ثبت مجموعه»). */
  cta?: WebNavItem;
  /** The app route this page maps to: signed-in visitors under 1024px are sent there (see `useAppRedirect`). */
  appHref?: string;
  /** Where «باز کردن اپ» leads for a signed-in visitor. */
  appHome?: string;
  /** Where the brand links to. */
  homeHref?: string;
}

/**
 * The desktop web frame: sticky header (brand, nav, login / open-app, optional CTA), the page and a footer.
 * Generic on purpose — pages configure `nav` / `cta` / `footerLinks` and never touch the markup. Store badges are not
 * rendered until real store links exist. Consumers must have the `webShell` namespace in their route scope
 * (`app/message-scopes.ts`). The root `.ws-root` also lifts the app's phone-width column (globals.css, CB-DIR-11).
 */
export function WebShell({ children, nav = [], footerLinks = [], cta, appHref, appHome = '/home', homeHref = '/places' }: WebShellProps) {
  const t = useTranslations('webShell');
  const pathname = usePathname();
  const mounted = useMounted();
  useAppRedirect(appHref);
  const authed = mounted && isAuthenticated();

  return (
    <div className="ws-root">
      <a className="ws-skip" href="#ws-main">
        {t('skip')}
      </a>
      <header className="ws-header">
        <div className="ws-bar">
          <Link href={homeHref} className="ws-brand">
            {t('brand')}
          </Link>
          {nav.length > 0 ? (
            <nav aria-label={t('navLabel')} className="ws-nav">
              {nav.map((n) => {
                const prefix = n.match ?? n.href;
                const on = pathname === prefix || pathname.startsWith(`${prefix}/`);
                return (
                  <Link key={n.href} href={n.href} className={clsx('ws-nav-link', on && 'on')} aria-current={on ? 'page' : undefined}>
                    {n.label}
                  </Link>
                );
              })}
            </nav>
          ) : (
            <span className="ws-grow" />
          )}
          <div className="ws-actions">
            {cta ? (
              <Link href={cta.href} className="ws-cta">
                {cta.label}
              </Link>
            ) : null}
            <Link href={authed ? appHome : '/signup'} className="ws-login">
              {authed ? t('openApp') : t('login')}
            </Link>
          </div>
        </div>
      </header>
      <main id="ws-main" className="ws-main">
        {children}
      </main>
      <footer className="ws-footer">
        <div className="ws-foot-in">
          <div className="ws-foot-brand">
            <strong>{t('brand')}</strong>
            <span>{t('footer.tagline')}</span>
          </div>
          {footerLinks.length > 0 ? (
            <nav aria-label={t('footer.label')} className="ws-foot-links">
              {footerLinks.map((n) => (
                <Link key={n.href} href={n.href}>
                  {n.label}
                </Link>
              ))}
            </nav>
          ) : null}
          <p className="ws-foot-note">{t('footer.note')}</p>
        </div>
      </footer>
    </div>
  );
}

'use client';

import { useTranslations } from 'next-intl';
import { useEffect, useRef, useState } from 'react';

import { useAppUpdate } from './useAppUpdate';

/**
 * Mounted once at the app root. Renders nothing until an update is detected,
 * then either a dismissible bottom toast (soft) or a full-screen blocking
 * overlay (forced — running version below minSupportedVersion).
 */
export function UpdateGate() {
  const t = useTranslations('pwa');
  const { status, latestVersion, releaseNotes, apply } = useAppUpdate();
  const [dismissedVersion, setDismissedVersion] = useState<string | null>(null);
  const [applying, setApplying] = useState(false);
  const forcedButtonRef = useRef<HTMLButtonElement>(null);

  const forced = status === 'forced';

  // The forced overlay must trap the user: no scroll-behind, no Escape,
  // focus pinned to the single update button.
  useEffect(() => {
    if (!forced) return;
    document.body.classList.add('pwa-locked');
    forcedButtonRef.current?.focus();
    const onKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') e.preventDefault();
      if (e.key === 'Tab') {
        e.preventDefault();
        forcedButtonRef.current?.focus();
      }
    };
    document.addEventListener('keydown', onKeyDown, true);
    return () => {
      document.body.classList.remove('pwa-locked');
      document.removeEventListener('keydown', onKeyDown, true);
    };
  }, [forced]);

  const onApply = () => {
    setApplying(true);
    void apply();
  };

  if (forced) {
    return (
      <div className="pwa-forced" role="alertdialog" aria-modal="true" aria-label={t('forcedTitle')}>
        <div className="pwa-forced-card">
          <span className="pwa-forced-badge" aria-hidden="true">
            <svg width="28" height="28" viewBox="0 0 24 24" fill="none" aria-hidden="true">
              <path
                fill="currentColor"
                d="M18 14l-5-5l1.4-1.4l2.6 2.6V3h2v7.2l2.6-2.6L23 9zM7 23q-.825 0-1.412-.587T5 21V3q0-.825.588-1.412T7 1h7v5H7v12h10v-2h2v5q0 .825-.587 1.413T17 23z"
              />
            </svg>
          </span>
          <h2 className="pwa-forced-title">{t('forcedTitle')}</h2>
          <p className="pwa-forced-body">{t('forcedBody')}</p>
          {releaseNotes ? <p className="pwa-notes">{releaseNotes}</p> : null}
          <button
            ref={forcedButtonRef}
            type="button"
            className="pwa-btn-primary"
            onClick={onApply}
            disabled={applying}
          >
            {applying ? t('updating') : t('updateNow')}
          </button>
        </div>
      </div>
    );
  }

  // SW-channel-only updates have no version string; key them as 'sw' so
  // dismissal still works (and re-arms when a different version appears).
  const updateKey = latestVersion ?? 'sw';

  if (status === 'soft' && updateKey !== dismissedVersion) {
    return (
      <div className="pwa-toast" role="status">
        <div className="pwa-toast-text">
          <strong>{t('softTitle')}</strong>
          {releaseNotes ? <span className="pwa-notes">{releaseNotes}</span> : null}
        </div>
        <div className="pwa-toast-actions">
          <button type="button" className="pwa-btn-primary" onClick={onApply} disabled={applying}>
            {applying ? t('updating') : t('updateAction')}
          </button>
          <button
            type="button"
            className="pwa-btn-dismiss"
            onClick={() => setDismissedVersion(updateKey)}
            aria-label={t('dismiss')}
          >
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" aria-hidden="true">
              <path
                fill="currentColor"
                d="M6.4 19L5 17.6l5.6-5.6L5 6.4L6.4 5l5.6 5.6L17.6 5L19 6.4L13.4 12l5.6 5.6l-1.4 1.4l-5.6-5.6z"
              />
            </svg>
          </button>
        </div>
      </div>
    );
  }

  return null;
}

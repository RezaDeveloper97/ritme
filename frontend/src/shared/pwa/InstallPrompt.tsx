'use client';

import { useTranslations } from 'next-intl';
import { useEffect, useState } from 'react';

// Icon paths from Material Symbols (via Iconify): ios-share, add-box-outline,
// install-mobile. Inlined so the prompt renders offline with zero requests.
function ShareIosIcon() {
  return (
    <svg width="18" height="18" viewBox="0 0 24 24" aria-hidden="true">
      <path
        fill="currentColor"
        d="M6 22q-.825 0-1.412-.587T4 20V10q0-.825.588-1.412T6 8h3v2H6v10h12V10h-3V8h3q.825 0 1.413.588T20 10v10q0 .825-.587 1.413T18 22zm5-6V4.825l-1.6 1.6L8 5l4-4l4 4l-1.4 1.425l-1.6-1.6V16z"
      />
    </svg>
  );
}

function AddToHomeIcon() {
  return (
    <svg width="18" height="18" viewBox="0 0 24 24" aria-hidden="true">
      <path
        fill="currentColor"
        d="M11 17h2v-4h4v-2h-4V7h-2v4H7v2h4zm-6 4q-.825 0-1.412-.587T3 19V5q0-.825.588-1.412T5 3h14q.825 0 1.413.588T21 5v14q0 .825-.587 1.413T19 21zm0-2h14V5H5z"
      />
    </svg>
  );
}

function InstallMobileIcon() {
  return (
    <svg width="22" height="22" viewBox="0 0 24 24" aria-hidden="true">
      <path
        fill="currentColor"
        d="M7 23q-.825 0-1.412-.587T5 21V3q0-.825.588-1.412T7 1h7v5H7v12h10v-2h2v5q0 .825-.587 1.413T17 23zm11-9l-5-5l1.4-1.4l2.6 2.6V3h2v7.2l2.6-2.6L23 9z"
      />
    </svg>
  );
}

interface BeforeInstallPromptEvent extends Event {
  prompt: () => Promise<void>;
  userChoice: Promise<{ outcome: 'accepted' | 'dismissed' }>;
}

const DISMISS_KEY = 'ritme-install-dismissed';

function isStandalone(): boolean {
  return (
    window.matchMedia('(display-mode: standalone)').matches ||
    (navigator as Navigator & { standalone?: boolean }).standalone === true
  );
}

function isIos(): boolean {
  const ua = navigator.userAgent;
  // iPadOS 13+ reports as Macintosh but has a touch screen.
  return /iPhone|iPad|iPod/.test(ua) || (ua.includes('Macintosh') && navigator.maxTouchPoints > 1);
}

/**
 * Custom install affordance. On Chromium (Android/desktop) it captures
 * `beforeinstallprompt` and shows a native-prompt button; on iOS Safari —
 * which has no install event — it shows Add-to-Home-Screen instructions
 * with the Share and Add icons. Dismissal is remembered per device.
 */
export function InstallPrompt() {
  const t = useTranslations('pwa');
  const [installEvent, setInstallEvent] = useState<BeforeInstallPromptEvent | null>(null);
  const [showIosGuide, setShowIosGuide] = useState(false);

  useEffect(() => {
    if (isStandalone() || window.localStorage.getItem(DISMISS_KEY)) return;

    if (isIos()) {
      setShowIosGuide(true);
      return;
    }

    const onPrompt = (e: Event) => {
      e.preventDefault();
      setInstallEvent(e as BeforeInstallPromptEvent);
    };
    const onInstalled = () => setInstallEvent(null);
    window.addEventListener('beforeinstallprompt', onPrompt);
    window.addEventListener('appinstalled', onInstalled);
    return () => {
      window.removeEventListener('beforeinstallprompt', onPrompt);
      window.removeEventListener('appinstalled', onInstalled);
    };
  }, []);

  const dismiss = () => {
    window.localStorage.setItem(DISMISS_KEY, '1');
    setInstallEvent(null);
    setShowIosGuide(false);
  };

  const install = async () => {
    if (!installEvent) return;
    await installEvent.prompt();
    const { outcome } = await installEvent.userChoice;
    if (outcome === 'dismissed') {
      window.localStorage.setItem(DISMISS_KEY, '1');
    }
    setInstallEvent(null);
  };

  if (installEvent) {
    return (
      <div className="pwa-install" role="complementary" aria-label={t('installTitle')}>
        <span className="pwa-install-icon" aria-hidden="true">
          <InstallMobileIcon />
        </span>
        <div className="pwa-install-text">
          <strong>{t('installTitle')}</strong>
          <span>{t('installBody')}</span>
        </div>
        <div className="pwa-toast-actions">
          <button type="button" className="pwa-btn-primary" onClick={() => void install()}>
            {t('installAction')}
          </button>
          <button type="button" className="pwa-btn-dismiss" onClick={dismiss} aria-label={t('dismiss')}>
            <svg width="16" height="16" viewBox="0 0 24 24" aria-hidden="true">
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

  if (showIosGuide) {
    return (
      <div className="pwa-install pwa-install-ios" role="complementary" aria-label={t('installTitle')}>
        <div className="pwa-install-text">
          <strong>{t('installTitle')}</strong>
          <span className="pwa-ios-step">
            {t('iosStep1')}
            <span className="pwa-step-icon" aria-hidden="true">
              <ShareIosIcon />
            </span>
          </span>
          <span className="pwa-ios-step">
            {t('iosStep2')}
            <span className="pwa-step-icon" aria-hidden="true">
              <AddToHomeIcon />
            </span>
          </span>
        </div>
        <button type="button" className="pwa-btn-dismiss" onClick={dismiss} aria-label={t('dismiss')}>
          <svg width="16" height="16" viewBox="0 0 24 24" aria-hidden="true">
            <path
              fill="currentColor"
              d="M6.4 19L5 17.6l5.6-5.6L5 6.4L6.4 5l5.6 5.6L17.6 5L19 6.4L13.4 12l5.6 5.6l-1.4 1.4l-5.6-5.6z"
            />
          </svg>
        </button>
      </div>
    );
  }

  return null;
}

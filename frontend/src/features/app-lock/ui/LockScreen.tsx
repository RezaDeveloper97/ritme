'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useEffect, useState } from 'react';

import { type EmergencyCard, EmergencyCardView, useLockEmergencyCard } from '@/entities/health-record';
import { apiClient } from '@/shared/api';
import { type Locale, localizeHref } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib/date';
import { clearAuthToken, isAuthenticated } from '@/shared/session';
import { Icon, IconCircle, PrimaryButton, ScreenHeader, SecondaryButton, SkyLayer } from '@/shared/ui';

import type { LockSnapshot } from '../model/controller';
import { getLockController } from '../model/store';
import { verifyBiometric } from '../model/webauthn';
import { PasscodePad } from './PasscodePad';

function useSecondsLeft(until: number): number {
  const [now, setNow] = useState(() => Date.now());
  useEffect(() => {
    if (until <= now) return;
    const id = window.setInterval(() => setNow(Date.now()), 1000);
    return () => window.clearInterval(id);
  }, [until, now]);
  return Math.max(0, Math.ceil((until - now) / 1000));
}

/**
 * Full-screen lock (B-N1-12). Rendered by {@link AppLockGate} *instead of* the
 * app, so nothing of the signed-in screens is in the DOM while it shows.
 * «رمز را فراموش کردی؟» signs out (server logout + local wipe, which also
 * removes the lock) — the only way past the lock without the passcode, and it
 * costs the session: getting back in needs the SMS code.
 */
let ending = false;

/** Server logout, then the local wipe (session cleanups remove the lock config) and a full document replace. */
async function endLockedSession(loc: Locale): Promise<void> {
  if (ending) return;
  ending = true;
  await apiClient.post('/auth/logout').catch(() => undefined);
  clearAuthToken();
  window.location.replace(localizeHref('/signup', loc));
}

/**
 * «کارت اضطراری» from the lock screen (CB-PRIV-01): only the emergency card is fetched and shown — the app's tree
 * stays unmounted behind the lock, so no other data is reachable from here. The link appears only when the owner
 * switched on «نمایش روی صفحه قفل» (CB-REC-03 `show_on_lock_screen`). It reads only
 * `GET /health-record/emergency-card/lock` (D-73) — the minimal card — never the owner view.
 */
function EmergencyCardPanel({ card, onBack }: { card: EmergencyCard; onBack: () => void }) {
  const t = useTranslations('common.appLock');
  return (
    <div className="view lk-page" role="dialog" aria-modal="true" aria-labelledby="lk-ec-title">
      <SkyLayer />
      <div className="scroll lk-ec">
        <ScreenHeader title={<span id="lk-ec-title">{t('emergency')}</span>} onBack={onBack} backLabel={t('emergencyBack')} />
        <EmergencyCardView card={card} variant="lock" />
        <aside className="nb-card lk-ec-note">
          <Icon name="shield" size={16} className="lk-ec-note-icon" />
          <p>{t('emergencyNote')}</p>
        </aside>
      </div>
    </div>
  );
}

export function LockScreen({ state }: { state: LockSnapshot }) {
  const t = useTranslations('common.appLock');
  const loc = useLocale() as Locale;
  const [showCard, setShowCard] = useState(false);
  // The lock-screen route answers no data unless the owner enabled it; no fetch once the session is ending.
  const emergency = useLockEmergencyCard({ enabled: !state.lockedOut && isAuthenticated() });
  const lockCard = state.lockedOut ? null : (emergency.data ?? null);
  const emergencyOn = lockCard !== null;
  const [value, setValue] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [checking, setChecking] = useState(false);
  const [forgot, setForgot] = useState(false);
  const [leaving, setLeaving] = useState(false);
  const secondsLeft = useSecondsLeft(state.blockedUntil);
  const blocked = secondsLeft > 0;

  const submit = async (code: string) => {
    const c = getLockController();
    if (!c || checking) return;
    setChecking(true);
    const result = await c.unlockWithPasscode(code);
    setChecking(false);
    if (result === 'ok') return;
    setValue('');
    setError(result === 'wrong' ? t('wrong') : null);
  };

  const biometric = async () => {
    const c = getLockController();
    const id = c?.getCredentialId();
    if (!c || !id) return;
    if (await verifyBiometric(id)) c.unlockWithBiometric();
    else setError(t('biometricFailed'));
  };

  const signOut = async () => {
    if (leaving) return;
    setLeaving(true);
    await endLockedSession(loc);
  };

  // Too many wrong passcodes: the session ends on its own (same path as «فراموشی رمز»).
  useEffect(() => {
    if (state.lockedOut) void endLockedSession(loc);
  }, [state.lockedOut, loc]);

  if (showCard && lockCard) return <EmergencyCardPanel card={lockCard} onBack={() => setShowCard(false)} />;

  const title = <h1 id="lk-title" className="lk-title">{t('title')}</h1>;

  if (state.lockedOut) {
    return (
      <div className="view lk-page" role="alertdialog" aria-modal="true" aria-labelledby="lk-title">
        <SkyLayer />
        <div className="lk-body">
          <IconCircle icon="lock" tone="brand" size="lg" className="lk-icon" />
          {title}
          <p className="lk-sub" role="alert">
            {t('lockedOut')}
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className="view lk-page" role="dialog" aria-modal="true" aria-labelledby="lk-title">
      <SkyLayer />
      <div className="lk-body">
        <IconCircle icon="lock" tone="brand" size="lg" className="lk-icon" />
        {forgot ? (
          <div className="lk-forgot">
            {title}
            <p className="lk-sub">{t('forgotBody')}</p>
            <PrimaryButton block loading={leaving} onClick={() => void signOut()}>
              {t('forgotConfirm')}
            </PrimaryButton>
            <SecondaryButton block onClick={() => setForgot(false)}>
              {t('forgotCancel')}
            </SecondaryButton>
          </div>
        ) : (
          <>
            {title}
            <p className="lk-sub">{t('subtitle')}</p>
            <PasscodePad
              labelledBy="lk-title"
              length={state.length}
              value={value}
              onChange={(v) => {
                setValue(v);
                if (v) setError(null);
              }}
              onComplete={(v) => void submit(v)}
              disabled={checking || blocked}
              error={blocked ? t('blocked', { seconds: formatNumber(secondsLeft, loc) }) : error}
              extraKey={
                state.biometric ? (
                  <button type="button" className="lk-key is-ghost" onClick={() => void biometric()} aria-label={t('biometric')}>
                    <Icon name="hand" size={24} />
                  </button>
                ) : null
              }
            />
            <button type="button" className="lk-link" onClick={() => setForgot(true)}>
              {t('forgot')}
            </button>
            {emergencyOn ? (
              <button type="button" className="lk-ec-link" onClick={() => setShowCard(true)}>
                <Icon name="card" size={18} />
                <span className="lk-ec-link-text">
                  <span className="lk-ec-link-title">{t('emergency')}</span>
                  <span className="lk-ec-link-hint">{t('emergencyHint')}</span>
                </span>
              </button>
            ) : null}
          </>
        )}
      </div>
    </div>
  );
}


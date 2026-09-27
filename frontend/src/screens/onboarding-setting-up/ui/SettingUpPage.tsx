'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useCallback, useEffect, useRef, useState } from 'react';

import { useActivatePregnancy, useCompleteOnboarding } from '@/entities/pregnancy';
import { useOnboardingStore } from '@/entities/user';
import { onboardingToProfileInput, useUpdateProfile } from '@/features/edit-profile';
import { type Locale, useRouter } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib/date';
import { clearOnboardingPending, getAuthToken, setAuthToken } from '@/shared/session';

import { planSetup, runSetup, type SetupPlan, type SetupStep } from '../model/plan';

const CIRCUMFERENCE = 553;

type SavePlan = Extract<SetupPlan, { kind: 'save' }>;

export function SettingUpPage() {
  const t = useTranslations('onboarding.settingUp');
  // Generic failure copy from namespaces this route already ships (no new keys).
  const te = useTranslations('profileEdit.errors');
  const tc = useTranslations('common.actions');
  const locale = useLocale() as Locale;
  const router = useRouter();
  const update = useUpdateProfile();
  const activate = useActivatePregnancy();
  const completeOnboarding = useCompleteOnboarding();

  const [pct, setPct] = useState(0);
  const [ringDone, setRingDone] = useState(false);
  const [landing, setLanding] = useState<SavePlan['landing'] | null>(null);
  const [saveDone, setSaveDone] = useState(false);
  const [failed, setFailed] = useState(false);
  const ringRef = useRef<SVGCircleElement>(null);
  const startedRef = useRef(false);
  const planRef = useRef<SavePlan | null>(null);
  const doneRef = useRef(new Set<SetupStep>());
  const runningRef = useRef(false);

  // Send the plan's requests; a failure shows the retry UI instead of moving
  // on with a half-saved account, and a retry resumes at the failed step.
  const save = useCallback(async () => {
    const plan = planRef.current;
    if (!plan || runningRef.current) return;
    runningRef.current = true;
    setFailed(false);
    const answers = useOnboardingStore.getState();
    try {
      await runSetup(
        plan.steps,
        (step) => {
          if (step === 'profile') return update.mutateAsync(onboardingToProfileInput(answers));
          if (step === 'activate') return activate.mutateAsync();
          if (!plan.pregnancy) throw new Error('pregnancy onboarding body missing');
          return completeOnboarding.mutateAsync(plan.pregnancy);
        },
        doneRef.current,
      );
      setSaveDone(true);
    } catch {
      setFailed(true);
    } finally {
      runningRef.current = false;
    }
  }, [update, activate, completeOnboarding]);

  // Decide once, from the *hydrated* store: missing answers send the user back
  // to the step that asks them (never a silent cycle-profile save); otherwise
  // persist. The ref keeps StrictMode's double-invoke from firing two POSTs.
  useEffect(() => {
    if (startedRef.current) return;
    startedRef.current = true;
    const start = () => {
      const plan = planSetup(useOnboardingStore.getState());
      if (plan.kind === 'resume') {
        router.replace(plan.route);
        return;
      }
      planRef.current = plan;
      setLanding(plan.landing);
      void save();
    };
    const persist = useOnboardingStore.persist;
    if (persist.hasHydrated()) start();
    else persist.onFinishHydration(() => start());
    // Run once on mount.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // The progress ring is a reassurance animation, independent of the request.
  useEffect(() => {
    let p = 0;
    const id = setInterval(() => {
      p += 2;
      if (p > 100) {
        clearInterval(id);
        setRingDone(true);
        return;
      }
      setPct(p);
      if (ringRef.current) {
        ringRef.current.style.strokeDashoffset = String(CIRCUMFERENCE * (1 - p / 100));
      }
    }, 45);
    return () => clearInterval(id);
  }, []);

  // Leave only once the ring has finished AND the save has settled, so the
  // profile is written before the next screen refetches it. Pregnant users land
  // in pregnancy mode; everyone else on home.
  //
  // This last hop is a **full document navigation**, not `router.replace`: the
  // target is behind the auth middleware, and a client-side transition can be
  // answered from the App Router's cache — including a redirect-to-signup
  // cached from before the session cookie existed — which dumped freshly
  // registered users back on the phone-number screen even though reloading
  // took them straight in. The auth cookie is also re-asserted first, so the
  // request the middleware sees always carries it.
  useEffect(() => {
    if (!ringDone || !saveDone || !landing) return;
    // Registration is done — drop the resume marker, or the middleware would
    // keep herding this session back into the flow it just finished.
    clearOnboardingPending();
    const token = getAuthToken();
    if (token) setAuthToken(token);
    window.location.replace(`/${locale}${landing}`);
  }, [ringDone, saveDone, landing, locale]);

  return (
    <div className="view onb-page">
      <div className="scroll setup-body">
        <div className="titr setup-titr">{t('title')}</div>
        <p className="sub setup-sub">{t('subtitle')}</p>

        <div className="setup-ring">
          <svg width="200" height="200" viewBox="0 0 200 200">
            <circle cx="100" cy="100" r="88" fill="none" stroke="var(--field-border)" strokeWidth="13" />
            <circle
              ref={ringRef}
              cx="100" cy="100" r="88" fill="none"
              stroke="var(--pink)" strokeWidth="13" strokeLinecap="round"
              strokeDasharray={CIRCUMFERENCE} strokeDashoffset={CIRCUMFERENCE}
            />
          </svg>
          <div className="setup-ring-pct">
            {formatNumber(pct, locale)}٪
          </div>
        </div>

        <p className="sub setup-note">
          {t('disclaimer')}
        </p>
      </div>
      <div className="setup-footer">
        {failed ? (
          <div className="flex flex-col items-center gap-3" role="alert">
            <span className="text-[13px] leading-[1.7] font-semibold text-(--danger-deep)">{te('generic')}</span>
            <button type="button" className="btn btn-primary" onClick={() => void save()}>
              {tc('save')}
            </button>
          </div>
        ) : (
          <span className="sub">{t('wait')}</span>
        )}
      </div>
    </div>
  );
}

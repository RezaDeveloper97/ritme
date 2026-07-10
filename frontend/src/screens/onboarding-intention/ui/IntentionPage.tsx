'use client';

import { useTranslations } from 'next-intl';

import { useRouter } from '@/shared/i18n';
import { Icon, NavBack } from '@/shared/ui';
import {
  nextOnboardingRoute,
  stepPosition,
  useOnboardingStore,
  type PregnancyIntention,
} from '@/entities/user';

const FA = ['۰','۱','۲','۳','۴','۵','۶','۷','۸','۹'];
const faNum = (n: string | number) => String(n).replace(/[0-9]/g, d => FA[Number(d)]);

const OPTIONS: PregnancyIntention[] = ['avoiding', 'pregnant', 'trying', 'unsure'];

/**
 * The branch point of onboarding: how the user relates to pregnancy. Choosing
 * "pregnant" routes to the dating-basis step (period tracking off); every other
 * answer continues to the cycle questions (see the step-sequence driver).
 */
export function IntentionPage() {
  const t = useTranslations('onboarding');
  const router = useRouter();
  const { intention, setIntention } = useOnboardingStore();
  const step = stepPosition('intention', intention);

  const select = (choice: PregnancyIntention) => {
    setIntention(choice);
    // Route on the freshly chosen value — the store update isn't observable yet.
    router.push(nextOnboardingRoute('intention', choice));
  };

  return (
    <div className="view" style={{ background: '#fff' }}>
      <div className="hdr">
        <NavBack onClick={() => router.back()} />
        <span className="stepcount">{faNum(step.index)}<span style={{ opacity: .5 }}> / {faNum(step.total)}</span></span>
      </div>

      <div className="scroll" style={{ padding: '8px 22px 0', display: 'flex', flexDirection: 'column' }}>
        <div style={{ textAlign: 'start', margin: '6px 0' }}>
          <div className="titr">{t('intention.title')}</div>
          <p className="sub" style={{ margin: '10px 0 0' }}>{t('intention.subtitle')}</p>
        </div>

        <div style={{ display: 'flex', flexDirection: 'column', gap: 12, marginTop: 18 }}>
          {OPTIONS.map((opt) => {
            const on = intention === opt;
            return (
              <button
                key={opt}
                type="button"
                onClick={() => select(opt)}
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: 12,
                  padding: '15px 16px',
                  borderRadius: 16,
                  textAlign: 'start',
                  cursor: 'pointer',
                  border: `2px solid ${on ? 'var(--ritme-pink, #FB64B6)' : '#EBEEF2'}`,
                  background: on ? '#FFF0F7' : '#fff',
                  transition: '.18s',
                }}
              >
                <span
                  style={{
                    flex: '0 0 auto',
                    width: 22,
                    height: 22,
                    borderRadius: '50%',
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    background: on ? 'var(--ritme-pink, #FB64B6)' : '#EBEEF2',
                    color: '#fff',
                  }}
                >
                  {on && <Icon name="check" size={14} />}
                </span>
                <span style={{ fontSize: 15, fontWeight: 700, color: on ? 'var(--ritme-pink, #FB64B6)' : 'var(--ink)' }}>
                  {t(`intention.options.${opt}`)}
                </span>
              </button>
            );
          })}
        </div>
      </div>
    </div>
  );
}

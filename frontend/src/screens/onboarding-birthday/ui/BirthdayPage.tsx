'use client';

import { useTranslations } from 'next-intl';

import { useRouter } from '@/shared/i18n';
import { NavBack, WheelPicker } from '@/shared/ui';
import { nextOnboardingRoute, stepPosition, useOnboardingStore } from '@/entities/user';

const FA = ['۰','۱','۲','۳','۴','۵','۶','۷','۸','۹'];
const faNum = (n: string | number) => String(n).replace(/[0-9]/g, d => FA[Number(d)]);

const FA_MONTHS = ['فروردین','اردیبهشت','خرداد','تیر','مرداد','شهریور','مهر','آبان','آذر','دی','بهمن','اسفند'];
const DAYS   = Array.from({ length: 31 }, (_, i) => faNum(i + 1));
const YEARS  = Array.from({ length: 60 }, (_, i) => faNum(1340 + i));

export function BirthdayPage() {
  const t = useTranslations('onboarding');
  const router = useRouter();
  const { birth, intention, setBirth } = useOnboardingStore();
  const step = stepPosition('birthday', intention);

  return (
    <div className="view" style={{ background: 'var(--surface)' }}>
      <div className="hdr">
        <NavBack onClick={() => router.back()} />
        <span className="stepcount">{faNum(step.index)}<span style={{ opacity: .5 }}> / {faNum(step.total)}</span></span>
      </div>

      <div className="scroll" style={{ padding: '8px 22px 0', display: 'flex', flexDirection: 'column' }}>
        <div style={{ textAlign: 'start', margin: '6px 0' }}>
          <div className="titr">{t('birthday.title')}</div>
          <p className="sub" style={{ margin: '10px 0 0' }}>{t('birthday.subtitle')}</p>
        </div>
        <div style={{ flex: 1, display: 'flex', flexDirection: 'column', justifyContent: 'center', padding: '8px 0' }}>
          <div style={{ display: 'flex', gap: 8, justifyContent: 'center', marginTop: 4, position: 'relative' }}>
            <div className="wheel-band" />
            <WheelPicker
              id="wD" items={DAYS} selectedIndex={birth.d - 1} width={56}
              onChange={i => setBirth({ ...birth, d: i + 1 })}
            />
            <WheelPicker
              id="wM" items={FA_MONTHS} selectedIndex={birth.m} width={112}
              onChange={i => setBirth({ ...birth, m: i })}
            />
            <WheelPicker
              id="wY" items={YEARS} selectedIndex={birth.y - 1340} width={80}
              onChange={i => setBirth({ ...birth, y: 1340 + i })}
            />
          </div>
        </div>
      </div>

      <div style={{ padding: '14px 16px 8px' }}>
        <button className="btn btn-primary" onClick={() => router.push(nextOnboardingRoute('birthday', intention))}>
          {t('continue')}
        </button>
      </div>
    </div>
  );
}

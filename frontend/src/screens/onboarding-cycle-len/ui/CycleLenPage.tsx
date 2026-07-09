'use client';

import { useTranslations } from 'next-intl';

import { useRouter } from '@/shared/i18n';
import { JalaliCalendar, NavBack } from '@/shared/ui';
import { useOnboardingStore } from '@/entities/user';

const FA = ['۰','۱','۲','۳','۴','۵','۶','۷','۸','۹'];
const faNum = (n: string | number) => String(n).replace(/[0-9]/g, d => FA[Number(d)]);

export function CycleLenPage() {
  const t = useTranslations('onboarding');
  const router = useRouter();
  const { lastPeriod, setLastPeriod } = useOnboardingStore();

  return (
    <div className="view" style={{ background: '#fff' }}>
      <div className="hdr">
        <NavBack onClick={() => router.back()} />
        <span className="stepcount">{faNum(8)}<span style={{ opacity: .5 }}> / ۸</span></span>
      </div>

      <div className="scroll" style={{ padding: '8px 22px 0', display: 'flex', flexDirection: 'column' }}>
        <div style={{ textAlign: 'start', margin: '6px 0' }}>
          <div className="titr">{t('cycleLen.title')}</div>
          <p className="sub" style={{ margin: '10px 0 0' }}>{t('cycleLen.subtitle')}</p>
        </div>
        <div style={{ flex: 1, display: 'flex', flexDirection: 'column', justifyContent: 'center', padding: '8px 0' }}>
          <JalaliCalendar value={lastPeriod} onSelect={setLastPeriod} />
          <p className="sub" style={{ textAlign: 'center', marginTop: 16 }}>{t('cycleLen.hint')}</p>
        </div>
      </div>

      <div style={{ padding: '14px 16px 8px' }}>
        <button className="btn btn-primary" onClick={() => router.push('/onboarding/setting-up')}>
          {t('finish')}
        </button>
      </div>
    </div>
  );
}

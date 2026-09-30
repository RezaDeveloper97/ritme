'use client';

import { useTranslations } from 'next-intl';

import { Icon } from '@/shared/ui';

/** `v19_Main` «امروز تست LH بزن» — shown by the home inside the fertile window while no LH test is logged today. */
export function LhTipCard() {
  const t = useTranslations('fertility.home.lhTip');
  return (
    <section className="ttc-tip">
      <span className="ttc-tip-disc" aria-hidden>
        <Icon name="sparkle" size={20} fill="currentColor" strokeWidth={0} />
      </span>
      <div className="ttc-tip-card-text">
        <h2 className="ttc-tip-card-title">{t('title')}</h2>
        <p className="ttc-tip-card-body">{t('body')}</p>
      </div>
    </section>
  );
}

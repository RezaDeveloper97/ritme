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
      <div className="flex min-w-0 flex-col gap-1 text-start">
        <h2 className="text-[14.5px] font-extrabold text-(--ink)">{t('title')}</h2>
        <p className="text-[13px] leading-[1.9] text-(--ink-3)">{t('body')}</p>
      </div>
    </section>
  );
}

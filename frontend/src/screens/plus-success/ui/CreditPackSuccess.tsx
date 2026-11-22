'use client';

import { useLocale, useTranslations } from 'next-intl';

import { formatCredits, useAiCredits } from '@/entities/ai-credit';
import { useRouter, type Locale } from '@/shared/i18n';
import { Art3d, PrimaryButton, SecondaryButton, SkyLayer } from '@/shared/ui';
import '@/shared/ui/styles/plus.css';

/** «اعتبار اضافه شد» — the gateway return of a credit pack (`invoice.kind === 'credit_pack'`). */
export function CreditPackSuccess({ credits, refId }: { credits: number; refId?: string | null }) {
  const t = useTranslations('credits.success');
  const tp = useTranslations('plus');
  const router = useRouter();
  const loc = useLocale() as Locale;
  const balance = useAiCredits();
  const locked = balance.data ? !balance.data.canUse : false;
  const count = formatCredits(credits, loc);
  return (
    <div className="view plus-page">
      <SkyLayer />
      <div className="scroll plus-scroll is-success">
        <div className="plus-hero is-success">
          <Art3d name="credits" size={120} />
          <h1 className="plus-display is-md">{t('title')}</h1>
          <p className="plus-lead">{locked ? t('bodyLocked', { count }) : t('body', { count })}</p>
          {refId ? (
            <p className="plus-ref">
              {tp('success.refLabel')} <bdi dir="ltr">{refId}</bdi>
            </p>
          ) : null}
        </div>
      </div>
      <div className="plus-footer">
        <PrimaryButton onClick={() => router.replace('/home')}>{t('home')}</PrimaryButton>
        <SecondaryButton variant="text" onClick={() => router.replace('/profile/credits')}>
          {t('usage')}
        </SecondaryButton>
      </div>
    </div>
  );
}

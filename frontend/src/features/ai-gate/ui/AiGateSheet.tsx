'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useEffect } from 'react';

import { type AiGateDenial, formatCredits } from '@/entities/ai-credit';
import { aiDenialOf } from '@/entities/ai-credit';
import { setApiFailureListener } from '@/shared/api';
import { type Locale, useRouter } from '@/shared/i18n';
import { formatLongDate, fromApiDate } from '@/shared/lib/date';
import { AppSheet } from '@/shared/sheet';
import { Art3d, IconCircle, PrimaryButton, SecondaryButton } from '@/shared/ui';

import { openAiGate, useAiGateStore } from '../model/store';
import './aigate.css';

/** Where the sheet's actions lead (explicit routes — never history, CLAUDE.md §4.2). */
export const CREDITS_BUY_ROUTE = '/plus/credits';

/**
 * The one global AI gate sheet: «اشتراک لازم است» / «اعتبار تمام شد» / «در دسترس نیست». A short decision, so a
 * sheet; the purchase itself is a route (CLAUDE.md §4.1). Mount once in the app shell beside the other global
 * sheets. It also registers the safety net: a 402 any surface forgot to handle opens it from here.
 */
export function AiGateSheet() {
  const denial = useAiGateStore((s) => s.denial);
  const close = useAiGateStore((s) => s.close);

  useEffect(
    () =>
      setApiFailureListener((error) => {
        const d = aiDenialOf(error);
        if (d && (d.kind === 'subscription' || d.kind === 'credits')) openAiGate(d);
      }),
    [],
  );

  return (
    <AppSheet open={denial !== null} onClose={close} size="half">
      {denial ? <GateBody denial={denial} onClose={close} /> : null}
    </AppSheet>
  );
}

function GateBody({ denial, onClose }: { denial: AiGateDenial; onClose: () => void }) {
  const t = useTranslations('aiGate');
  const locale = useLocale() as Locale;
  const router = useRouter();
  const go = (href: string) => {
    onClose();
    router.push(href);
  };

  if (denial.kind === 'disabled') {
    return (
      <div className="aigate">
        <IconCircle icon="clock" tone="neutral" size="lg" />
        <h2 className="aigate-title">{t('disabled.title')}</h2>
        <p className="aigate-body">{t('disabled.body')}</p>
        <PrimaryButton onClick={onClose}>{t('disabled.close')}</PrimaryButton>
      </div>
    );
  }

  if (denial.kind === 'subscription') {
    return (
      <div className="aigate">
        <Art3d name="credits" size={88} />
        <h2 className="aigate-title">{t('subscription.title')}</h2>
        <p className="aigate-body">{t('subscription.body')}</p>
        {denial.extraCredits > 0 ? (
          <p className="aigate-kept">{t('subscription.kept', { count: formatCredits(denial.extraCredits, locale) })}</p>
        ) : null}
        <PrimaryButton onClick={() => go('/plus')}>{t('subscription.cta')}</PrimaryButton>
        <SecondaryButton variant="text" onClick={onClose}>
          {t('later')}
        </SecondaryButton>
      </div>
    );
  }

  const renews = denial.periodEndsAt ? formatLongDate(fromApiDate(denial.periodEndsAt.slice(0, 10)), locale) : null;
  return (
    <div className="aigate">
      <Art3d name="credits" size={88} />
      <h2 className="aigate-title">{t('credits.title')}</h2>
      <p className="aigate-body">
        {t('credits.need', { required: formatCredits(denial.required, locale), balance: formatCredits(denial.balance, locale) })}
      </p>
      {renews ? <p className="aigate-kept">{t('credits.renews', { date: renews })}</p> : null}
      <PrimaryButton onClick={() => go(CREDITS_BUY_ROUTE)}>{t('credits.cta')}</PrimaryButton>
      <SecondaryButton variant="text" onClick={() => go('/plus/plans')}>
        {t('credits.plans')}
      </SecondaryButton>
    </div>
  );
}

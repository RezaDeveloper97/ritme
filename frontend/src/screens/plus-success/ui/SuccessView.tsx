'use client';

import { useLocale, useTranslations } from 'next-intl';
import type { ReactNode } from 'react';

import { PlusCrown, type PlusStatus } from '@/entities/plus';
import { useRouter, type Locale } from '@/shared/i18n';
import { formatLongDate } from '@/shared/lib/date';
import { ListGroup, ListRow, PrimaryButton, SkyLayer, StatusPill } from '@/shared/ui';

interface SuccessViewProps {
  status: PlusStatus;
  /** The bank reference of the payment just verified, when there is one. */
  refId?: string | null;
}

/**
 * «به ریتمی پلاس خوش آمدی» (`nbl_Prem_Success` / `nbd_Prem_Success`): crown,
 * the plan and its end date, the bank reference, «از اینجا شروع کن» and
 * «بزن بریم». Rows whose screens ship later (doctor report B-N6, assistant
 * B-N7) say «به‌زودی» and are not buttons.
 */
export function SuccessView({ status, refId }: SuccessViewProps) {
  const t = useTranslations('plus');
  const router = useRouter();
  const loc = useLocale() as Locale;
  const sub = status.subscription;
  const trial = status.trial?.isActive ? status.trial : null;
  const endsAt = sub?.endsAt ?? trial?.endsAt ?? null;
  const date = endsAt ? formatLongDate(new Date(endsAt), loc) : '';
  const bold = (chunks: ReactNode) => <b className="plus-strong">{chunks}</b>;
  const soon = <StatusPill tone="neutral">{t('success.soon')}</StatusPill>;

  return (
    <div className="view plus-page">
      <SkyLayer />
      <div className="scroll plus-scroll is-success">
        <div className="plus-hero is-success">
          <PlusCrown size="lg" />
          <h1 className="plus-display is-md">{t('success.title')}</h1>
          {endsAt ? (
            <p className="plus-lead">
              {sub
                ? t.rich('success.body', { plan: sub.plan?.title ?? '', date, b: bold })
                : t.rich('success.trialBody', { date, b: bold })}
            </p>
          ) : null}
          {refId ? (
            <p className="plus-ref">
              {t('success.refLabel')} <bdi dir="ltr">{refId}</bdi>
            </p>
          ) : null}
        </div>

        <ListGroup title={t('success.startHere')} className="plus-start">
          <ListRow icon="chart" iconTone="brand" title={t('success.rows.analysis')} onClick={() => router.push('/cycle')} />
          <ListRow icon="note" iconTone="data" title={t('success.rows.report')} trailing={soon} />
          <ListRow icon="sparkle" iconTone="bloom" title={t('success.rows.assistant')} trailing={soon} />
        </ListGroup>
      </div>
      <div className="plus-footer">
        <PrimaryButton onClick={() => router.replace('/home')}>{t('success.cta')}</PrimaryButton>
      </div>
    </div>
  );
}

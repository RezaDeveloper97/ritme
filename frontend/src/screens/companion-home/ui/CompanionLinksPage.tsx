'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useState } from 'react';

import { useCompanionLinks, useLeaveCompanionLink, type ViewerLink } from '@/entities/companion';
import { InviteAppLink, JoinCodeStep, JoinConfirmStep, useJoinCompanion } from '@/features/join-companion';
import { type Locale, useRouter } from '@/shared/i18n';
import { formatDayMonth, fromApiDate } from '@/shared/lib/date';
import { AppSheet } from '@/shared/sheet';
import {
  Avatar,
  PrimaryButton,
  ScreenHeader,
  SecondaryButton,
  SectionTitle,
  Skeleton,
  SkeletonGroup,
  SkyLayer,
} from '@/shared/ui';

/**
 * `/companion/links` — «افزودن همراه» (companion v2 `join` → `joinConfirm`): enter a code, see whose
 * cycle you are joining and the boundaries, then «شروع همراهی» accepts it and opens `/companion`. Below the
 * code step: who you are linked to, and leaving a link (`DELETE /companions/links/{id}`, confirmed in a sheet).
 */
export function CompanionLinksPage() {
  const t = useTranslations('companionHome');
  const locale = useLocale() as Locale;
  const router = useRouter();
  const links = useCompanionLinks();
  const leave = useLeaveCompanionLink();
  const [confirm, setConfirm] = useState<ViewerLink | null>(null);
  const someone = t('someone');
  const join = useJoinCompanion(() => router.replace('/companion'));
  const confirming = join.step === 'confirm';

  const doLeave = () => {
    if (!confirm || leave.isPending) return;
    leave.mutate(confirm.id, { onSuccess: () => setConfirm(null) });
  };

  return (
    <div className="view cmh-page">
      <SkyLayer />
      <ScreenHeader
        title={t('links.title')}
        onBack={() => (confirming ? join.backToCode() : router.replace('/companion'))}
        backLabel={t('links.back')}
      />
      <div className="scroll cmh-scroll">
        <div className="cmh-body">
          {confirming ? (
            <>
              <JoinConfirmStep join={join} />
              <PrimaryButton onClick={join.confirm} loading={join.accepting}>
                {t('join.confirm.start')}
              </PrimaryButton>
            </>
          ) : (
            <>
              <JoinCodeStep join={join} />
              <PrimaryButton onClick={join.submitCode} disabled={!join.complete} loading={join.checking}>
                {join.checking ? t('code.connecting') : t('code.connect')}
              </PrimaryButton>
              <InviteAppLink />
            </>
          )}

          {confirming ? null : (
          <section className="cmh-sec" aria-labelledby="cmh-links-title">
            <SectionTitle id="cmh-links-title" title={t('links.linkedTitle')} />
            {links.isPending ? (
              <SkeletonGroup label={t('loading')} className="nb-card cmh-links">
                <Skeleton width="medium" />
                <Skeleton width="short" />
              </SkeletonGroup>
            ) : links.isError ? (
              <div className="nb-card cmh-links">
                <p className="cmh-shared-note" role="alert">
                  {t('links.loadError')}
                </p>
                <SecondaryButton onClick={() => void links.refetch()}>{t('links.retry')}</SecondaryButton>
              </div>
            ) : !links.data?.length ? (
              <div className="nb-card cmh-links">
                <p className="cmh-shared-note">{t('links.none')}</p>
              </div>
            ) : (
              <ul className="nb-card cmh-links">
                {links.data.map((l) => {
                  const name = l.owner.name ?? someone;
                  return (
                    <li key={l.id} className="cmh-link-row">
                      <Avatar name={name} size="md" />
                      <span className="cmh-shared-text">
                        <b className="cmh-shared-title">{name}</b>
                        {l.acceptedAt ? (
                          <span className="cmh-shared-sub">
                            {t('links.since', { date: formatDayMonth(fromApiDate(l.acceptedAt.slice(0, 10)), locale) })}
                          </span>
                        ) : null}
                      </span>
                      <button
                        type="button"
                        className="cmh-leave"
                        aria-label={t('links.leaveLabel', { name })}
                        onClick={() => {
                          leave.reset();
                          setConfirm(l);
                        }}
                      >
                        {t('links.leave')}
                      </button>
                    </li>
                  );
                })}
              </ul>
            )}
            {links.data?.length ? (
              <SecondaryButton variant="text" onClick={() => router.push('/companion')}>
                {t('links.toHome')}
              </SecondaryButton>
            ) : null}
          </section>
          )}
        </div>
      </div>

      <AppSheet
        open={confirm != null}
        onClose={() => setConfirm(null)}
        size="half"
        title={t('links.leaveConfirmTitle', { name: confirm?.owner.name ?? someone })}
        footer={
          <div className="cmh-confirm-actions">
            <PrimaryButton onClick={doLeave} loading={leave.isPending}>
              {t('links.leaveConfirm')}
            </PrimaryButton>
            <SecondaryButton variant="text" onClick={() => setConfirm(null)}>
              {t('links.cancel')}
            </SecondaryButton>
          </div>
        }
      >
        <p className="cmh-confirm-body">{t('links.leaveConfirmBody', { name: confirm?.owner.name ?? someone })}</p>
        {leave.isError ? (
          <p className="onb2-error" role="alert">
            {t('links.leaveError')}
          </p>
        ) : null}
      </AppSheet>
    </div>
  );
}

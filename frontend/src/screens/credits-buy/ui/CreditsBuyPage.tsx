'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useEffect, useState } from 'react';

import { defaultPack, formatCredits, useAiCredits, useCreditPacks } from '@/entities/ai-credit';
import { formatToman } from '@/entities/plus';
import { useBillingChannel } from '@/shared/billing';
import { useRouter, type Locale } from '@/shared/i18n';
import {
  EmptyState,
  InfoNote,
  PrimaryButton,
  RadioCardGroup,
  ScreenHeader,
  SecondaryButton,
  Skeleton,
  SkeletonGroup,
  SkyLayer,
  StatusPill,
} from '@/shared/ui';
import { useNavMode } from '@/widgets/bottom-nav';

/**
 * `/plus/credits` «خرید اعتبار»: the credit packs as choice cards (credits, price in toman, badge, the
 * highlighted one pre-selected) → the existing checkout (`/plus/checkout?pack=<id>`). Not offered to teen /
 * companion accounts nor in store builds (the web gateway is not their sales channel).
 */
export function CreditsBuyPage() {
  const t = useTranslations('credits');
  const loc = useLocale() as Locale;
  const router = useRouter();
  const navMode = useNavMode().mode;
  const { channel, ready } = useBillingChannel();
  const packs = useCreditPacks();
  const credits = useAiCredits();
  const [picked, setPicked] = useState<string | null>(null);
  const chosen = picked ?? (packs.data ? String(defaultPack(packs.data.packs)?.id ?? '') : '');
  const pack = packs.data?.packs.find((p) => String(p.id) === chosen) ?? null;
  const blocked = navMode === 'teen' || navMode === 'companion';

  useEffect(() => {
    if (blocked) router.replace('/profile');
  }, [blocked, router]);

  let body;
  if (packs.isPending || !ready) {
    body = (
      <SkeletonGroup label={t('loading')} className="cr-body">
        <Skeleton shape="card" />
        <Skeleton shape="card" />
        <Skeleton shape="card" />
      </SkeletonGroup>
    );
  } else if (channel !== 'web') {
    body = <EmptyState icon="sparkle" title={t('buy.storeTitle')} body={t('buy.storeBody')} />;
  } else if (packs.isError) {
    body = (
      <EmptyState
        icon="sparkle"
        title={t('error.title')}
        body={t('error.body')}
        action={
          <PrimaryButton icon="refresh" loading={packs.isFetching} onClick={() => void packs.refetch()}>
            {t('retry')}
          </PrimaryButton>
        }
      />
    );
  } else if (packs.data.packs.length === 0) {
    body = <EmptyState icon="sparkle" title={t('buy.empty')} />;
  } else {
    body = (
      <div className="cr-body">
        <p className="cr-lead">{t('buy.lead')}</p>
        <RadioCardGroup
          label={t('buy.groupLabel')}
          value={chosen}
          onChange={setPicked}
          options={packs.data.packs.map((p) => ({
            value: String(p.id),
            art: 'credits' as const,
            title: (
              <>
                {t('buy.credits', { count: formatCredits(p.credits, loc) })}
                {p.badge ? (
                  <>
                    {' '}
                    <StatusPill tone="warm">{p.badge}</StatusPill>
                  </>
                ) : null}
              </>
            ),
            description: t('buy.price', { price: formatToman(p.price, loc) }),
          }))}
        />
        {credits.data && !credits.data.canUse ? (
          <InfoNote>
            {t('buy.noSubscription')}{' '}
            <SecondaryButton block={false} variant="text" onClick={() => router.push('/plus')}>
              {t('buy.noSubscriptionCta')}
            </SecondaryButton>
          </InfoNote>
        ) : null}
      </div>
    );
  }

  return (
    <div className="view cr-page">
      <SkyLayer />
      <div className="scroll cr-scroll">
        <ScreenHeader title={t('buy.title')} onBack={() => router.push('/profile/credits')} backLabel={t('back')} />
        {body}
      </div>
      {pack && channel === 'web' ? (
        <div className="plus-footer">
          <PrimaryButton onClick={() => router.push(`/plus/checkout?pack=${pack.id}`)}>
            {t('buy.cta', { pack: t('buy.credits', { count: formatCredits(pack.credits, loc) }) })}
          </PrimaryButton>
        </div>
      ) : null}
    </div>
  );
}

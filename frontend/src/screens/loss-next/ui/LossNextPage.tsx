'use client';

import { useQueryClient } from '@tanstack/react-query';
import { useLocale, useTranslations } from 'next-intl';
import { useState } from 'react';

import {
  LOSS_NEXT_STEPS,
  type LossNextStep,
  type LossState,
  lifeModeOfNextStep,
  useLossCatalog,
  useLossState,
  useSaveLossNextStep,
} from '@/entities/loss';
import { writeLifeModeHint } from '@/entities/user';
import { getApiSaveErrorMessage } from '@/shared/api';
import { type Locale, useRouter } from '@/shared/i18n';
import {
  EmptyState,
  type IconName,
  InfoNote,
  PrimaryButton,
  RadioCardGroup,
  type RadioCardOption,
  ScreenHeader,
  Skeleton,
  SkeletonGroup,
  SkyLayer,
  type Tone,
} from '@/shared/ui';

const LOOK: Record<LossNextStep, { icon: IconName; tone: Tone }> = {
  cycle: { icon: 'drop', tone: 'period' },
  ttc: { icon: 'sprout', tone: 'data' },
  nothing: { icon: 'moon', tone: 'brand' },
};

/**
 * «از اینجا به بعد» (`/loss/next`, CB-LOSS-02, nbl_Loss_Next): three calm
 * options — just the cycle, trying again, or nothing for now. «ذخیره» sends
 * `PUT /loss/next-step` (which switches the life mode server-side: ttc → TTC,
 * otherwise cycle) and opens the matching home. The «سقط دوم یا سوم» note
 * shows only from the second recorded loss (`recurrent_hint`).
 */
export function LossNextPage() {
  const t = useTranslations('loss.common');
  const tn = useTranslations('loss.next');
  const router = useRouter();
  const state = useLossState();

  let body;
  if (state.isPending) {
    body = (
      <SkeletonGroup label={t('loading')} className="lnx-skel">
        <Skeleton shape="block" className="lnx-skel-card" />
        <Skeleton shape="block" className="lnx-skel-card" />
        <Skeleton shape="block" className="lnx-skel-card" />
      </SkeletonGroup>
    );
  } else if (state.isError) {
    body = (
      <EmptyState
        icon="heart"
        title={t('loadError')}
        action={
          <PrimaryButton icon="refresh" loading={state.isFetching} onClick={() => void state.refetch()}>
            {t('retry')}
          </PrimaryButton>
        }
      />
    );
  } else if (!state.data.loss) {
    body = (
      <EmptyState
        icon="heart"
        title={t('noLoss.title')}
        body={t('noLoss.body')}
        action={<PrimaryButton onClick={() => router.replace('/loss')}>{t('noLoss.action')}</PrimaryButton>}
      />
    );
  } else {
    body = <NextForm state={state.data} />;
  }

  return (
    <div className="view lnx-page">
      <SkyLayer />
      <div className="scroll lnx-scroll">
        <ScreenHeader title={tn('screenTitle')} onBack={() => router.push('/loss/care')} backLabel={t('back')} />
        {body}
      </div>
    </div>
  );
}

function NextForm({ state }: { state: LossState }) {
  const t = useTranslations('loss.next');
  const tc = useTranslations('loss.common');
  const locale = useLocale() as Locale;
  const router = useRouter();
  const queryClient = useQueryClient();
  const steps = useLossCatalog('loss_next_steps', locale);
  const support = useLossCatalog('loss_support', locale);
  const save = useSaveLossNextStep();
  // The gentlest option is preselected (as on the board); her earlier answer wins.
  const [choice, setChoice] = useState<LossNextStep>(state.loss?.nextStep ?? 'cycle');

  const item = (code: LossNextStep) => steps.data?.find((i) => i.code === code) ?? null;
  const options: RadioCardOption<LossNextStep>[] = LOSS_NEXT_STEPS.map((code) => ({
    value: code,
    title: item(code)?.title ?? t(`steps.${code}.title`),
    description: item(code)?.body ?? t(`steps.${code}.body`),
    icon: LOOK[code].icon,
    iconTone: LOOK[code].tone,
  }));
  const recurrent = support.data?.find((i) => i.code === 'recurrent_hint');

  const submit = () => {
    if (save.isPending) return;
    save.mutate(choice, {
      onSuccess: () => {
        // The mode reshapes the whole app (nav, home, messages): refetch everything.
        writeLifeModeHint(lifeModeOfNextStep(choice, item(choice)));
        void queryClient.invalidateQueries();
        router.replace('/home');
      },
    });
  };

  return (
    <>
      <div className="lnx-intro">
        <h1 className="lnx-title">{t('title')}</h1>
        <p className="lnx-lead">{t('lead')}</p>
      </div>

      <RadioCardGroup options={options} value={choice} onChange={setChoice} label={t('title')} />

      {state.recurrentHint ? <InfoNote className="lnx-note">{recurrent?.body ?? t('recurrentFallback')}</InfoNote> : null}

      <div className="lnx-footer">
        {save.isError ? (
          <p className="lnx-error" role="alert">
            {getApiSaveErrorMessage(save.error, tc('saveError'))}
          </p>
        ) : null}
        <PrimaryButton loading={save.isPending} onClick={submit}>
          {t('save')}
        </PrimaryButton>
      </div>
    </>
  );
}

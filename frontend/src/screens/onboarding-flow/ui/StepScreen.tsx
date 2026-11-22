'use client';

import { useTranslations } from 'next-intl';
import { type ReactNode, useEffect } from 'react';

import { ALL_MODULES_ON, type AppModules, useAppModulesQuery } from '@/entities/app-module';

import { useRouter } from '@/shared/i18n';
import { EmptyState, PrimaryButton, Skeleton, SkeletonGroup } from '@/shared/ui';
import { OnbFrame } from '@/features/auth';

import { useOnboardingState } from '../api/onboarding';
import { FLOW_ROUTES, type FlowStep, isGoal, nextStep, type OnboardingGoal, previousRoute, PROGRESS, PROGRESS_TOTAL } from '../model/flow';
import type { OnboardingState } from '../model/state';
import '@/shared/ui/styles/onb.css';

export interface StepContext {
  state: OnboardingState;
  goal: OnboardingGoal | null;
  /** Admin module switches: an off module's choices are hidden (companion off = no gender question). */
  modules: AppModules;
  progress: { current: number; total: number } | undefined;
  back: () => void;
  /** Go to the screen after `step`; `override` replaces the stored answers (just saved). */
  next: (override?: Partial<Pick<OnboardingState, 'gender' | 'goal'>>) => void;
}

interface StepScreenProps {
  step: FlowStep;
  children: (ctx: StepContext) => ReactNode;
}

/**
 * Loads the answers so far (`GET /onboarding`) for one step and hands the
 * screen its navigation. Shows the skeleton while loading and a retry card
 * when the request fails; the screen itself mounts only with data, so its
 * local form state can be initialised from the saved answers.
 */
export function StepScreen({ step, children }: StepScreenProps) {
  const t = useTranslations('onboarding.flow');
  const router = useRouter();
  const query = useOnboardingState();
  const modulesQuery = useAppModulesQuery();
  // A failed read fails open (every module on), like the API.
  const modules = modulesQuery.data ?? ALL_MODULES_ON;
  const progress = step === 'ready' ? undefined : { current: PROGRESS[step], total: PROGRESS_TOTAL };
  const storedGoal = isGoal(query.data?.goal) ? query.data.goal : null;
  const back = () => router.replace(previousRoute(step, storedGoal, modules.companion));
  // No gender / partner screens while the companion module is off (a resumed or typed URL lands on Goal).
  const skipped = !modulesQuery.isPending && !modules.companion && (step === 'gender' || step === 'partner');
  useEffect(() => {
    if (skipped) router.replace(FLOW_ROUTES.goal);
  }, [skipped, router]);

  if (query.isPending || modulesQuery.isPending || skipped) {
    return (
      <OnbFrame progress={progress} onBack={step === 'ready' ? undefined : back} privacyNote={false}>
        <SkeletonGroup label={t('loading')} className="onb2-skel">
          <Skeleton width="medium" />
          <Skeleton width="short" />
          <Skeleton shape="block" />
          <Skeleton shape="block" />
        </SkeletonGroup>
      </OnbFrame>
    );
  }

  if (query.isError || !query.data) {
    return (
      <OnbFrame progress={progress} onBack={step === 'ready' ? undefined : back} privacyNote={false}>
        <EmptyState
          icon="refresh"
          title={t('loadError.title')}
          body={t('loadError.body')}
          action={<PrimaryButton onClick={() => void query.refetch()}>{t('retry')}</PrimaryButton>}
        />
      </OnbFrame>
    );
  }

  const state = query.data;
  const ctx: StepContext = {
    state,
    goal: storedGoal,
    modules,
    progress,
    back,
    next: (override) => {
      const gender = override?.gender ?? state.gender;
      const goalRaw = override?.goal ?? state.goal;
      router.push(FLOW_ROUTES[nextStep(step, { gender, goal: isGoal(goalRaw) ? goalRaw : null, companion: modules.companion })]);
    },
  };
  return <>{children(ctx)}</>;
}

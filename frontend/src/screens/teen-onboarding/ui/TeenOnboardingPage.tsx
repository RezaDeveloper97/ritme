'use client';

import { useTranslations } from 'next-intl';
import { useState } from 'react';

import {
  TEEN_AGE_BANDS,
  TEEN_MENARCHE,
  type TeenAgeBand,
  type TeenMenarche,
  type TeenProfile,
  useSaveTeenProfile,
  useTeenProfile,
} from '@/entities/teen';
import { useLifeStage, useUpdateLifeStage } from '@/entities/user';
import { getApiSaveErrorMessage } from '@/shared/api';
import { useRouter } from '@/shared/i18n';
import {
  ChipGroup,
  PillChip,
  PrimaryButton,
  RadioCardGroup,
  type RadioCardOption,
  ScreenHeader,
  Skeleton,
  SkeletonGroup,
  SkyLayer,
} from '@/shared/ui';

const LOOK: Record<TeenMenarche, Pick<RadioCardOption<TeenMenarche>, 'icon' | 'iconTone'>> = {
  not_yet: { icon: 'sprout', iconTone: 'data' },
  under_1y: { icon: 'drop', iconTone: 'period' },
  over_1y: { icon: 'calendar', iconTone: 'brand' },
};

// «بیشتر از یک سال است» has no second line on the board.
const DESCRIPTION: Record<
  TeenMenarche,
  'menarcheOptions.not_yet.description' | 'menarcheOptions.under_1y.description' | null
> = {
  not_yet: 'menarcheOptions.not_yet.description',
  under_1y: 'menarcheOptions.under_1y.description',
  over_1y: null,
};

/**
 * «سلام! کمی از خودت بگو» (`/teen/onboarding`, CB-TEEN-02, nbl_Teen_Onb): age
 * chips, «پریود شده‌ای؟» cards and the privacy note. «ادامه» switches the life
 * stage to teen (`PUT /profile/life-stage`) and then saves the answers
 * (`PUT /teen/profile`), in that order (CB-TEEN-01). A form: no bottom nav.
 * Reached from the teen home while the answers are missing; back leads to the
 * mode switcher, never to `/home` (which would send her straight back here).
 */
export function TeenOnboardingPage() {
  const t = useTranslations('teen.onboarding');
  const router = useRouter();
  const query = useTeenProfile();

  return (
    <div className="view tob-page">
      <SkyLayer />
      <div className="scroll tob-scroll">
        <ScreenHeader
          title={null}
          center={<span className="nb-hdr-titles" aria-hidden />}
          onBack={() => router.push('/profile/mode')}
          backLabel={t('back')}
        />
        {query.isPending ? (
          <SkeletonGroup label={t('loading')} className="tob-skel">
            <Skeleton shape="line" width="medium" />
            <Skeleton shape="block" className="tob-skel-card" />
            <Skeleton shape="block" className="tob-skel-card is-tall" />
          </SkeletonGroup>
        ) : (
          // A failed read still lets her answer: the form just starts empty.
          <TeenForm saved={query.data?.profile ?? null} />
        )}
      </div>
    </div>
  );
}

function TeenForm({ saved }: { saved: TeenProfile | null }) {
  const t = useTranslations('teen.onboarding');
  const router = useRouter();
  const life = useLifeStage();
  const updateLife = useUpdateLifeStage();
  const save = useSaveTeenProfile();
  const [age, setAge] = useState<TeenAgeBand | null>(saved?.ageBand ?? null);
  const [menarche, setMenarche] = useState<TeenMenarche | null>(saved?.menarche ?? null);
  const [touched, setTouched] = useState(false);
  const busy = updateLife.isPending || save.isPending;
  const error = updateLife.error ?? save.error;

  const options = TEEN_MENARCHE.map((value) => {
    const desc = DESCRIPTION[value];
    return { value, title: t(`menarcheOptions.${value}.title`), description: desc ? t(desc) : undefined, ...LOOK[value] };
  });

  const submit = async () => {
    setTouched(true);
    if (!age || !menarche || busy) return;
    try {
      // The teen profile belongs to a teen-mode account: switch first (CB-TEEN-01).
      if (life.data?.mode !== 'teen') await updateLife.mutateAsync({ mode: 'teen' });
      await save.mutateAsync({ ageBand: age, menarche });
      router.replace('/home');
    } catch {
      // Shown below from the mutation's error state.
    }
  };

  return (
    <>
      <div className="tob-intro">
        <h1 className="tob-title">{t('title')}</h1>
        <p className="tob-lead">{t('privacy')}</p>
      </div>

      <section className="nb-card tob-card" aria-labelledby="tob-age">
        <h2 id="tob-age" className="tob-q">
          {t('age')}
        </h2>
        <ChipGroup label={t('age')} className="tob-ages">
          {TEEN_AGE_BANDS.map((band) => (
            <PillChip key={band} pressed={age === band} onPressedChange={() => setAge(band)}>
              {t(`ages.${band}`)}
            </PillChip>
          ))}
        </ChipGroup>
        {touched && !age ? (
          <p className="tob-error" role="alert">
            {t('pickAge')}
          </p>
        ) : null}
      </section>

      <section className="nb-card tob-card" aria-labelledby="tob-menarche">
        <h2 id="tob-menarche" className="tob-q">
          {t('menarche')}
        </h2>
        <RadioCardGroup options={options} value={menarche} onChange={setMenarche} label={t('menarche')} />
        {touched && !menarche ? (
          <p className="tob-error" role="alert">
            {t('pickMenarche')}
          </p>
        ) : null}
      </section>

      <div className="tob-footer">
        {error ? (
          <p className="tob-error" role="alert">
            {getApiSaveErrorMessage(error, t('saveError'))}
          </p>
        ) : null}
        <PrimaryButton loading={busy} disabled={touched && (!age || !menarche)} onClick={() => void submit()}>
          {t('continue')}
        </PrimaryButton>
      </div>
    </>
  );
}

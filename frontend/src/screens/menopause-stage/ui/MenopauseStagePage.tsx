'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useState } from 'react';

import {
  MENOPAUSE_STAGE_ANSWERS,
  type MenopauseProfile,
  type MenopauseStageAnswer,
  useMenopauseProfile,
  useSaveMenopauseProfile,
} from '@/entities/menopause';
import { getApiSaveErrorMessage } from '@/shared/api';
import { type Locale, useDirection, useRouter } from '@/shared/i18n';
import { formatMonthLabel, fromApiDate, partsToDate, toApiDate, todayParts, toParts } from '@/shared/lib/date';
import {
  EmptyState,
  Icon,
  PillChip,
  PrimaryButton,
  RadioCardGroup,
  type RadioCardOption,
  ScreenHeader,
  Skeleton,
  SkeletonGroup,
  SkyLayer,
} from '@/shared/ui';

import { canStepForward, type PickedMonth, stepMonth } from '../model/month';

const LOOK: Record<MenopauseStageAnswer, Pick<RadioCardOption<MenopauseStageAnswer>, 'icon' | 'iconTone'>> = {
  peri: { icon: 'clock', iconTone: 'period' },
  meno: { icon: 'moon', iconTone: 'brand' },
  post: { icon: 'shield', iconTone: 'data' },
  unsure: { icon: 'info', iconTone: 'neutral' },
};

/**
 * «حالت یائسگی» (`/menopause/stage`, CB-MENO-05, nbl_Meno_Stage): four stage
 * cards, the approximate last period (a month in the locale's calendar),
 * surgical cause and HRT → `PUT /menopause/profile`. Reached from the home's
 * stage chip / «انتخاب مرحله». A form: no bottom nav (IA_Nav).
 */
export function MenopauseStagePage() {
  const t = useTranslations('menopause.stage');
  const router = useRouter();
  const query = useMenopauseProfile();

  let body;
  if (query.isPending) {
    body = (
      <SkeletonGroup label={t('loading')} className="mst-skel">
        {Array.from({ length: 4 }, (_, i) => (
          <Skeleton key={i} shape="block" className="mst-skel-card" />
        ))}
        <Skeleton shape="card" />
      </SkeletonGroup>
    );
  } else if (query.isError) {
    body = (
      <EmptyState
        icon="moon"
        title={t('loadError')}
        action={
          <PrimaryButton icon="refresh" loading={query.isFetching} onClick={() => void query.refetch()}>
            {t('retry')}
          </PrimaryButton>
        }
      />
    );
  } else {
    body = <StageForm saved={query.data} />;
  }

  return (
    <div className="view mst-page">
      <SkyLayer />
      <div className="scroll mst-scroll">
        <ScreenHeader title={t('title')} onBack={() => router.push('/home')} backLabel={t('back')} />
        {body}
      </div>
    </div>
  );
}

function initialMonth(lastPeriod: string | null, locale: Locale): PickedMonth | null {
  if (!lastPeriod) return null;
  const p = toParts(fromApiDate(lastPeriod), locale);
  return { year: p.year, month: p.month };
}

function YesNo({ label, value, onChange }: { label: string; value: boolean | null; onChange: (v: boolean | null) => void }) {
  const t = useTranslations('menopause.stage');
  return (
    <div className="mst-row" role="group" aria-label={label}>
      <span className="mst-q">{label}</span>
      <span className="mst-yn">
        <PillChip mode="multi" pressed={value === true} onPressedChange={(on) => onChange(on ? true : null)}>
          {t('yes')}
        </PillChip>
        <PillChip mode="multi" pressed={value === false} onPressedChange={(on) => onChange(on ? false : null)}>
          {t('no')}
        </PillChip>
      </span>
    </div>
  );
}

function StageForm({ saved }: { saved: MenopauseProfile }) {
  const t = useTranslations('menopause.stage');
  const tStage = useTranslations('menopause.stages');
  const locale = useLocale() as Locale;
  const rtl = useDirection() === 'rtl';
  const router = useRouter();
  const save = useSaveMenopauseProfile();
  const [stage, setStage] = useState<MenopauseStageAnswer | null>(saved.storedStage);
  const [lastPeriod, setLastPeriod] = useState<PickedMonth | null>(() => initialMonth(saved.lastPeriod, locale));
  const [surgical, setSurgical] = useState<boolean | null>(saved.surgical);
  const [hrt, setHrt] = useState<boolean | null>(saved.hrt);
  const [touched, setTouched] = useState(false);

  const now = todayParts(locale);
  const nowMonth = { year: now.year, month: now.month };

  const options = MENOPAUSE_STAGE_ANSWERS.map((value) => ({
    value,
    title: tStage(value),
    description: t(`desc.${value}`),
    ...LOOK[value],
  }));

  const submit = () => {
    setTouched(true);
    if (!stage || save.isPending) return;
    save.mutate(
      {
        stage,
        // Approximate: the first day of the chosen month (the API's contract).
        lastPeriod: lastPeriod ? toApiDate(partsToDate({ ...lastPeriod, day: 1 }, locale)) : null,
        surgical,
        hrt,
      },
      { onSuccess: () => router.replace('/home') },
    );
  };

  return (
    <>
      <div className="mst-intro">
        <h2 className="mst-question">{t('question')}</h2>
        <p className="mst-lead">{t('lead')}</p>
      </div>

      <RadioCardGroup options={options} value={stage} onChange={setStage} label={t('question')} />
      {touched && !stage ? (
        <p className="mst-error" role="alert">
          {t('pickStage')}
        </p>
      ) : null}

      <section className="nb-card mst-card" aria-label={t('details')}>
        <div className="mst-row">
          <span className="mst-label">{t('lastPeriod')}</span>
          <span className="mst-month">
            <button
              type="button"
              className="mst-step"
              aria-label={t('prevMonth')}
              onClick={() => setLastPeriod((p) => stepMonth(p, nowMonth, -1))}
            >
              <Icon name={rtl ? 'chevronRight' : 'chevronLeft'} size={18} />
            </button>
            <b className="mst-month-value" aria-live="polite">
              {lastPeriod
                ? t('about', { month: formatMonthLabel(lastPeriod.year, lastPeriod.month, locale) })
                : t('notSet')}
            </b>
            <button
              type="button"
              className="mst-step"
              aria-label={t('nextMonth')}
              disabled={!canStepForward(lastPeriod, nowMonth)}
              onClick={() => setLastPeriod((p) => stepMonth(p, nowMonth, 1))}
            >
              <Icon name={rtl ? 'chevronLeft' : 'chevronRight'} size={18} />
            </button>
          </span>
        </div>
        <YesNo label={t('surgical')} value={surgical} onChange={setSurgical} />
        <YesNo label={t('hrt')} value={hrt} onChange={setHrt} />
      </section>

      <div className="mst-footer">
        {save.isError ? (
          <p className="mst-error" role="alert">
            {getApiSaveErrorMessage(save.error, t('saveError'))}
          </p>
        ) : null}
        <PrimaryButton loading={save.isPending} disabled={touched && !stage} onClick={submit}>
          {saved.storedStage ? t('save') : t('start')}
        </PrimaryButton>
      </div>
    </>
  );
}

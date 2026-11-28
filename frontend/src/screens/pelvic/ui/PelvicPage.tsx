'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useState } from 'react';

import {
  PELVIC_KEGEL_PATH,
  PELVIC_LEAKS,
  PELVIC_MAX_NIGHT_VOIDS,
  PELVIC_UTI_SYMPTOMS,
  type PelvicDiary,
  type PelvicDiaryPatch,
  type PelvicProgram,
  usePelvic,
  usePelvicAlerts,
  useSavePelvicDiary,
  useStartPelvicProgram,
} from '@/entities/pelvic';
import { type Locale, useRouter } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib/date';
import { pilot, pilotOption } from '@/shared/pilot';
import {
  Card,
  ChipGroup,
  EmptyState,
  IconCircle,
  InfoNote,
  NumberStepper,
  PillChip,
  PrimaryButton,
  ScreenHeader,
  SecondaryButton,
  Skeleton,
  SkeletonGroup,
  SkyLayer,
  StatusPill,
  UrgentCard,
  WeekDots,
} from '@/shared/ui';
import { BottomNav } from '@/widgets/bottom-nav';

import { nextLeak, sessionMinutes, toggleSymptom, weekDotStates } from '../model/plan';
import './pv.css';

/**
 * «کف لگن و مثانه» (`/programs/pelvic`, CB-PELV-02, nbl_Pelvic_Plan): the 8-week plan card (week · level, streak,
 * this week's dots, «start today's session» → the Kegel trainer), today's bladder diary (leak, night waking), UTI
 * symptoms with the see-a-doctor warning, and the suitability note. Listed in the programs hub; a services hub, so
 * the bottom nav shows (خدمات).
 */
export function PelvicPage() {
  const t = useTranslations('pelvic');
  const locale = useLocale();
  const router = useRouter();
  const query = usePelvic();
  const alerts = usePelvicAlerts(locale);

  let body;
  if (query.isPending) {
    body = (
      <SkeletonGroup label={t('plan.loading')} className="pv-skel">
        <Skeleton shape="card" className="pv-skel-plan" />
        <Skeleton shape="card" className="pv-skel-card" />
        <Skeleton shape="card" className="pv-skel-card" />
      </SkeletonGroup>
    );
  } else if (query.isError) {
    body = (
      <EmptyState
        icon="warning"
        title={t('plan.loadError')}
        action={
          <SecondaryButton icon="refresh" block={false} loading={query.isFetching} onClick={() => void query.refetch()}>
            {t('plan.retry')}
          </SecondaryButton>
        }
      />
    );
  } else {
    const { program, diaryToday } = query.data;
    body = (
      <>
        {program ? <PlanCard program={program} today={diaryToday.date} /> : <JoinCard />}
        <DiarySections key={diaryToday.date} diary={diaryToday} utiWarning={alerts.data?.utiWarning ?? null} />
      </>
    );
  }

  return (
    <div className="view pv-page">
      <SkyLayer />
      <div className="scroll pv-scroll">
        <ScreenHeader title={t('plan.title')} onBack={() => router.back('/programs')} backLabel={t('plan.back')} />
        <div className="pv-body">
          {body}
          <InfoNote className="pv-note">{alerts.data?.suitability ?? t('suitability')}</InfoNote>
        </div>
      </div>
      <BottomNav />
    </div>
  );
}

function PlanCard({ program, today }: { program: PelvicProgram; today: string }) {
  const t = useTranslations('pelvic.plan');
  const locale = useLocale() as Locale;
  const router = useRouter();
  const n = (value: number) => formatNumber(value, locale);
  const { states, todayIndex } = weekDotStates(program.weekDays, today, locale);
  const level = program.level;
  const minutes = n(sessionMinutes(level?.sessionSec ?? 0));
  return (
    <Card as="section" className="pv-plan" aria-labelledby="pv-plan-title">
      <div className="pv-plan-head">
        <div className="pv-plan-titles">
          <p className="pv-plan-kicker">{t('programWeeks', { weeks: n(program.weeksTotal) })}</p>
          <h2 id="pv-plan-title" className="pv-plan-title">
            {level
              ? t('weekLevel', { week: n(program.week), level: n(level.number) })
              : t('weekOnly', { week: n(program.week) })}
          </h2>
        </div>
        {program.streakDays > 0 ? (
          <StatusPill tone="data" className="pv-streak">
            {t('streak', { days: n(program.streakDays) })}
          </StatusPill>
        ) : null}
      </div>
      <WeekDots
        days={states}
        todayIndex={todayIndex}
        locale={locale}
        label={t('weekDotsLabel')}
        stateLabels={{ done: t('dot.done'), missed: t('dot.missed'), future: t('dot.future') }}
        className="pv-dots"
      />
      {program.completed ? <p className="pv-plan-line">{t('completed')}</p> : null}
      {program.todayDone ? <p className="pv-plan-line is-done">{t('todayDone')}</p> : null}
      {level ? (
        <PrimaryButton onClick={() => router.push(PELVIC_KEGEL_PATH)} {...pilot('pelvic.session-start')}>
          {t(program.todayDone ? 'again' : 'start', { minutes })}
        </PrimaryButton>
      ) : null}
    </Card>
  );
}

function JoinCard() {
  const t = useTranslations('pelvic.plan.join');
  const start = useStartPelvicProgram();
  return (
    <Card as="section" className="pv-plan pv-join" aria-labelledby="pv-join-title">
      <div className="pv-join-head">
        <IconCircle icon="target" tone="data" size="lg" />
        <h2 id="pv-join-title" className="pv-plan-title is-join">
          {t('title')}
        </h2>
      </div>
      <p className="pv-join-body">{t('body')}</p>
      {start.isError ? (
        <p className="pv-error" role="alert">
          {t('error')}
        </p>
      ) : null}
      <PrimaryButton loading={start.isPending} onClick={() => start.mutate()} {...pilot('pelvic.program-join')}>
        {t('action')}
      </PrimaryButton>
    </Card>
  );
}

/** Today's diary: every change is saved at once (PUT with only that key); the screen mirrors the answer. */
function DiarySections({ diary, utiWarning }: { diary: PelvicDiary; utiWarning: string | null }) {
  const t = useTranslations('pelvic');
  const locale = useLocale() as Locale;
  const save = useSavePelvicDiary();
  const [leak, setLeak] = useState(diary.leak);
  const [voids, setVoids] = useState(diary.nightVoids ?? 0);
  const [symptoms, setSymptoms] = useState(diary.utiSymptoms);

  const send = (patch: PelvicDiaryPatch) => save.mutate({ date: diary.date, patch });
  const alert = save.data ? save.data.utiAlert : diary.utiAlert;

  return (
    <>
      <section className="pv-sec" aria-labelledby="pv-diary">
        <h2 id="pv-diary" className="nb-sect-title">
          {t('diary.title')}
        </h2>
        <Card className="pv-card">
          <p id="pv-leak" className="pv-q">
            {t('diary.leak.question')}
          </p>
          <ChipGroup label={t('diary.leak.question')} className="pv-chips" pilot={pilot('pelvic.leak', 'choice')}>
            {PELVIC_LEAKS.map((code) => (
              <PillChip
                key={code}
                mode={code === 'none' ? 'single' : 'multi'}
                tone="period"
                pressed={leak === code}
                onPressedChange={() => {
                  const next = nextLeak(leak, code);
                  setLeak(next);
                  send({ leak: next });
                }}
                {...pilotOption(code)}
              >
                {t(`diary.leak.${code}`)}
              </PillChip>
            ))}
          </ChipGroup>
          <hr className="pv-rule" />
          <NumberStepper
            className="pv-voids"
            value={voids}
            min={0}
            max={PELVIC_MAX_NIGHT_VOIDS}
            label={t('diary.nightVoids')}
            decrementLabel={t('diary.nightVoidsLess')}
            incrementLabel={t('diary.nightVoidsMore')}
            locale={locale}
            pilot={pilot('pelvic.night-voids', 'stepper')}
            onChange={(value) => {
              setVoids(value);
              send({ night_voids: value });
            }}
          />
        </Card>
      </section>

      <section className="pv-sec" aria-labelledby="pv-uti">
        <h2 id="pv-uti" className="nb-sect-title">
          {t('uti.title')}
        </h2>
        <Card className="pv-card">
          <ChipGroup label={t('uti.label')} className="pv-chips" pilot={pilot('pelvic.uti-symptoms', 'choice')}>
            {PELVIC_UTI_SYMPTOMS.map((code) => (
              <PillChip
                key={code}
                mode="multi"
                tone="danger"
                pressed={symptoms.includes(code)}
                onPressedChange={() => {
                  const next = toggleSymptom(symptoms, code, PELVIC_UTI_SYMPTOMS);
                  setSymptoms(next);
                  send({ uti_symptoms: next });
                }}
                {...pilotOption(code)}
              >
                {t(`uti.${code}`)}
              </PillChip>
            ))}
          </ChipGroup>
          {alert ? (
            <UrgentCard variant="note" urgent title={utiWarning ?? t('uti.warning')} className="pv-uti-alert" />
          ) : (
            <p className="pv-uti-hint">{utiWarning ?? t('uti.warning')}</p>
          )}
        </Card>
        {save.isError ? (
          <p className="pv-error" role="alert">
            {t('diary.saveError')}
          </p>
        ) : null}
      </section>
    </>
  );
}

'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useId, useState } from 'react';

import {
  LOSS_MOODS,
  type LossCatalogItem,
  type LossFollowup,
  type LossMood,
  type LossState,
  crisisHotlines,
  emergencyNumber,
  shouldRecordLoss,
  splitWallClock,
  useLogLossMood,
  useLossCatalog,
  useLossState,
  warningSignParts,
} from '@/entities/loss';
import { getApiSaveErrorMessage } from '@/shared/api';
import { type Locale, useRouter } from '@/shared/i18n';
import { formatDayMonth, formatNumber, fromApiDate, toApiDate, today } from '@/shared/lib/date';
import {
  ChipGroup,
  EmptyState,
  Icon,
  ListGroup,
  ListRow,
  PillChip,
  PrimaryButton,
  ScreenHeader,
  Skeleton,
  SkeletonGroup,
  SkyLayer,
  UrgentCard,
} from '@/shared/ui';

import { BetaSheet, BleedingSheet, NoteSheet, VisitSheet } from './FollowupSheets';

type SheetId = 'bleeding' | 'beta' | 'visit' | 'note';

const BLEEDING_LEVELS = ['spotting', 'light', 'medium', 'heavy', 'very_heavy'] as const;
type BleedingLevel = (typeof BLEEDING_LEVELS)[number];
const isBleedingLevel = (v: string | null): v is BleedingLevel => BLEEDING_LEVELS.includes(v as BleedingLevel);

/**
 * «مراقبت از خودت» (`/loss/care`, CB-LOSS-02, nbl_Loss_Care): the warning signs
 * with a one-tap 115 call, the physical follow-up (bleeding until it stops,
 * beta hCG until negative, a visit ~2 weeks later — dated ones become private
 * care appointments via `PUT /loss/followup`), today's mood, the encrypted
 * private note and the crisis helplines. Copy is catalog content (`loss_*`,
 * [needs clinical review]); the bundle only fills gaps.
 *
 * The «گفت‌وگو با مشاور» row of the board stays hidden until the N7 doctors &
 * counsellors directory exists. Full screen: no tab bar, banners or shop.
 */
export function LossCarePage() {
  const t = useTranslations('loss.common');
  const tc = useTranslations('loss.care');
  const router = useRouter();
  const state = useLossState();
  const [sheet, setSheet] = useState<SheetId | null>(null);
  const loss = state.data?.loss ?? null;
  const followup = state.data?.followup ?? null;
  // Back to the start only while it can still be corrected (the same day).
  const back = loss && shouldRecordLoss(loss, false, toApiDate(today())) ? '/loss' : '/home';

  let body;
  if (state.isPending) {
    body = (
      <SkeletonGroup label={t('loading')} className="lsc-skel">
        <Skeleton shape="card" />
        <Skeleton shape="card" />
        <Skeleton shape="card" />
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
  } else if (!state.data.loss || !state.data.followup) {
    body = (
      <EmptyState
        icon="heart"
        title={t('noLoss.title')}
        body={t('noLoss.body')}
        action={<PrimaryButton onClick={() => router.replace('/loss')}>{t('noLoss.action')}</PrimaryButton>}
      />
    );
  } else {
    body = <CareBody state={state.data} followup={state.data.followup} onOpen={setSheet} />;
  }

  return (
    <div className="view lsc-page">
      <SkyLayer />
      <div className="scroll lsc-scroll">
        <ScreenHeader title={tc('title')} onBack={() => router.push(back)} backLabel={t('back')} />
        {body}
      </div>
      {/* Sheets sit outside the scroller: AppSheet is positioned against the screen, not the scrolled content. */}
      {sheet === 'bleeding' ? <BleedingSheet onClose={() => setSheet(null)} /> : null}
      {sheet === 'beta' && followup ? <BetaSheet beta={followup.beta} onClose={() => setSheet(null)} /> : null}
      {sheet === 'visit' && followup ? <VisitSheet visit={followup.visit} onClose={() => setSheet(null)} /> : null}
      {sheet === 'note' ? <NoteSheet onClose={() => setSheet(null)} /> : null}
    </div>
  );
}

function byCode(items: readonly LossCatalogItem[] | undefined, code: string): LossCatalogItem | null {
  return items?.find((i) => i.code === code) ?? null;
}

function CareBody({
  state,
  followup,
  onOpen: setSheet,
}: {
  state: LossState;
  followup: LossFollowup;
  onOpen: (sheet: SheetId) => void;
}) {
  const t = useTranslations('loss.care');
  const tc = useTranslations('loss.common');
  const locale = useLocale() as Locale;
  const router = useRouter();
  const ids = useId();
  const warnings = useLossCatalog('loss_warning_signs', locale);
  const hotlines = useLossCatalog('loss_hotlines', locale);
  const followups = useLossCatalog('loss_followups', locale);
  const moods = useLossCatalog('loss_moods', locale);
  const support = useLossCatalog('loss_support', locale);
  const mood = useLogLossMood();

  const day = (iso: string) => formatDayMonth(fromApiDate(iso), locale);

  // Danger card: the warning signs as one sentence, and the emergency number.
  const parts = warningSignParts(warnings.data ?? []);
  const warnBody = parts.length ? `${parts.join(tc('listSeparator'))}.` : t('warnFallback');
  const emergency = emergencyNumber(warnings.data ?? [], hotlines.data ?? []);
  const emergencyItem = (hotlines.data ?? []).find((h) => h.code === 'emergency');
  const callLabel = t('call', {
    name: emergencyItem?.title ?? t('callFallback', { number: formatNumber(emergency, locale) }),
  });

  // Follow-up rows.
  const bleedingItem = byCode(followups.data, 'bleeding');
  const betaItem = byCode(followups.data, 'beta');
  const visitItem = byCode(followups.data, 'visit');
  const { bleeding, beta, visit } = followup;
  const level = isBleedingLevel(bleeding.today) ? t(`bleeding.levels.${bleeding.today}`) : t('bleeding.notLogged');
  const bleedingSub = bleeding.stopped
    ? t('bleeding.stopped', { date: bleeding.stoppedOn ? day(bleeding.stoppedOn) : '' })
    : `${level} · ${bleedingItem?.body ?? t('bleeding.sub')}`;
  const betaSub = beta.negative
    ? t('beta.negative', { date: beta.negativeOn ? day(beta.negativeOn) : '' })
    : beta.nextOn
      ? t('beta.next', { date: day(beta.nextOn) })
      : (betaItem?.body ?? t('beta.sub'));
  const booked = visit.appointment ? splitWallClock(visit.appointment.scheduledAt) : null;
  const visitSub = booked
    ? t('visit.booked', { date: day(booked.day), time: formatNumber(booked.time, locale) })
    : (visitItem?.body ?? t('visit.sub'));

  const moodSupport = byCode(support.data, 'mood_support');
  const crisis = byCode(support.data, 'crisis');
  const crisisLines = crisisHotlines(crisis, hotlines.data ?? []);

  return (
    <>
      <UrgentCard
        title={t('warnTitle')}
        urgent={false}
        className="lsc-danger"
        action={
          <a className="nb-btn is-primary is-block lsc-call" href={`tel:${emergency}`}>
            <Icon name="phone" size={18} />
            {callLabel}
          </a>
        }
      >
        <p className="lsc-danger-body">{warnBody}</p>
      </UrgentCard>

      <section className="lsc-sec" aria-labelledby={`${ids}-phys`}>
        <h2 id={`${ids}-phys`} className="lsc-sec-title">
          {t('physical')}
        </h2>
        <ListGroup className="lsc-list">
          <ListRow
            icon="drop"
            iconTone="period"
            title={t('bleeding.title')}
            description={bleedingSub}
            trailing={
              bleeding.stopped ? (
                <DoneMark label={t('done')} />
              ) : (
                <button type="button" className="lsc-link" onClick={() => setSheet('bleeding')}>
                  {t('bleeding.action')}
                </button>
              )
            }
          />
          <ListRow
            icon="note"
            iconTone="brand"
            title={betaItem?.title ?? t('beta.title')}
            description={betaSub}
            trailing={
              beta.negative ? (
                <DoneMark label={t('done')} />
              ) : (
                <button type="button" className="lsc-pill" onClick={() => setSheet('beta')}>
                  {t('beta.action')}
                </button>
              )
            }
          />
          <ListRow
            icon="stetho"
            iconTone="data"
            title={visitItem?.title ?? t('visit.title')}
            description={visitSub}
            trailing={
              <button type="button" className="lsc-pill is-brand" onClick={() => setSheet('visit')}>
                {booked ? t('visit.change') : t('visit.action')}
              </button>
            }
          />
        </ListGroup>
      </section>

      <section className="lsc-sec" aria-labelledby={`${ids}-feel`}>
        <h2 id={`${ids}-feel`} className="lsc-sec-title">
          {moodSupport?.title ?? t('feelings')}
        </h2>
        <div className="nb-card lsc-mood">
          <p className="lsc-mood-q">
            {t('moodQuestion')}
          </p>
          <ChipGroup label={t('moodQuestion')} className="lsc-chips">
            {LOSS_MOODS.map((code: LossMood) => (
              <PillChip
                key={code}
                pressed={state.mood?.today === code}
                disabled={mood.isPending}
                onPressedChange={() => state.mood?.today !== code && mood.mutate(code)}
              >
                {byCode(moods.data, code)?.title ?? t(`moods.${code}`)}
              </PillChip>
            ))}
          </ChipGroup>
          <p className="lsc-mood-note">{moodSupport?.body ?? t('moodSupport')}</p>
          <p className="lsc-status" role="status">
            {mood.isError ? getApiSaveErrorMessage(mood.error, tc('saveError')) : mood.isSuccess ? t('moodSaved') : ''}
          </p>
        </div>

        {/* «گفت‌وگو با مشاور» — hidden until the N7 counsellor directory exists (docs/canvas-build/README.md). */}
        <ListGroup className="lsc-list">
          <ListRow
            icon="pen"
            iconTone="period"
            title={t('note.title')}
            description={state.note?.hasNote ? t('note.saved') : t('note.sub')}
            onClick={() => setSheet('note')}
          />
        </ListGroup>
      </section>

      <UrgentCard
        variant="note"
        icon="heart"
        title={crisis?.body ?? t('crisisFallback')}
        hotlinesLabel={t('hotlines')}
        hotlines={crisisLines.map((line) => ({
          number: line.number,
          label: line.label ?? t('hotlineFallback'),
          display: formatNumber(line.number, locale),
        }))}
        className="lsc-crisis"
      />

      <div className="lsc-footer">
        <PrimaryButton onClick={() => router.push('/loss/next')}>{tc('continue')}</PrimaryButton>
      </div>
    </>
  );
}

function DoneMark({ label }: { label: string }) {
  return (
    <span className="lsc-done">
      <Icon name="checkCircle" size={18} />
      <span>{label}</span>
    </span>
  );
}


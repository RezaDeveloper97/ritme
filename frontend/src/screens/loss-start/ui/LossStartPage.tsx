'use client';

import { useQueryClient } from '@tanstack/react-query';
import { useLocale, useTranslations } from 'next-intl';
import { useEffect, useId, useState } from 'react';

import { useCompanions } from '@/entities/companion';
import {
  LOSS_TYPES,
  type LossEvent,
  type LossType,
  lossFieldError,
  shouldRecordLoss,
  useLossCatalog,
  useLossState,
  useRecordLoss,
} from '@/entities/loss';
import { useLifeStage } from '@/entities/user';
import { getApiSaveErrorMessage } from '@/shared/api';
import { type Locale, useRouter } from '@/shared/i18n';
import { formatLongDate, fromApiDate, toApiDate, today } from '@/shared/lib/date';
import {
  EmptyState,
  type IconName,
  PrimaryButton,
  RadioCardGroup,
  type RadioCardOption,
  ScreenHeader,
  Skeleton,
  SkeletonGroup,
  SkyLayer,
  Switch,
  type Tone,
} from '@/shared/ui';

import { LossDateSheet } from './LossDateSheet';

const LOOK: Record<LossType, { icon: IconName; tone: Tone }> = {
  early_miscarriage: { icon: 'heart', tone: 'period' },
  late_miscarriage: { icon: 'heart', tone: 'period' },
  ectopic: { icon: 'warning', tone: 'bloom' },
  chemical: { icon: 'dropLine', tone: 'brand' },
  unspecified: { icon: 'info', tone: 'neutral' },
};

/**
 * «متأسفیم که این را تجربه کردی» (`/loss`, CB-LOSS-02, nbl_Loss_Start): the
 * calm start of the loss path — what happened (optional), an approximate date
 * (optional), the always-on stop of pregnancy content, and an opt-in one-line
 * notice to a companion who holds the pregnancy grant (off by default, hidden
 * without such a link). Full screen: no tab bar, no banners, nothing celebratory.
 *
 * Entry: bloom's calm exit in `/profile/mode` (B-N2-03) and, later, the IVF
 * negative result (CB-IVF-05). «ادامه» records the loss (`POST /loss`, which
 * ends pregnancy mode server-side) and opens `/loss/care`. An earlier loss is
 * re-posted only on the same day (a correction) or while she is pregnant again;
 * otherwise she goes straight on to the care screen.
 */
export function LossStartPage() {
  const t = useTranslations('loss.common');
  const router = useRouter();
  const state = useLossState();
  const stage = useLifeStage();
  const [date, setDate] = useState<string | null>(null);
  const [dateOpen, setDateOpen] = useState(false);

  const pregnant = stage.data?.mode === 'pregnancy';
  const loss = state.data?.loss ?? null;
  const ready = state.isSuccess && !stage.isPending;
  const record = ready ? shouldRecordLoss(loss, pregnant, toApiDate(today())) : true;

  useEffect(() => {
    if (ready && !record) router.replace('/loss/care');
  }, [ready, record, router]);

  let body;
  if (state.isError) {
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
  } else if (!ready || !record) {
    body = (
      <SkeletonGroup label={t('loading')} className="lst-skel">
        <Skeleton shape="block" className="lst-skel-card" />
        <Skeleton shape="block" className="lst-skel-card" />
        <Skeleton shape="block" className="lst-skel-card" />
      </SkeletonGroup>
    );
  } else {
    // Same-day correction: start from what she answered earlier today.
    body = <StartForm previous={loss && !pregnant ? loss : null} date={date} setDate={setDate} onPickDate={() => setDateOpen(true)} />;
  }

  return (
    <div className="view lst-page">
      <SkyLayer />
      <div className="scroll lst-scroll">
        <ScreenHeader
          title={null}
          onBack={() => router.push(pregnant ? '/profile/mode' : '/home')}
          backLabel={t('back')}
        />
        {body}
      </div>
      {/* Outside the scroller: AppSheet is positioned against the screen, not the scrolled content. */}
      {dateOpen ? (
        <LossDateSheet
          value={date}
          onClose={() => setDateOpen(false)}
          onPick={(picked) => {
            setDate(picked);
            setDateOpen(false);
          }}
        />
      ) : null}
    </div>
  );
}

function StartForm({
  previous,
  date,
  setDate,
  onPickDate,
}: {
  previous: LossEvent | null;
  date: string | null;
  setDate: (date: string | null) => void;
  onPickDate: () => void;
}) {
  const t = useTranslations('loss.start');
  const tc = useTranslations('loss.common');
  const locale = useLocale() as Locale;
  const router = useRouter();
  const queryClient = useQueryClient();
  const ids = useId();
  const types = useLossCatalog('loss_types', locale);
  const companions = useCompanions();
  const save = useRecordLoss();

  const [type, setType] = useState<LossType | null>(previous?.type ?? null);
  const [notify, setNotify] = useState(previous?.notifyCompanion ?? false);

  // A same-day correction starts from the date she gave earlier today.
  const prefill = previous?.occurredOn ?? null;
  useEffect(() => {
    if (prefill) setDate(prefill);
  }, [prefill, setDate]);

  // The notice only reaches an active companion holding the pregnancy grant (CB-LOSS-01).
  const canTell = (companions.data ?? []).some((c) => c.status === 'active' && c.grants.pregnancy !== 'none');
  const told = previous?.companionNotified ?? false;

  const titles = new Map((types.data ?? []).map((i) => [i.code, i.title]));
  const options: RadioCardOption<LossType>[] = LOSS_TYPES.map((code) => ({
    value: code,
    title: titles.get(code) ?? t(`types.${code}`),
    icon: LOOK[code].icon,
    iconTone: LOOK[code].tone,
  }));

  const submit = () => {
    if (save.isPending) return;
    save.mutate(
      {
        ...(type ? { type } : {}),
        // A cleared date on a same-day correction is sent as null (clears); otherwise only when given.
        ...(date || previous?.occurredOn ? { occurredOn: date } : {}),
        notifyCompanion: canTell && notify,
      },
      {
        onSuccess: () => {
          // The loss ends pregnancy mode server-side: every pregnancy surface refetches.
          void queryClient.invalidateQueries();
          router.push('/loss/care');
        },
      },
    );
  };

  const saveError = lossFieldError(save.error) ?? getApiSaveErrorMessage(save.error, tc('saveError'));

  return (
    <>
      <div className="lst-intro">
        <h1 className="lst-title">{t('title')}</h1>
        <p className="lst-lead">{t('lead')}</p>
      </div>

      <section className="nb-card lst-card" aria-labelledby={`${ids}-q`}>
        <h2 id={`${ids}-q`} className="lst-card-title">
          {t('question')}
        </h2>
        <RadioCardGroup options={options} value={type} onChange={setType} label={t('question')} />
      </section>

      <section className="nb-card lst-card lst-settings" aria-label={t('date')}>
        <button type="button" className="lst-row lst-date" onClick={onPickDate}>
          <span className="lst-row-label">{t('date')}</span>
          <b className="lst-date-value">{date ? formatLongDate(fromApiDate(date), locale) : t('dateNone')}</b>
        </button>

        <div className="lst-row">
          <span className="lst-row-text">
            <span id={`${ids}-stop`} className="lst-row-title">
              {t('stopTitle')}
            </span>
            <span id={`${ids}-stop-d`} className="lst-row-sub">
              {t('stopSub')}
              <span className="sr-only"> · {t('stopFixed')}</span>
            </span>
          </span>
          {/* CB-LOSS-01 always stops pregnancy content: shown as a fixed ON state. */}
          <Switch
            checked
            disabled
            onCheckedChange={() => undefined}
            labelledBy={`${ids}-stop`}
            describedBy={`${ids}-stop-d`}
            className="lst-fixed"
          />
        </div>

        {canTell || told ? (
          <div className="lst-row">
            <span className="lst-row-text">
              <span id={`${ids}-tell`} className="lst-row-title">
                {t('companionTitle')}
              </span>
              <span id={`${ids}-tell-d`} className="lst-row-sub">
                {told ? t('companionSent') : t('companionSub')}
              </span>
            </span>
            <Switch
              checked={told || notify}
              disabled={told}
              onCheckedChange={setNotify}
              labelledBy={`${ids}-tell`}
              describedBy={`${ids}-tell-d`}
              className={told ? 'lst-fixed' : undefined}
            />
          </div>
        ) : null}
      </section>

      <div className="lst-footer">
        {save.isError ? (
          <p className="lst-error" role="alert">
            {saveError}
          </p>
        ) : null}
        <PrimaryButton loading={save.isPending} onClick={submit}>
          {save.isPending ? t('saving') : tc('continue')}
        </PrimaryButton>
      </div>
    </>
  );
}

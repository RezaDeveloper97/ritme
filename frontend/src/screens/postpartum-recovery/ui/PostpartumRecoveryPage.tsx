'use client';

import { useQueryClient } from '@tanstack/react-query';
import { useLocale, useTranslations } from 'next-intl';
import { useEffect, useRef, useState } from 'react';

import { logKeys } from '@/entities/health-log';
import {
  BREAST_SYMPTOMS,
  fieldError,
  LOCHIA_AMOUNTS,
  LOCHIA_COLORS,
  MAX_FEEDS,
  MAX_SLEEP_HOURS,
  MAX_TEMPERATURE_C,
  MIN_TEMPERATURE_C,
  PAIN_LEVELS,
  PAIN_LOCATIONS,
  type PostpartumAlert,
  type PostpartumOverview,
  type PostpartumRecovery,
  usePostpartum,
  usePostpartumRecovery,
  useSaveRecovery,
} from '@/entities/postpartum';
import { getApiErrorStatus, getApiSaveErrorMessage } from '@/shared/api';
import { Link, type Locale, useDirection, useRouter } from '@/shared/i18n';
import { addDays, formatNumber, fromApiDate, monthName, toParts } from '@/shared/lib/date';
import {
  Card,
  ChipGroup,
  EmptyState,
  Icon,
  NumberStepper,
  PillChip,
  PrimaryButton,
  ScreenHeader,
  SecondaryButton,
  Skeleton,
  SkeletonGroup,
  SkyLayer,
  UrgentCard,
} from '@/shared/ui';

import {
  diffForm,
  formFrom,
  locationMissing,
  needsLocation,
  RED_FLAGS,
  type RecoveryForm,
  roundTemperature,
  TEMPERATURE_START,
  toggleIn,
} from '../model/form';

/**
 * `/postpartum/recovery` — today's recovery form (nbl_v15_Recovery): lochia
 * amount + colour, pain + places, feeds, breasts, sleep → partial
 * `PUT /postpartum/recovery`. A heavy-bleeding answer comes back with a call
 * alert, shown here before leaving. A form: no bottom nav.
 */
export function PostpartumRecoveryPage() {
  const t = useTranslations('postpartum.recovery');
  const tc = useTranslations('postpartum.common');
  const router = useRouter();
  const recovery = usePostpartumRecovery(null);
  const overview = usePostpartum();
  const day = overview.data?.status?.daysSinceBirth ?? null;
  const locale = useLocale() as Locale;

  let body;
  if (recovery.isPending) {
    body = (
      <SkeletonGroup label={tc('loading')} className="pp-form">
        <Skeleton shape="card" />
        <Skeleton shape="card" />
      </SkeletonGroup>
    );
  } else if (recovery.isError) {
    const notActive = getApiErrorStatus(recovery.error) === 409;
    body = (
      <div className="pp-form">
        <Card>
          <EmptyState
            icon={notActive ? 'heart' : 'warning'}
            title={notActive ? t('notActive') : tc('loadError')}
            action={
              notActive ? (
                <Link href="/postpartum" className="nb-btn is-primary is-block">
                  {t('goHome')}
                </Link>
              ) : (
                <PrimaryButton icon="refresh" loading={recovery.isFetching} onClick={() => void recovery.refetch()}>
                  {tc('retry')}
                </PrimaryButton>
              )
            }
          />
        </Card>
      </div>
    );
  } else {
    body = <RecoveryFormView key={recovery.dataUpdatedAt} saved={recovery.data} overview={overview.data} />;
  }

  return (
    <div className="view pp-screen pp-form-page">
      <SkyLayer />
      <div className="scroll">
        <ScreenHeader
          title={t('title')}
          subtitle={day != null ? t('subtitle', { day: formatNumber(day, locale) }) : undefined}
          onBack={() => router.push('/postpartum')}
          backLabel={tc('back')}
        />
        {body}
      </div>
    </div>
  );
}

function RecoveryFormView({ saved, overview }: { saved: PostpartumRecovery; overview: PostpartumOverview | undefined }) {
  const t = useTranslations('postpartum.recovery');
  const locale = useLocale() as Locale;
  const router = useRouter();
  const queryClient = useQueryClient();
  const save = useSaveRecovery();
  const base = formFrom(saved);
  const [form, setForm] = useState<RecoveryForm>(base);
  const [tried, setTried] = useState(false);
  const [alerts, setAlerts] = useState<PostpartumAlert[]>([]);
  const alertsRef = useRef<HTMLDivElement>(null);
  // An urgent answer comes back at the top of the form: bring it into view (the save button is at the bottom).
  useEffect(() => {
    if (alerts.length) alertsRef.current?.scrollIntoView({ block: 'start', behavior: 'smooth' });
  }, [alerts]);
  const set = <K extends keyof RecoveryForm>(key: K, value: RecoveryForm[K]) => setForm((f) => ({ ...f, [key]: value }));

  const missing = locationMissing(form);
  const changes = diffForm(base, form);
  const dirty = Object.keys(changes).length > 0;
  const serverPlace = fieldError(save.error, 'pain_locations');

  const submit = () => {
    setTried(true);
    if (missing || save.isPending) return;
    if (!dirty) {
      router.push('/postpartum');
      return;
    }
    save.mutate(changes, {
      onSuccess: (r) => {
        void queryClient.invalidateQueries({ queryKey: logKeys.day(r.date) });
        if (r.alerts.length) setAlerts(r.alerts);
        else router.push('/postpartum');
      },
    });
  };

  const status = overview?.status;
  const birth = overview?.profile?.birthDate;
  const visitDate = status && birth && status.daysSinceBirth < status.puerperiumDays
    ? addDays(fromApiDate(birth), status.puerperiumDays)
    : null;

  return (
    <div className="pp-form">
      <div ref={alertsRef} className="flex scroll-mt-4 flex-col gap-[14px] empty:hidden">
      {alerts.map((a) => (
        <UrgentCard
          key={a.key}
          title={a.title}
          action={
            a.action?.type === 'call' ? (
              <a className="nb-btn is-primary is-block" href={`tel:${a.action.number}`}>
                <Icon name="phone" size={18} />
                {a.actionLabel ?? formatNumber(a.action.number, locale)}
              </a>
            ) : undefined
          }
          actions={
            <SecondaryButton onClick={() => router.push('/postpartum')}>{t('goHome')}</SecondaryButton>
          }
        >
          {a.body}
        </UrgentCard>
      ))}
      </div>

      <Card as="section" className="pp-field" aria-labelledby="pp-redflags-title">
        <h2 id="pp-redflags-title" className="pp-field-title">
          {t('redFlags.title')}
        </h2>
        <p className="pp-field-hint">{t('redFlags.hint')}</p>
        {form.temperatureC === null ? (
          <ChipGroup label={t('redFlags.temperature')}>
            <PillChip pressed={false} onPressedChange={() => set('temperatureC', TEMPERATURE_START)}>
              {t('redFlags.addTemperature')}
            </PillChip>
          </ChipGroup>
        ) : (
          <>
            <NumberStepper
              className="pp-stepper"
              label={t('redFlags.temperature')}
              unit={t('redFlags.celsius')}
              value={form.temperatureC}
              onChange={(v) => set('temperatureC', roundTemperature(v, MIN_TEMPERATURE_C, MAX_TEMPERATURE_C))}
              min={MIN_TEMPERATURE_C}
              max={MAX_TEMPERATURE_C}
              step={0.1}
              decrementLabel={t('decrease')}
              incrementLabel={t('increase')}
              locale={locale}
            />
            <SecondaryButton block={false} icon="x" onClick={() => set('temperatureC', null)}>
              {t('redFlags.clearTemperature')}
            </SecondaryButton>
          </>
        )}
        <h3 className="pp-field-sub">{t('redFlags.signs')}</h3>
        <ChipGroup label={t('redFlags.signs')}>
          {RED_FLAGS.map((flag) => (
            <PillChip key={flag} pressed={form[flag] === true} onPressedChange={(on) => set(flag, on)}>
              {t(`redFlags.flags.${flag}`)}
            </PillChip>
          ))}
        </ChipGroup>
      </Card>

      <Card as="section" className="pp-field">
        <h2 className="pp-field-title">{t('lochia')}</h2>
        <ChipGroup label={t('lochia')}>
          {LOCHIA_AMOUNTS.map((v) => (
            <PillChip key={v} pressed={form.lochiaAmount === v} onPressedChange={(on) => set('lochiaAmount', on ? v : null)}>
              {t(`amounts.${v}`)}
            </PillChip>
          ))}
        </ChipGroup>
        <h3 className="pp-field-sub">{t('color')}</h3>
        <ChipGroup label={t('color')}>
          {LOCHIA_COLORS.map((v) => (
            <PillChip key={v} pressed={form.lochiaColor === v} onPressedChange={(on) => set('lochiaColor', on ? v : null)}>
              {t(`colors.${v}`)}
            </PillChip>
          ))}
        </ChipGroup>
        <h3 className="pp-field-sub">{t('pain')}</h3>
        <ChipGroup label={t('pain')}>
          {PAIN_LEVELS.map((v) => (
            <PillChip key={v} pressed={form.painLevel === v} onPressedChange={(on) => set('painLevel', on ? v : null)}>
              {t(`painLevels.${v}`)}
            </PillChip>
          ))}
        </ChipGroup>
        {needsLocation(form.painLevel) && (
          <>
            <h3 className="pp-field-sub">{t('painLocation')}</h3>
            <ChipGroup label={t('painLocation')}>
              {PAIN_LOCATIONS.map((v) => (
                <PillChip
                  key={v}
                  pressed={form.painLocations.includes(v)}
                  onPressedChange={() => set('painLocations', toggleIn(form.painLocations, v))}
                >
                  {t(`painLocations.${v}`)}
                </PillChip>
              ))}
            </ChipGroup>
            {((tried && missing) || serverPlace) && (
              <p className="pp-error" role="alert">
                {serverPlace ?? t('painLocationRequired')}
              </p>
            )}
          </>
        )}
      </Card>

      <Card as="section" className="pp-field">
        <NumberStepper
          className="pp-stepper"
          label={t('feeds')}
          unit={t('times')}
          value={form.feedsCount ?? 0}
          onChange={(v) => set('feedsCount', v)}
          min={0}
          max={MAX_FEEDS}
          decrementLabel={t('decrease')}
          incrementLabel={t('increase')}
          locale={locale}
        />
        <h3 className="pp-field-sub">{t('breasts')}</h3>
        <ChipGroup label={t('breasts')}>
          <PillChip pressed={form.breasts !== null && form.breasts.length === 0} onPressedChange={(on) => set('breasts', on ? [] : null)}>
            {t('normal')}
          </PillChip>
          {BREAST_SYMPTOMS.map((v) => (
            <PillChip
              key={v}
              pressed={form.breasts?.includes(v) ?? false}
              onPressedChange={() => {
                const next = toggleIn(form.breasts ?? [], v);
                set('breasts', next.length ? next : null);
              }}
            >
              {t(`breastSymptoms.${v}`)}
            </PillChip>
          ))}
        </ChipGroup>
        <NumberStepper
          className="pp-stepper"
          label={t('sleep')}
          description={t('sleepHint')}
          unit={t('hours')}
          value={form.sleepHours ?? 0}
          onChange={(v) => set('sleepHours', v)}
          min={0}
          max={MAX_SLEEP_HOURS}
          step={0.5}
          decrementLabel={t('decrease')}
          incrementLabel={t('increase')}
          locale={locale}
        />
      </Card>

      {visitDate && <VisitCard date={visitDate} />}

      {save.isError && !serverPlace && (
        <p className="pp-error" role="alert">
          {getApiSaveErrorMessage(save.error, t('saveError'))}
        </p>
      )}
      <PrimaryButton loading={save.isPending} onClick={submit}>
        {t('save')}
      </PrimaryButton>
    </div>
  );
}

function VisitCard({ date }: { date: Date }) {
  const t = useTranslations('postpartum.recovery');
  const locale = useLocale() as Locale;
  const rtl = useDirection() === 'rtl';
  const p = toParts(date, locale);
  return (
    <Card as="section" className="pp-field" aria-labelledby="pp-visit-title">
      <h2 id="pp-visit-title" className="pp-field-title">
        {t('visitTitle')}
      </h2>
      <Link href="/reminders/appointment/new" className="pp-visit">
        <span className="pp-date nb-tone-warm" aria-hidden>
          <span className="pp-date-day">{formatNumber(p.day, locale)}</span>
          <span className="pp-date-month">{monthName(p.month, locale)}</span>
        </span>
        <span className="pp-visit-text">
          <span className="pp-visit-title">{t('visitBook')}</span>
          <span className="pp-visit-sub">{t('visitBody')}</span>
        </span>
        <Icon name={rtl ? 'chevronLeft' : 'chevronRight'} size={18} className="pp-visit-chev" />
      </Link>
    </Card>
  );
}

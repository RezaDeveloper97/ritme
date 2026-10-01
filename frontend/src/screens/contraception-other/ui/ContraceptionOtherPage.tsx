'use client';

import { useQueryClient } from '@tanstack/react-query';
import { useLocale, useTranslations } from 'next-intl';
import { useState } from 'react';

import {
  type ContraceptionMethod,
  contraceptionKeys,
  daysUntil,
  hasMethodReminders,
  INJECTION_DONE_WINDOW,
  INJECTION_WEEKS,
  METHOD_LOOK,
  type MethodReminder,
  methodReminder,
  methodToPayload,
  useContraception,
  useSaveContraceptionMethod,
} from '@/entities/contraception';
import { useUpdateReminder } from '@/features/manage-reminders';
import { getApiSaveErrorMessage } from '@/shared/api';
import { type Locale, useRouter } from '@/shared/i18n';
import {
  formatDayMonth,
  formatLongDate,
  formatNumber,
  formatYear,
  fromApiDate,
  toApiDate,
  today as todayDate,
  toParts,
} from '@/shared/lib/date';
import {
  Card,
  EmptyState,
  IconCircle,
  ListGroup,
  ListRow,
  PrimaryButton,
  ScreenHeader,
  SecondaryButton,
  SectionTitle,
  Skeleton,
  SkeletonGroup,
  SkyLayer,
  StatusPill,
  Switch,
  UrgentCard,
  InfoNote,
} from '@/shared/ui';

import { DateSheet } from './DateSheet';

/**
 * «یادآورهای روش پیشگیری» (`/contraception/other`, CB-CONTRA-03,
 * nbl_Contra_Other): the dates and done states of a long-acting method — IUD
 * (monthly string check switch, replacement year, 6-week visit «انجام شد»),
 * 3-month injection (countdown, «امروز تزریق کردم») or implant (removal /
 * replacement date). The board stacks all three; a user has one method, so
 * only hers is shown. Every reminder is a care `reminders` row the server
 * keeps in sync with the method (CB-CONTRA-01): the switch is
 * `PUT /reminders/{id}`, dates and «انجام شد» re-save the method.
 * Back-header screen: no bottom nav.
 */
export function ContraceptionOtherPage() {
  const t = useTranslations('contraception');
  const router = useRouter();
  const query = useContraception();
  const data = query.data;
  const method = data?.method ?? null;

  let body;
  if (query.isPending) {
    body = (
      <SkeletonGroup label={t('other.loading')} className="ctr-skel">
        <Skeleton width="short" />
        <Skeleton shape="card" />
        <Skeleton shape="card" />
      </SkeletonGroup>
    );
  } else if (query.isError || !data) {
    body = (
      <EmptyState
        icon="bellRing"
        title={t('error.title')}
        body={t('error.body')}
        action={
          <PrimaryButton icon="refresh" loading={query.isFetching} onClick={() => void query.refetch()}>
            {t('error.retry')}
          </PrimaryButton>
        }
      />
    );
  } else if (!method) {
    body = (
      <EmptyState
        icon="shield"
        title={t('empty.title')}
        body={t('empty.body')}
        action={<PrimaryButton onClick={() => router.push('/contraception/setup')}>{t('empty.cta')}</PrimaryButton>}
      />
    );
  } else if (!hasMethodReminders(method.method)) {
    body = (
      <EmptyState
        icon="bellRing"
        title={t('other.noReminders.title')}
        body={t('other.noReminders.body')}
        action={
          <PrimaryButton onClick={() => router.push('/contraception/setup')}>{t('other.noReminders.cta')}</PrimaryButton>
        }
      />
    );
  } else if (method.method === 'injection') {
    body = <InjectionSection method={method} />;
  } else if (method.method === 'implant') {
    body = <ImplantSection method={method} />;
  } else {
    body = <IudSection method={method} reminders={data.reminders} />;
  }

  return (
    <div className="view ctr-page">
      <SkyLayer />
      <div className="scroll ctr-scroll">
        <ScreenHeader
          title={t('other.title')}
          onBack={() => router.push('/contraception')}
          backLabel={t('other.back')}
        />
        {body}
      </div>
    </div>
  );
}

function SaveError({ error }: { error: unknown }) {
  const t = useTranslations('contraception.other');
  return (
    <p className="ctr-error is-inline" role="alert">
      {getApiSaveErrorMessage(error, t('saveError'))}
    </p>
  );
}

/** «۱۸ روز دیگر» / «امروز» / «۳ روز گذشته». */
function useDaysLabel() {
  const t = useTranslations('contraception.other.days');
  const locale = useLocale() as Locale;
  return (days: number) => {
    if (days === 0) return t('today');
    const count = Math.abs(days);
    const n = formatNumber(count, locale);
    return days > 0 ? t('left', { count, n }) : t('ago', { count, n });
  };
}

function IudSection({ method, reminders }: { method: ContraceptionMethod; reminders: MethodReminder[] }) {
  const t = useTranslations('contraception');
  const locale = useLocale() as Locale;
  const router = useRouter();
  const queryClient = useQueryClient();
  const save = useSaveContraceptionMethod();
  const toggle = useUpdateReminder();
  const look = METHOD_LOOK[method.method];
  const stringCheck = methodReminder(reminders, 'iud_string_check');

  const setStringCheck = (on: boolean) => {
    if (stringCheck) {
      toggle.mutate(
        { id: String(stringCheck.reminderId), isActive: on },
        { onSuccess: () => void queryClient.invalidateQueries({ queryKey: contraceptionKeys.overview() }) },
      );
    } else if (on) {
      // The user deleted it in the care screens: re-saving the method re-creates it.
      save.mutate(methodToPayload(method));
    }
  };

  let rows;
  if (!method.insertedOn) {
    rows = (
      <ListRow
        icon="calendar"
        iconTone="brand"
        title={t('other.iud.setInserted')}
        description={t('other.iud.setInsertedSub')}
        onClick={() => router.push('/contraception/setup')}
      />
    );
  } else {
    const insertedYear = toParts(fromApiDate(method.insertedOn), locale).year;
    const replaceYear = method.iudReplaceOn ? toParts(fromApiDate(method.iudReplaceOn), locale).year : null;
    rows = (
      <>
        <ListRow
          id="ctro-string"
          icon="shield"
          iconTone={look.tone}
          title={t('other.iud.string')}
          description={t('other.iud.stringSub')}
          trailing={
            <Switch
              checked={stringCheck?.isActive ?? false}
              onCheckedChange={setStringCheck}
              labelledBy="ctro-string-title"
              disabled={toggle.isPending || save.isPending}
            />
          }
        />
        <ListRow
          icon="calendar"
          iconTone="brand"
          title={t('other.iud.replace')}
          description={
            method.iudLifetimeYears
              ? t('other.iud.replaceSub', {
                  year: formatYear(insertedYear, locale),
                  years: formatNumber(method.iudLifetimeYears, locale),
                })
              : undefined
          }
          trailing={
            replaceYear ? (
              <span className="ctro-tag">
                <span className="sr-only">{t('other.iud.replaceYear', { year: formatYear(replaceYear, locale) })}</span>
                <span aria-hidden>{formatYear(replaceYear, locale)}</span>
              </span>
            ) : null
          }
        />
        <ListRow
          icon="stetho"
          iconTone="data"
          title={t('other.iud.followup')}
          description={
            method.followupOn
              ? t('other.iud.followupSub', { date: formatDayMonth(fromApiDate(method.followupOn), locale) })
              : undefined
          }
          trailing={
            method.followupDone ? (
              <StatusPill tone="success" icon="check">
                {t('other.iud.followupDone')}
              </StatusPill>
            ) : (
              <button
                type="button"
                className="ctro-act nb-tone-data"
                aria-label={t('other.iud.followupMarkLabel')}
                disabled={save.isPending}
                onClick={() => save.mutate({ ...methodToPayload(method), followup_done: true })}
              >
                {t('other.iud.followupMark')}
              </button>
            )
          }
        />
      </>
    );
  }

  return (
    <>
      <section className="ctro-section" aria-labelledby="ctro-iud-title">
        <SectionTitle id="ctro-iud-title" title={t(`setup.methods.${method.method}`)} />
        <ListGroup className="ctro-list">{rows}</ListGroup>
        {save.isError ? <SaveError error={save.error} /> : null}
        {toggle.isError ? <SaveError error={toggle.error} /> : null}
      </section>
      <UrgentCard variant="note" title={t('other.iud.note')} className="ctro-danger" />
    </>
  );
}

function InjectionSection({ method }: { method: ContraceptionMethod }) {
  const t = useTranslations('contraception');
  const locale = useLocale() as Locale;
  const router = useRouter();
  const save = useSaveContraceptionMethod();
  const daysLabel = useDaysLabel();
  const look = METHOD_LOOK.injection;
  const today = toApiDate(todayDate());

  let content;
  if (!method.nextInjectionOn) {
    content = (
      <ListGroup className="ctro-list">
        <ListRow
          icon={look.icon}
          iconTone={look.tone}
          title={t('other.injection.set')}
          description={t('other.injection.setSub')}
          onClick={() => router.push('/contraception/setup')}
        />
      </ListGroup>
    );
  } else {
    const days = daysUntil(method.nextInjectionOn, today);
    const injectedToday = method.injectedOn === today;
    content = (
      <Card className="ctro-inj">
        <div className="ctro-inj-head">
          <IconCircle icon={look.icon} tone={look.tone} size="lg" />
          <div className="ctro-inj-text">
            <p className="ctro-inj-cap">{t('other.injection.next')}</p>
            <p className={days < 0 ? 'ctro-inj-num is-late' : 'ctro-inj-num'}>{daysLabel(days)}</p>
            <p className="ctro-inj-sub">
              {t('other.injection.sub', {
                date: formatDayMonth(fromApiDate(method.nextInjectionOn), locale),
                weeks: formatNumber(INJECTION_WEEKS, locale),
              })}
            </p>
          </div>
        </div>
        {days <= INJECTION_DONE_WINDOW && !injectedToday ? (
          <SecondaryButton
            icon="check"
            loading={save.isPending}
            onClick={() => save.mutate({ method: 'injection', injected_on: today })}
          >
            {t('other.injection.done')}
          </SecondaryButton>
        ) : null}
        {save.isError ? <SaveError error={save.error} /> : null}
      </Card>
    );
  }

  return (
    <>
      <section className="ctro-section" aria-labelledby="ctro-inj-title">
        <SectionTitle id="ctro-inj-title" title={t('setup.methods.injection')} />
        {content}
      </section>
      <InfoNote className="ctro-note">{t('other.note')}</InfoNote>
    </>
  );
}

function ImplantSection({ method }: { method: ContraceptionMethod }) {
  const t = useTranslations('contraception');
  const locale = useLocale() as Locale;
  const save = useSaveContraceptionMethod();
  const daysLabel = useDaysLabel();
  const [open, setOpen] = useState(false);
  const look = METHOD_LOOK.implant;
  const today = toApiDate(todayDate());
  const replaceOn = method.replaceOn;

  return (
    <>
      <section className="ctro-section" aria-labelledby="ctro-imp-title">
        <SectionTitle id="ctro-imp-title" title={t('setup.methods.implant')} />
        <ListGroup className="ctro-list">
          <ListRow
            icon={look.icon}
            iconTone={look.tone}
            title={t('other.implant.replace')}
            description={
              replaceOn
                ? t('other.implant.replaceOn', {
                    date: formatLongDate(fromApiDate(replaceOn), locale),
                    left: daysLabel(daysUntil(replaceOn, today)),
                  })
                : t('other.implant.replaceSub')
            }
            trailing={
              <button
                type="button"
                className="ctro-act nb-tone-brand"
                aria-label={t('other.implant.setLabel')}
                onClick={() => setOpen(true)}
              >
                {replaceOn ? t('other.implant.change') : t('other.implant.set')}
              </button>
            }
          />
        </ListGroup>
      </section>
      <InfoNote className="ctro-note">{t('other.note')}</InfoNote>
      {open ? (
        <DateSheet
          title={t('other.implant.replace')}
          value={replaceOn}
          busy={save.isPending}
          error={save.isError ? getApiSaveErrorMessage(save.error, t('other.saveError')) : null}
          onClose={() => setOpen(false)}
          onPick={(date) =>
            save.mutate({ ...methodToPayload(method), replace_on: date }, { onSuccess: () => setOpen(false) })
          }
        />
      ) : null}
    </>
  );
}

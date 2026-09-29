'use client';

import { useQueryClient } from '@tanstack/react-query';
import clsx from 'clsx';
import { useSearchParams } from 'next/navigation';
import { useLocale, useTranslations } from 'next-intl';
import { useEffect, useState } from 'react';

import {
  APPOINTMENT_KINDS,
  APPOINTMENT_TOPICS,
  REMIND_BEFORE,
  type Appointment,
  type AppointmentKind,
  useAppointment,
} from '@/entities/care-reminder';
import { pregnancyKeys } from '@/entities/pregnancy';
import { useCreateAppointment, useUpdateAppointment } from '@/features/manage-appointment';
import { getApiLimitMessage } from '@/shared/api';
import { Link, type Locale, useDirection, useRouter } from '@/shared/i18n';
import {
  type DateParts,
  formatNumber,
  fromApiDate,
  monthName,
  partsToDate,
  toApiDate,
  toParts,
} from '@/shared/lib/date';
import { clearHandoff, readHandoff } from '@/shared/lib/handoff';
import { AppSheet } from '@/shared/sheet';
import { CalendarPicker, Icon, type IconName, WheelPicker } from '@/shared/ui';

import {
  type AppointmentFormState,
  type FormError,
  type AppointmentPrefill,
  formFromAppointment,
  formFromPrefill,
  isoDay,
  prefillFromHandoff,
  prepFromText,
  returnPathFor,
  validateForm,
} from '../model/form';

const KIND_ICON: Record<AppointmentKind, IconName> = {
  in_person: 'doctor',
  phone: 'phone',
  online: 'video',
};

const HOURS = Array.from({ length: 24 }, (_, i) => String(i).padStart(2, '0'));
const MINUTES = Array.from({ length: 12 }, (_, i) => String(i * 5).padStart(2, '0'));

interface Props {
  /** `?kind=` on the "new" route; the rest comes from the `?prefill=<id>` handoff. */
  prefill?: AppointmentPrefill;
  /** Present on `/reminders/appointment/[id]/edit`. */
  id?: number;
}

/**
 * Add / edit appointment (`v13_AddAppointment`): kind, who, specialty, topic,
 * short description, date + time (locale calendar), place, remind-before,
 * add-to-calendar and the prep list (one line = one checklist item).
 *
 * Privacy (§11): nothing typed here is logged; it goes to `/care/appointments` only.
 */
export function AppointmentFormPage({ prefill, id }: Props) {
  const existing = useAppointment(id ?? null);
  if (id !== undefined && !existing.data) {
    return <FormShell id={id} status={existing.isError ? 'error' : 'loading'} onRetry={() => void existing.refetch()} />;
  }
  return <AppointmentForm key={id ?? 'new'} prefill={prefill} existing={existing.data ?? null} />;
}

function FormShell({ id, status, onRetry }: { id: number; status: 'loading' | 'error'; onRetry: () => void }) {
  const t = useTranslations('care');
  return (
    <div className="view rmd-page">
      <div className="scroll">
        <FormHeader title={t('appointmentForm.editTitle')} backHref={`/reminders/appointment/${id}`} />
        <div className="rmd-body">
          {status === 'loading' ? (
            <div className="rmd-state" aria-busy="true" aria-label={t('loading')}>
              <span className="skeleton-line rmd-row-skel" />
            </div>
          ) : (
            <div className="rmd-state">
              <p className="rmd-empty">{t('loadError')}</p>
              <button type="button" className="rmd-retry" onClick={onRetry}>
                {t('retry')}
              </button>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

function FormHeader({ title, backHref, sub }: { title: string; backHref: string; sub?: string }) {
  const t = useTranslations('care');
  const dir = useDirection();
  return (
    <header className="rmd-hdr">
      <Link href={backHref} className="rmd-hdr-btn" aria-label={t('back')}>
        <Icon name={dir === 'rtl' ? 'chevronRight' : 'chevronLeft'} size={20} strokeWidth={1.8} />
      </Link>
      <div className="rmd-hdr-text">
        <h1 className="rmd-hdr-title">{title}</h1>
        {sub && <p className="rmd-hdr-sub">{sub}</p>}
      </div>
      <span className="rmd-hdr-btn invisible" aria-hidden />
    </header>
  );
}

function AppointmentForm({ prefill, existing }: { prefill?: AppointmentPrefill; existing: Appointment | null }) {
  const t = useTranslations('care.appointmentForm');
  const tc = useTranslations('care');
  const locale = useLocale() as Locale;
  const router = useRouter();
  const searchParams = useSearchParams();
  const queryClient = useQueryClient();
  const create = useCreateAppointment();
  const update = useUpdateAppointment();

  const [form, setForm] = useState<AppointmentFormState>(() =>
    existing ? formFromAppointment(existing) : formFromPrefill(prefill ?? {}, isoDay(new Date())),
  );
  const [error, setError] = useState<FormError | null>(null);
  const [saveError, setSaveError] = useState<string | null>(null);
  const [picker, setPicker] = useState<'date' | 'time' | null>(null);
  const [draftDate, setDraftDate] = useState<DateParts | null>(null);
  const [draftTime, setDraftTime] = useState<[number, number]>([10, 0]);

  const set = <K extends keyof AppointmentFormState>(key: K, value: AppointmentFormState[K]) => {
    setForm((f) => ({ ...f, [key]: value }));
    setError(null);
  };

  // A booking from the checkups card or the pregnancy care plan: its title,
  // topic, date and care-plan key were handed over in memory / sessionStorage
  // under `?prefill=<id>` — never in the URL (audit M3-M7 #3). Read after mount:
  // the server render can't see sessionStorage.
  useEffect(() => {
    if (existing) return;
    const handed = readHandoff(searchParams.get('prefill'));
    if (handed) setForm(formFromPrefill({ ...prefill, ...prefillFromHandoff(handed) }, isoDay(new Date())));
    // Once, on mount.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    if (picker === 'date') setDraftDate(form.date ? toParts(fromApiDate(form.date), locale) : null);
    if (picker === 'time') {
      const [h = '10', m = '00'] = form.time ? form.time.split(':') : [];
      setDraftTime([Number(h), Math.round(Number(m) / 5) % 12]);
    }
    // Only when a picker opens.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [picker]);

  // `?return_to=` (allow-listed) or a care-plan booking → back to that screen after saving (9b).
  const returnTo = returnPathFor(searchParams.get('return_to'), !existing, form.careItemKey);
  const pending = create.isPending || update.isPending;
  const dateParts = form.date ? toParts(fromApiDate(form.date), locale) : null;
  const placeLabel =
    form.kind === 'online' ? t('placeOnline') : form.kind === 'phone' ? t('placePhone') : t('place');
  const placeHint =
    form.kind === 'online'
      ? t('placeOnlinePlaceholder')
      : form.kind === 'phone'
        ? t('placePhonePlaceholder')
        : t('placePlaceholder');

  const submit = () => {
    const problem = validateForm(form);
    if (problem) {
      setError(problem);
      return;
    }
    setSaveError(null);
    const input = {
      kind: form.kind,
      withWhom: form.withWhom,
      specialty: form.specialty,
      topic: form.topic,
      title: form.title,
      date: form.date,
      time: form.time,
      location: form.location,
      remindBefore: form.remindBefore,
      addToCalendar: form.addToCalendar,
      prep: prepFromText(form.prepText, existing?.prep ?? []),
      // Links a pregnancy care-plan item (→ `booked`); edit keeps the stored key.
      careItemKey: form.careItemKey || null,
    };
    // A per-user cap (422 limit_reached) shows the server's localized message.
    const onError = (error: unknown) => setSaveError(getApiLimitMessage(error) ?? tc('saveError'));
    const done = (detailHref: string) => {
      clearHandoff();
      if (!returnTo) return router.push(detailHref);
      // The calendar and Today show this visit (booked care-plan state, next visit).
      void queryClient.invalidateQueries({ queryKey: pregnancyKeys.v2.calendarAll() });
      void queryClient.invalidateQueries({ queryKey: pregnancyKeys.v2.today() });
      router.replace(returnTo);
    };
    if (existing) {
      update.mutate(
        { id: existing.id, patch: input },
        { onSuccess: () => done(`/reminders/appointment/${existing.id}`), onError },
      );
    } else {
      create.mutate(input, {
        onSuccess: (created) => done(`/reminders/appointment/${created.id}`),
        onError,
      });
    }
  };

  return (
    <div className="view rmd-page">
      <div className="scroll">
        <FormHeader
          title={existing ? t('editTitle') : t('title')}
          sub={t('subtitle')}
          backHref={returnTo ?? (existing ? `/reminders/appointment/${existing.id}` : '/reminders?tab=appointments')}
        />

        <form
          className="rmd-body"
          noValidate
          onSubmit={(event) => {
            event.preventDefault();
            submit();
          }}
        >
          <div className="cfm-kinds" role="radiogroup" aria-label={t('subtitle')}>
            {APPOINTMENT_KINDS.map((kind) => (
              <button
                key={kind}
                type="button"
                role="radio"
                aria-checked={form.kind === kind}
                className="cfm-kind"
                onClick={() => set('kind', kind)}
              >
                <Icon name={KIND_ICON[kind]} size={20} strokeWidth={1.8} />
                {t(`kinds.${kind}`)}
              </button>
            ))}
          </div>

          {/* ── Card 1: who and what ── */}
          <section className="cfm-card">
            <label className="cfm-group">
              <span className="cfm-label">{t('with')}</span>
              <span className="cfm-field">
                <Icon name="user" size={18} strokeWidth={1.8} />
                <input
                  value={form.withWhom}
                  placeholder={t('withPlaceholder')}
                  aria-invalid={error === 'with'}
                  aria-describedby={error === 'with' ? 'apf-err-with' : undefined}
                  onChange={(e) => set('withWhom', e.target.value)}
                />
              </span>
              {error === 'with' && (
                <p className="cfm-error" id="apf-err-with" role="alert">
                  {t('errors.with')}
                </p>
              )}
            </label>
            <label className="cfm-group">
              <span className="cfm-label">{t('specialty')}</span>
              <span className="cfm-field">
                <input
                  value={form.specialty}
                  placeholder={t('specialtyPlaceholder')}
                  onChange={(e) => set('specialty', e.target.value)}
                />
              </span>
            </label>
            <div className="cfm-group is-loose">
              <span className="cfm-label" id="apf-topic">{t('topic')}</span>
              <div className="cfm-chips" role="radiogroup" aria-labelledby="apf-topic">
                {APPOINTMENT_TOPICS.map((topic) => (
                  <button
                    key={topic}
                    type="button"
                    role="radio"
                    aria-checked={form.topic === topic}
                    className="cfm-chip"
                    onClick={() => set('topic', topic)}
                  >
                    {t(`topics.${topic}`)}
                  </button>
                ))}
              </div>
            </div>
            <label className="cfm-group">
              <span className="cfm-label">{t('shortDescription')}</span>
              <span className="cfm-field">
                <Icon name="note" size={18} strokeWidth={1.8} />
                <input
                  value={form.title}
                  placeholder={t('shortDescriptionPlaceholder')}
                  onChange={(e) => set('title', e.target.value)}
                />
              </span>
            </label>
          </section>

          {/* ── Card 2: when and where ── */}
          <section className="cfm-card">
            <div className="cfm-two">
              <div className="cfm-group">
                <span className="cfm-label" id="apf-date">{t('date')}</span>
                <button
                  type="button"
                  className="cfm-pick"
                  aria-labelledby="apf-date apf-date-v"
                  onClick={() => setPicker('date')}
                >
                  <span id="apf-date-v" className={clsx('cfm-pick-v', !dateParts && 'is-empty')}>
                    {dateParts
                      ? `${formatNumber(dateParts.day, locale)} ${monthName(dateParts.month, locale)}`
                      : t('pickDate')}
                  </span>
                  <Icon name="calendar" size={18} strokeWidth={1.8} />
                </button>
              </div>
              <div className="cfm-group">
                <span className="cfm-label" id="apf-time">{t('time')}</span>
                <button
                  type="button"
                  className="cfm-pick"
                  aria-labelledby="apf-time apf-time-v"
                  onClick={() => setPicker('time')}
                >
                  <span id="apf-time-v" className={clsx('cfm-pick-v', !form.time && 'is-empty')} dir="ltr">
                    {form.time ? formatNumber(form.time, locale) : t('pickTime')}
                  </span>
                  <Icon name="clock" size={18} strokeWidth={1.8} />
                </button>
              </div>
            </div>
            {(error === 'date' || error === 'time') && (
              <p className="cfm-error" role="alert">{t(`errors.${error}`)}</p>
            )}
            <label className="cfm-group">
              <span className="cfm-label">{placeLabel}</span>
              <span className="cfm-field">
                <Icon name={form.kind === 'online' ? 'video' : form.kind === 'phone' ? 'phone' : 'mapPin'} size={18} strokeWidth={1.8} />
                <input
                  value={form.location}
                  placeholder={placeHint}
                  inputMode={form.kind === 'phone' ? 'tel' : form.kind === 'online' ? 'url' : 'text'}
                  dir={form.kind === 'in_person' ? undefined : 'auto'}
                  onChange={(e) => set('location', e.target.value)}
                />
              </span>
            </label>
          </section>

          {/* ── Card 3: reminder and prep ── */}
          <section className="cfm-card is-tight">
            <div className="cfm-group is-loose">
              <span className="cfm-label" id="apf-remind">{t('remindBefore')}</span>
              <div className="cfm-chips" role="radiogroup" aria-labelledby="apf-remind">
                {REMIND_BEFORE.map((value) => (
                  <button
                    key={value}
                    type="button"
                    role="radio"
                    aria-checked={form.remindBefore === value}
                    className="cfm-chip"
                    onClick={() => set('remindBefore', value)}
                  >
                    {t(`remindOptions.${value}`)}
                  </button>
                ))}
              </div>
            </div>
            <div className="cfm-set">
              <span className="cfm-set-body">
                <span className="cfm-set-t">{t('addToCalendar')}</span>
                <span className="cfm-set-s">{t('addToCalendarHint')}</span>
              </span>
              <button
                type="button"
                role="switch"
                aria-checked={form.addToCalendar}
                aria-label={t('addToCalendar')}
                className="rmd-switch"
                onClick={() => set('addToCalendar', !form.addToCalendar)}
              >
                <span className="rmd-switch-knob" />
              </button>
            </div>
            <label className="cfm-group">
              <span className="cfm-label">{t('prep')}</span>
              <textarea
                className="cfm-textarea"
                rows={3}
                value={form.prepText}
                placeholder={t('prepPlaceholder')}
                aria-describedby="apf-prep-hint"
                onChange={(e) => set('prepText', e.target.value)}
              />
              <span className="cfm-caption" id="apf-prep-hint">{t('prepHint')}</span>
            </label>
          </section>

          {saveError && <p className="cfm-error is-center" role="alert">{saveError}</p>}

          <button type="submit" className="rmd-cta" disabled={pending}>
            {pending ? t('saving') : t('save')}
          </button>
        </form>
      </div>

      <AppSheet
        open={picker === 'date'}
        onClose={() => setPicker(null)}
        size="half"
        title={t('date')}
        footer={
          <button
            type="button"
            className="btn btn-primary"
            disabled={!draftDate}
            onClick={() => {
              if (draftDate) set('date', toApiDate(partsToDate(draftDate, locale)));
              setPicker(null);
            }}
          >
            {t('confirm')}
          </button>
        }
      >
        <CalendarPicker value={draftDate} onSelect={setDraftDate} />
      </AppSheet>

      <AppSheet
        open={picker === 'time'}
        onClose={() => setPicker(null)}
        size="half"
        title={t('time')}
        footer={
          <button
            type="button"
            className="btn btn-primary"
            onClick={() => {
              set('time', `${HOURS[draftTime[0]]}:${MINUTES[draftTime[1]]}`);
              setPicker(null);
            }}
          >
            {t('confirm')}
          </button>
        }
      >
        <div className="jdw-row" dir="ltr">
          <WheelPicker
            id="apf-hour"
            items={HOURS.map((h) => formatNumber(h, locale))}
            selectedIndex={draftTime[0]}
            onChange={(i) => setDraftTime(([, m]) => [i, m])}
          />
          <WheelPicker
            id="apf-minute"
            items={MINUTES.map((m) => formatNumber(m, locale))}
            selectedIndex={draftTime[1]}
            onChange={(i) => setDraftTime(([h]) => [h, i])}
          />
        </div>
      </AppSheet>
    </div>
  );
}

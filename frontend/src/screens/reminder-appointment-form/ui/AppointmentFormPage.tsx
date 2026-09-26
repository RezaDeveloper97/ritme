'use client';

import clsx from 'clsx';
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
import { useCreateAppointment, useUpdateAppointment } from '@/features/manage-appointment';
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
import { AppSheet } from '@/shared/sheet';
import { CalendarPicker, Icon, type IconName, WheelPicker } from '@/shared/ui';

import {
  type AppointmentFormState,
  type FormError,
  type AppointmentPrefill,
  formFromAppointment,
  formFromPrefill,
  isoDay,
  prepFromText,
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
  /** `?kind=&title=&date=&care_item_key=` on the "new" route. */
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
  const create = useCreateAppointment();
  const update = useUpdateAppointment();

  const [form, setForm] = useState<AppointmentFormState>(() =>
    existing ? formFromAppointment(existing) : formFromPrefill(prefill ?? {}, isoDay(new Date())),
  );
  const [error, setError] = useState<FormError | null>(null);
  const [saveFailed, setSaveFailed] = useState(false);
  const [picker, setPicker] = useState<'date' | 'time' | null>(null);
  const [draftDate, setDraftDate] = useState<DateParts | null>(null);
  const [draftTime, setDraftTime] = useState<[number, number]>([10, 0]);

  const set = <K extends keyof AppointmentFormState>(key: K, value: AppointmentFormState[K]) => {
    setForm((f) => ({ ...f, [key]: value }));
    setError(null);
  };

  useEffect(() => {
    if (picker === 'date') setDraftDate(form.date ? toParts(fromApiDate(form.date), locale) : null);
    if (picker === 'time') {
      const [h = '10', m = '00'] = form.time ? form.time.split(':') : [];
      setDraftTime([Number(h), Math.round(Number(m) / 5) % 12]);
    }
    // Only when a picker opens.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [picker]);

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
    setSaveFailed(false);
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
    const onError = () => setSaveFailed(true);
    if (existing) {
      update.mutate(
        { id: existing.id, patch: input },
        { onSuccess: () => router.push(`/reminders/appointment/${existing.id}`), onError },
      );
    } else {
      create.mutate(input, {
        onSuccess: (created) => router.push(`/reminders/appointment/${created.id}`),
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
          backHref={existing ? `/reminders/appointment/${existing.id}` : '/reminders?tab=appointments'}
        />

        <form
          className="rmd-body"
          noValidate
          onSubmit={(event) => {
            event.preventDefault();
            submit();
          }}
        >
          <div className="rmd-tabs" role="radiogroup" aria-label={t('subtitle')}>
            {APPOINTMENT_KINDS.map((kind) => (
              <button
                key={kind}
                type="button"
                role="radio"
                aria-checked={form.kind === kind}
                className={clsx('rmd-tab', form.kind === kind && 'on')}
                onClick={() => set('kind', kind)}
              >
                <Icon name={KIND_ICON[kind]} size={16} strokeWidth={1.8} /> {t(`kinds.${kind}`)}
              </button>
            ))}
          </div>

          <section className="card fld-card">
            <label className="fld-label">
              <span className="fld-label-t">{t('with')}</span>
              <span className="field">
                <input
                  value={form.withWhom}
                  placeholder={t('withPlaceholder')}
                  aria-invalid={error === 'with'}
                  onChange={(e) => set('withWhom', e.target.value)}
                />
              </span>
            </label>
            {error === 'with' && <p className="rem-form-error" role="alert">{t('errors.with')}</p>}
            <div className="fld-row" />
            <label className="fld-label">
              <span className="fld-label-t">{t('specialty')}</span>
              <span className="field">
                <input
                  value={form.specialty}
                  placeholder={t('specialtyPlaceholder')}
                  onChange={(e) => set('specialty', e.target.value)}
                />
              </span>
            </label>
          </section>

          <section className="card fld-card">
            <span className="fld-label-t" id="apf-topic">{t('topic')}</span>
            <div className="fld-chips" role="radiogroup" aria-labelledby="apf-topic">
              {APPOINTMENT_TOPICS.map((topic) => (
                <button
                  key={topic}
                  type="button"
                  role="radio"
                  aria-checked={form.topic === topic}
                  className={clsx('chip', form.topic === topic && 'on')}
                  onClick={() => set('topic', topic)}
                >
                  {t(`topics.${topic}`)}
                </button>
              ))}
            </div>
            <div className="fld-row" />
            <label className="fld-label">
              <span className="fld-label-t">{t('shortDescription')}</span>
              <span className="field">
                <input
                  value={form.title}
                  placeholder={t('shortDescriptionPlaceholder')}
                  onChange={(e) => set('title', e.target.value)}
                />
              </span>
            </label>
          </section>

          <section className="card fld-card">
            <div className="fld-row">
              <button
                type="button"
                className="rmd-row rmd-row-link"
                onClick={() => setPicker('date')}
              >
                <span className="rmd-date" aria-hidden>
                  {dateParts ? (
                    <>
                      <span className="rmd-date-day">{formatNumber(dateParts.day, locale)}</span>
                      <span className="rmd-date-month">{monthName(dateParts.month, locale)}</span>
                    </>
                  ) : (
                    <Icon name="calendar" size={20} />
                  )}
                </span>
                <span className="rmd-row-body">
                  <span className="rmd-row-meta">{t('date')}</span>
                  <span className="rmd-row-title">
                    {dateParts
                      ? `${formatNumber(dateParts.day, locale)} ${monthName(dateParts.month, locale)} ${formatNumber(dateParts.year, locale)}`
                      : t('pickDate')}
                  </span>
                </span>
              </button>
              <button
                type="button"
                className="rmd-row rmd-row-link"
                onClick={() => setPicker('time')}
              >
                <span className="rmd-date is-teal" aria-hidden>
                  {form.time ? (
                    <span className="rmd-date-day">{formatNumber(form.time, locale)}</span>
                  ) : (
                    <Icon name="clock" size={20} />
                  )}
                </span>
                <span className="rmd-row-body">
                  <span className="rmd-row-meta">{t('time')}</span>
                  <span className="rmd-row-title">{form.time ? formatNumber(form.time, locale) : t('pickTime')}</span>
                </span>
              </button>
            </div>
            {(error === 'date' || error === 'time') && (
              <p className="rem-form-error" role="alert">{t(`errors.${error}`)}</p>
            )}
            <label className="fld-label">
              <span className="fld-label-t">{placeLabel}</span>
              <span className="field">
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

          <section className="card fld-card">
            <span className="fld-label-t" id="apf-remind">{t('remindBefore')}</span>
            <div className="fld-chips" role="radiogroup" aria-labelledby="apf-remind">
              {REMIND_BEFORE.map((value) => (
                <button
                  key={value}
                  type="button"
                  role="radio"
                  aria-checked={form.remindBefore === value}
                  className={clsx('chip', form.remindBefore === value && 'on')}
                  onClick={() => set('remindBefore', value)}
                >
                  {t(`remindOptions.${value}`)}
                </button>
              ))}
            </div>
            <div className="fld-row">
              <span className="rmd-row-body">
                <span className="fld-row-label">{t('addToCalendar')}</span>
                <span className="rmd-row-meta">{t('addToCalendarHint')}</span>
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
          </section>

          <section className="card fld-card">
            <label className="fld-label">
              <span className="fld-label-t">{t('prep')}</span>
              <textarea
                className="field lfr-textarea fld-textarea"
                rows={4}
                value={form.prepText}
                placeholder={t('prepPlaceholder')}
                onChange={(e) => set('prepText', e.target.value)}
              />
            </label>
            <p className="rmd-row-meta">{t('prepHint')}</p>
          </section>

          {saveFailed && <p className="rem-form-error" role="alert">{tc('saveError')}</p>}

          <button type="submit" className="rmd-cta" disabled={pending}>
            <Icon name="check" size={18} strokeWidth={2.2} />
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

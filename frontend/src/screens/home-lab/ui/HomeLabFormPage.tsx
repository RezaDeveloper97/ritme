'use client';

import clsx from 'clsx';
import { useFormatter, useLocale, useTranslations } from 'next-intl';
import { useMemo, useState } from 'react';

import { LAB_WINDOWS, useCreateLabRequest, useLabServices } from '@/entities/partner-lab';
import { getApiErrorMessage } from '@/shared/api';
import { type Locale, useRouter } from '@/shared/i18n';
import { addDays, formatLongDate, toApiDate, today } from '@/shared/lib/date';
import { pilot, pilotOption } from '@/shared/pilot';
import { Card, Checkbox, EmptyState, Icon, InfoNote, PrimaryButton, ScreenHeader, Skeleton, SkeletonGroup, SkyLayer } from '@/shared/ui';

import {
  ADDRESS_MAX,
  CITY_MAX,
  DAY_CHOICES,
  emptyLabForm,
  type LabFormErrors,
  type LabFormState,
  NAME_MAX,
  NOTE_MAX,
  validateLabForm,
} from '../model/form';

/**
 * `/services/home-lab/new?service=ID` — the booking form: name, phone, city / address, preferred day + window, a
 * note with a «don't write medical details» warning, the consent line naming the partner and the optional consent
 * to import the result. Personal fields go to the API only (never to storage or analytics).
 */
export function HomeLabFormPage({ serviceId }: { serviceId: number | null }) {
  const t = useTranslations('homeLab');
  const fmt = useFormatter();
  const locale = useLocale() as Locale;
  const router = useRouter();
  const services = useLabServices();
  const create = useCreateLabRequest();

  const days = useMemo(() => Array.from({ length: DAY_CHOICES }, (_, i) => addDays(today(), i)), []);
  const [state, setState] = useState<LabFormState>(() => emptyLabForm(toApiDate(today())));
  const [errors, setErrors] = useState<LabFormErrors>({});
  const [formError, setFormError] = useState<string | null>(null);

  const set = <K extends keyof LabFormState>(key: K, value: LabFormState[K]) => setState((s) => ({ ...s, [key]: value }));
  const back = () => router.push('/services/home-lab');
  const service = services.data?.services.find((s) => s.id === serviceId) ?? services.data?.services[0] ?? null;

  const onSubmit = () => {
    setFormError(null);
    const found = validateLabForm(state);
    setErrors(found);
    if (Object.keys(found).length > 0 || !service) return;
    create.mutate(
      {
        serviceId: service.id,
        fullName: state.fullName.trim(),
        phone: state.phone,
        city: state.city.trim(),
        address: state.address.trim(),
        preferredDate: state.day,
        window: state.window,
        note: state.note,
        consentShare: state.consentShare,
        consentImport: state.consentImport,
      },
      {
        onSuccess: (r) => router.push(`/services/home-lab/${r.id}`),
        onError: (e) => setFormError(getApiErrorMessage(e) ?? t('form.saveError')),
      },
    );
  };

  const err = (k: keyof LabFormErrors) =>
    errors[k] ? (
      <p id={`hlb-err-${k}`} role="alert" className="hlb-err">
        {t(`form.${errors[k]}`)}
      </p>
    ) : null;

  let body;
  if (services.isPending) {
    body = (
      <SkeletonGroup label={t('loading')} className="hlb-stack">
        <Skeleton shape="block" />
        <Skeleton shape="block" />
      </SkeletonGroup>
    );
  } else if (services.isError) {
    body = (
      <Card>
        <EmptyState
          icon="warning"
          title={t('loadError')}
          action={
            <PrimaryButton onClick={() => void services.refetch()} loading={services.isFetching}>
              {t('retry')}
            </PrimaryButton>
          }
        />
      </Card>
    );
  } else if (!service) {
    body = (
      <Card>
        <EmptyState icon="flask" title={t('list.noServices.title')} body={t('form.noService')} />
      </Card>
    );
  } else {
    body = (
      <form
        className="hlb-stack"
        noValidate
        onSubmit={(e) => {
          e.preventDefault();
          onSubmit();
        }}
      >
        <Card className="hlb-svc">
          <span className="hlb-row-art" aria-hidden>
            <Icon name={service.kind === 'checkup' ? 'stetho' : 'flask'} size={22} strokeWidth={1.6} />
          </span>
          <span className="hlb-row-main">
            <span className="hlb-row-title">{service.title}</span>
            <span className="hlb-row-sub">{t('form.partnerLine', { name: service.partner_name })}</span>
          </span>
        </Card>

        <InfoNote icon="shield">{t('form.warn')}</InfoNote>

        <section className="card fld-card flex flex-col gap-3.5">
          {(
            [
              ['fullName', 'user', 'text', NAME_MAX, 'name'],
              ['phone', 'phone', 'tel', 24, 'tel'],
              ['city', 'mapPin', 'text', CITY_MAX, 'address-level2'],
            ] as const
          ).map(([key, icon, type, max, auto]) => (
            <label key={key} className="fld-label">
              <span className="fld-label-t">{t(`form.${key}`)}</span>
              <span className="field">
                <Icon name={icon} size={18} />
                <input
                  type={type}
                  inputMode={type === 'tel' ? 'tel' : undefined}
                  autoComplete={auto}
                  maxLength={max}
                  value={state[key]}
                  aria-invalid={Boolean(errors[key])}
                  aria-describedby={errors[key] ? `hlb-err-${key}` : undefined}
                  onChange={(e) => set(key, e.target.value)}
                  {...pilot(`home-lab-form.field:${key}`, 'text')}
                />
              </span>
              {err(key)}
            </label>
          ))}
          <label className="fld-label">
            <span className="fld-label-t">{t('form.address')}</span>
            <span className="field hlb-area">
              <textarea
                rows={3}
                maxLength={ADDRESS_MAX}
                autoComplete="street-address"
                value={state.address}
                aria-invalid={Boolean(errors.address)}
                aria-describedby={errors.address ? 'hlb-err-address' : undefined}
                onChange={(e) => set('address', e.target.value)}
                {...pilot('home-lab-form.field:address', 'text')}
              />
            </span>
            {err('address')}
          </label>
        </section>

        <section className="card fld-card flex flex-col gap-3.5">
          <div className="flex flex-col gap-2">
            <span className="fld-label-t text-start">{t('form.day')}</span>
            <div role="radiogroup" aria-label={t('form.day')} className="fld-chips hlb-days" {...pilot('home-lab-form.day', 'choice')}>
              {days.map((d) => {
                const v = toApiDate(d);
                return (
                  <button
                    key={v}
                    type="button"
                    role="radio"
                    aria-checked={state.day === v}
                    {...pilotOption(v)}
                    className={clsx('chip', state.day === v && 'on')}
                    onClick={() => set('day', v)}
                  >
                    {formatLongDate(d, locale)}
                  </button>
                );
              })}
            </div>
          </div>
          <div className="flex flex-col gap-2">
            <span className="fld-label-t text-start">{t('form.window')}</span>
            <div role="radiogroup" aria-label={t('form.window')} className="fld-chips" {...pilot('home-lab-form.window', 'choice')}>
              {LAB_WINDOWS.map((w) => (
                <button
                  key={w}
                  type="button"
                  role="radio"
                  aria-checked={state.window === w}
                  {...pilotOption(w)}
                  className={clsx('chip', state.window === w && 'on')}
                  onClick={() => set('window', w)}
                >
                  {t(`form.windows.${w}`)}
                </button>
              ))}
            </div>
          </div>
          <label className="fld-label">
            <span className="fld-label-t">{t('form.note')}</span>
            <span className="field hlb-area">
              <textarea
                rows={2}
                maxLength={NOTE_MAX}
                placeholder={t('form.notePlaceholder')}
                value={state.note}
                onChange={(e) => set('note', e.target.value)}
                {...pilot('home-lab-form.field:note', 'text')}
              />
            </span>
          </label>
        </section>

        <section className="card fld-card flex flex-col gap-3">
          <Checkbox
            checked={state.consentShare}
            onCheckedChange={(v) => set('consentShare', v)}
            pilot={pilot('home-lab-form.consent-share')}
            label={t('form.consentShare', { partner: service.partner_name })}
          />
          {err('consentShare')}
          <Checkbox checked={state.consentImport} onCheckedChange={(v) => set('consentImport', v)} pilot={pilot('home-lab-form.consent-import')} label={t('form.consentImport')} />
          <p className="hlb-note">{t('form.retention', { days: fmt.number(services.data.purge_days) })}</p>
        </section>

        {formError && (
          <p role="alert" className="hlb-err">
            {formError}
          </p>
        )}
        <PrimaryButton type="submit" block loading={create.isPending}>
          {create.isPending ? t('form.submitting') : t('form.submit')}
        </PrimaryButton>
      </form>
    );
  }

  return (
    <div className="view hlb-view">
      <SkyLayer />
      <div className="scroll">
        <ScreenHeader
          title={service ? t('form.title', { service: service.title }) : t('title')}
          subtitle={t('tag')}
          onBack={back}
          backLabel={t('back')}
        />
        <div className="hlb-body">{body}</div>
        <div className="page-tail" />
      </div>
    </div>
  );
}

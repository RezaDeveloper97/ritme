'use client';

import { useLocale, useTranslations } from 'next-intl';
import { useState } from 'react';

import { type JoinApplication, type JoinOffer, useSaveJoinStep } from '@/entities/city-service';
import type { Locale } from '@/shared/i18n';
import { formatNumber } from '@/shared/lib/date';
import { pilot, pilotOption } from '@/shared/pilot';
import { AppSheet } from '@/shared/sheet';
import { ChipGroup, Icon, InfoNote, PillChip, PrimaryButton, SecondaryButton } from '@/shared/ui';

import {
  emptyOffer,
  type OfferForm,
  type OfferProblem,
  offerFromForm,
  offerProblems,
  offerToForm,
  step3Body,
} from '../model/join';
import { useAutosave } from '../model/use-autosave';
import { SaveHint, useErrorText } from './parts';

const MAX_SERVICES = 30;

function ServiceSheet({ initial, onClose, onSave }: { initial: OfferForm; onClose: () => void; onSave: (o: JoinOffer) => void }) {
  const t = useTranslations('cityJoin.services');
  const [form, setForm] = useState(initial);
  const [problems, setProblems] = useState<OfferProblem[]>([]);
  const set = <K extends keyof OfferForm>(key: K, value: OfferForm[K]) => setForm((f) => ({ ...f, [key]: value }));
  const num = (key: 'duration' | 'ageMin' | 'ageMax' | 'capacity' | 'price' | 'sessions' | 'validity', label: string, problem?: OfferProblem) => (
    <label className="fld-label">
      <span className="fld-label-t">{label}</span>
      <span className="field">
        <input
          type="text"
          inputMode="numeric"
          value={form[key]}
          aria-invalid={problem ? problems.includes(problem) : undefined}
          onChange={(e) => set(key, e.target.value)}
          {...pilot(`city-join.service-field:${key}`, 'text')}
        />
      </span>
    </label>
  );
  const submit = () => {
    const found = offerProblems(form);
    setProblems(found);
    if (found.length === 0) onSave(offerFromForm(form));
  };
  return (
    <AppSheet
      open
      onClose={onClose}
      size="full"
      title={t('edit')}
      footer={
        <div className="cjn-row">
          <SecondaryButton block={false} onClick={onClose}>
            {t('cancel')}
          </SecondaryButton>
          <PrimaryButton block={false} onClick={submit} {...pilot('city-join.service-save')}>
            {t('save')}
          </PrimaryButton>
        </div>
      }
    >
      <div className="cjn-stack">
        <ChipGroup label={t('kind')} pilot={pilot('city-join.service-kind', 'choice')}>
          {(['session', 'package'] as const).map((k) => (
            <PillChip key={k} mode="single" pressed={form.kind === k} onPressedChange={() => set('kind', k)} {...pilotOption(k)}>
              {t(`kinds.${k}`)}
            </PillChip>
          ))}
        </ChipGroup>
        <label className="fld-label">
          <span className="fld-label-t">{t('name')}</span>
          <span className="field">
            <input type="text" maxLength={160} value={form.name} placeholder={t('namePh')} aria-invalid={problems.includes('name')} onChange={(e) => set('name', e.target.value)} {...pilot('city-join.service-name', 'text')} />
          </span>
        </label>
        <label className="fld-label">
          <span className="fld-label-t">{t('desc')}</span>
          <span className="field cjn-area">
            <textarea rows={2} maxLength={300} value={form.description} onChange={(e) => set('description', e.target.value)} {...pilot('city-join.service-desc', 'text')} />
          </span>
        </label>
        <div className="cjn-group">
          <span className="fld-label-t">{t('format')}</span>
          <ChipGroup label={t('format')} pilot={pilot('city-join.service-format', 'choice')}>
            {(['group', 'private'] as const).map((f) => (
              <PillChip key={f} mode="single" pressed={form.format === f} onPressedChange={() => set('format', form.format === f ? null : f)} {...pilotOption(f)}>
                {t(`formats.${f}`)}
              </PillChip>
            ))}
          </ChipGroup>
        </div>
        <div className="cjn-row">
          {num('duration', t('duration'), 'duration')}
          {num('capacity', t('capacity'), 'capacity')}
        </div>
        <div className="cjn-row">
          {num('ageMin', t('ageMin'), 'age')}
          {num('ageMax', t('ageMax'), 'age')}
        </div>
        {num('price', t('price'))}
        {form.kind === 'package' ? (
          <div className="cjn-row">
            {num('sessions', t('sessions'), 'sessions')}
            {num('validity', t('validity'), 'validity')}
          </div>
        ) : null}
        {problems.map((p) => (
          <p key={p} role="alert" className="cjn-err">
            {t(`errors.${p}`)}
          </p>
        ))}
      </div>
    </AppSheet>
  );
}

function OfferSummary({ o }: { o: JoinOffer }) {
  const t = useTranslations('cityJoin.services');
  const locale = useLocale() as Locale;
  const n = (v: number) => formatNumber(v, locale);
  const min = o.age_min_months ?? null;
  const max = o.age_max_months ?? null;
  const parts = [
    o.format ? t(`formats.${o.format}`) : null,
    o.duration_minutes ? t('minutes', { n: n(o.duration_minutes) }) : null,
    min !== null && max !== null ? t('agesFromTo', { min: n(min), max: n(max) }) : min !== null ? t('agesFrom', { min: n(min) }) : max !== null ? t('agesTo', { max: n(max) }) : null,
    o.capacity ? t('capacityOf', { n: n(o.capacity) }) : null,
    o.kind === 'package' && o.package_sessions ? t('package', { n: n(o.package_sessions), days: n(o.package_validity_days ?? 0) }) : null,
  ].filter(Boolean);
  return (
    <>
      <span className="cjn-offer-name">{o.name}</span>
      {parts.length > 0 ? <span className="cjn-offer-sub">{parts.join(' · ')}</span> : null}
      {o.price_toman != null ? <span className="cjn-offer-price">{t('price_toman', { price: n(o.price_toman) })}</span> : null}
    </>
  );
}

/** Step 3 «خدمات» (nbl_Dir_JoinDocs, top): the list of services with prices; each is added / edited in a sheet. */
export function StepServices({ app, onNext, onSaveError }: { app: JoinApplication; onNext: () => void; onSaveError: (e: unknown) => void }) {
  const t = useTranslations('cityJoin');
  const errorText = useErrorText();
  const save = useSaveJoinStep(app.id);
  const [services, setServices] = useState<JoinOffer[]>(app.services);
  const [editing, setEditing] = useState<{ index: number | null; form: OfferForm } | null>(null);
  const { state, flush } = useAutosave(services, (v) =>
    save.mutateAsync({ step: 3, body: step3Body(v) }).catch((e) => {
      onSaveError(e);
      throw e;
    }),
  );

  const next = async () => {
    if (await flush()) onNext();
  };

  return (
    <div className="cjn-stack">
      <div>
        <h2 className="cjn-h">{t('services.title')}</h2>
        <p className="cjn-hint">{t('services.lead')}</p>
      </div>

      {services.length === 0 ? (
        <InfoNote icon="info">{t('services.empty')}</InfoNote>
      ) : (
        <ul className="cjn-offers">
          {services.map((o, i) => (
            <li key={`${o.name}-${i}`} className="card cjn-offer">
              <button type="button" className="cjn-offer-main" aria-label={t('services.editLabel', { name: o.name })} onClick={() => setEditing({ index: i, form: offerToForm(o) })}>
                <OfferSummary o={o} />
              </button>
              <button
                type="button"
                className="cjn-x is-static"
                aria-label={t('services.removeLabel', { name: o.name })}
                onClick={() => setServices((list) => list.filter((_, j) => j !== i))}
              >
                <Icon name="trash" size={16} />
              </button>
            </li>
          ))}
        </ul>
      )}

      <SecondaryButton icon="plus" disabled={services.length >= MAX_SERVICES} onClick={() => setEditing({ index: null, form: emptyOffer() })} {...pilot('city-join.service-add')}>
        {t('services.add')}
      </SecondaryButton>

      {editing ? (
        <ServiceSheet
          initial={editing.form}
          onClose={() => setEditing(null)}
          onSave={(o) => {
            setServices((list) => (editing.index === null ? [...list, o] : list.map((x, j) => (j === editing.index ? o : x))));
            setEditing(null);
          }}
        />
      ) : null}

      <SaveHint state={state} />
      {save.isError ? (
        <p className="cjn-err" role="alert">
          {errorText(save.error)}
        </p>
      ) : null}
      <PrimaryButton onClick={() => void next()} {...pilot('city-join.next:services')}>{t('next')}</PrimaryButton>
    </div>
  );
}
